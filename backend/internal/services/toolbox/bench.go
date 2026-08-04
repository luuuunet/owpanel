package toolbox

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type BenchKind string

const (
	BenchCPU     BenchKind = "cpu"
	BenchMemory  BenchKind = "memory"
	BenchDisk    BenchKind = "disk"
	BenchNetwork BenchKind = "network"
)

type BenchResult struct {
	Kind        string            `json:"kind"`
	Score       float64           `json:"score"`
	Unit        string            `json:"unit"`
	DurationMs  int64             `json:"duration_ms"`
	Detail      string            `json:"detail"`
	Metrics     map[string]any    `json:"metrics,omitempty"`
	StartedAt   time.Time         `json:"started_at"`
	FinishedAt  time.Time         `json:"finished_at"`
}

var (
	benchMu   sync.Mutex
	benchBusy atomic.Bool
)

func (s *Service) RunBench(kind string) (*BenchResult, error) {
	k := BenchKind(kind)
	switch k {
	case BenchCPU, BenchMemory, BenchDisk, BenchNetwork:
	default:
		return nil, fmt.Errorf("未知测试类型: %s", kind)
	}
	if !benchBusy.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("已有性能测试在运行，请稍后再试")
	}
	defer benchBusy.Store(false)

	benchMu.Lock()
	defer benchMu.Unlock()

	start := time.Now()
	var (
		res *BenchResult
		err error
	)
	switch k {
	case BenchCPU:
		res, err = benchCPU()
	case BenchMemory:
		res, err = benchMemory()
	case BenchDisk:
		res, err = benchDisk()
	case BenchNetwork:
		res, err = benchNetwork()
	}
	if err != nil {
		return nil, err
	}
	res.Kind = string(k)
	res.StartedAt = start
	res.FinishedAt = time.Now()
	res.DurationMs = res.FinishedAt.Sub(start).Milliseconds()
	return res, nil
}

func benchCPU() (*BenchResult, error) {
	cores := runtime.NumCPU()
	if cores < 1 {
		cores = 1
	}
	duration := 3 * time.Second
	var ops atomic.Uint64
	ctx, cancel := context.WithTimeout(context.Background(), duration+500*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < cores; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, 4096)
			_, _ = rand.Read(buf)
			h := sha256.New()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					h.Reset()
					h.Write(buf)
					_ = h.Sum(nil)
					ops.Add(1)
				}
			}
		}()
	}
	time.Sleep(duration)
	cancel()
	wg.Wait()

	total := ops.Load()
	opsPerSec := float64(total) / duration.Seconds()
	// Normalize to a rough single-core MHash/s-like score (ops of 4KB SHA256).
	score := opsPerSec / 1000
	return &BenchResult{
		Score:  round2(score),
		Unit:   "K ops/s",
		Detail: fmt.Sprintf("%d 核心并行 SHA-256，%.0f ops/s", cores, opsPerSec),
		Metrics: map[string]any{
			"cores":       cores,
			"ops_total":   total,
			"ops_per_sec": round2(opsPerSec),
			"seconds":     duration.Seconds(),
		},
	}, nil
}

func benchMemory() (*BenchResult, error) {
	size := 64 << 20 // 64 MiB
	if runtime.GOOS == "windows" {
		size = 32 << 20
	}
	buf := make([]byte, size)
	pattern := make([]byte, 4096)
	_, _ = rand.Read(pattern)

	start := time.Now()
	for off := 0; off < size; off += len(pattern) {
		copy(buf[off:], pattern)
	}
	writeDur := time.Since(start)

	start = time.Now()
	var checksum byte
	for i := 0; i < size; i += 64 {
		checksum ^= buf[i]
	}
	readDur := time.Since(start)
	_ = checksum

	writeMBps := float64(size) / writeDur.Seconds() / (1 << 20)
	readMBps := float64(size) / readDur.Seconds() / (1 << 20)
	score := (writeMBps + readMBps) / 2
	return &BenchResult{
		Score:  round2(score),
		Unit:   "MB/s",
		Detail: fmt.Sprintf("写入 %.1f MB/s · 读取 %.1f MB/s（%d MB 缓冲）", writeMBps, readMBps, size>>20),
		Metrics: map[string]any{
			"buffer_mb":    size >> 20,
			"write_mb_s":   round2(writeMBps),
			"read_mb_s":    round2(readMBps),
			"write_ms":     writeDur.Milliseconds(),
			"read_ms":      readDur.Milliseconds(),
		},
	}, nil
}

func benchDisk() (*BenchResult, error) {
	dir := os.TempDir()
	_ = os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, fmt.Sprintf("owpanel-bench-%d.bin", time.Now().UnixNano()))
	defer os.Remove(path)

	size := int64(64 << 20) // 64 MiB
	if runtime.GOOS == "windows" {
		size = 32 << 20
	}
	chunk := make([]byte, 1<<20)
	_, _ = rand.Read(chunk)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("创建测试文件失败: %w", err)
	}
	start := time.Now()
	var written int64
	for written < size {
		n, err := f.Write(chunk)
		written += int64(n)
		if err != nil {
			f.Close()
			return nil, err
		}
	}
	_ = f.Sync()
	f.Close()
	writeDur := time.Since(start)

	rf, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	start = time.Now()
	readBuf := make([]byte, 1<<20)
	var readTotal int64
	for {
		n, err := rf.Read(readBuf)
		readTotal += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			rf.Close()
			return nil, err
		}
	}
	rf.Close()
	readDur := time.Since(start)

	writeMBps := float64(written) / writeDur.Seconds() / (1 << 20)
	readMBps := float64(readTotal) / readDur.Seconds() / (1 << 20)
	score := (writeMBps + readMBps) / 2
	return &BenchResult{
		Score:  round2(score),
		Unit:   "MB/s",
		Detail: fmt.Sprintf("顺序写 %.1f MB/s · 顺序读 %.1f MB/s（%d MB）", writeMBps, readMBps, written>>20),
		Metrics: map[string]any{
			"path":       path,
			"size_mb":    written >> 20,
			"write_mb_s": round2(writeMBps),
			"read_mb_s":  round2(readMBps),
			"write_ms":   writeDur.Milliseconds(),
			"read_ms":    readDur.Milliseconds(),
		},
	}, nil
}

func benchNetwork() (*BenchResult, error) {
	// Prefer CacheFly / Cloudflare style fixed-size downloads.
	urls := []struct {
		URL  string
		Hint string
	}{
		{"https://speed.cloudflare.com/__down?bytes=10000000", "Cloudflare 10MB"},
		{"https://cachefly.cachefly.net/10mb.test", "CacheFly 10MB"},
	}
	client := &http.Client{
		Timeout: 45 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	var lastErr error
	for _, u := range urls {
		req, err := http.NewRequest(http.MethodGet, u.URL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "OWPanel-SpeedTest/1.0")
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, u.Hint)
			continue
		}
		n, err := io.Copy(io.Discard, io.LimitReader(resp.Body, 20<<20))
		resp.Body.Close()
		if err != nil && err != io.EOF {
			lastErr = err
			continue
		}
		if n < 1<<20 {
			lastErr = fmt.Errorf("%s 下载字节过少", u.Hint)
			continue
		}
		dur := time.Since(start)
		mbps := float64(n*8) / dur.Seconds() / 1e6
		MBps := float64(n) / dur.Seconds() / (1 << 20)
		return &BenchResult{
			Score:  round2(mbps),
			Unit:   "Mbps",
			Detail: fmt.Sprintf("%s 下载 %.1f MB，用时 %.1fs（%.1f MB/s）", u.Hint, float64(n)/(1<<20), dur.Seconds(), MBps),
			Metrics: map[string]any{
				"source":     u.Hint,
				"bytes":      n,
				"seconds":    round2(dur.Seconds()),
				"mbps":       round2(mbps),
				"mb_per_sec": round2(MBps),
			},
		}, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("所有测速源均失败")
	}
	return nil, lastErr
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
