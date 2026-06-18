package process

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

var (
	ErrInvalidPID    = errors.New("invalid pid")
	ErrProtectedPID  = errors.New("protected process")
	ErrProcessGone   = errors.New("process not found")
)

type Info struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Memory  float32 `json:"memory"`
	Status  string  `json:"status"`
	Command string  `json:"command"`
}

type Service struct{}

func NewService() *Service { return &Service{} }

const listCacheTTL = 45 * time.Second

var (
	listCacheMu sync.Mutex
	listCache   []Info
	listCacheAt time.Time
)

func (s *Service) List() ([]Info, error) {
	listCacheMu.Lock()
	if len(listCache) > 0 && time.Since(listCacheAt) < listCacheTTL {
		out := make([]Info, len(listCache))
		copy(out, listCache)
		listCacheMu.Unlock()
		return out, nil
	}
	listCacheMu.Unlock()

	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}

	result := make([]Info, 0, len(procs))
	for _, p := range procs {
		name, _ := p.Name()
		user := ""
		if u, err := p.Username(); err == nil {
			user = u
		}
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		status, _ := p.Status()
		cmd, _ := p.Cmdline()
		if len(cmd) > 200 {
			cmd = cmd[:200]
		}
		st := ""
		if len(status) > 0 {
			st = status[0]
		}
		result = append(result, Info{
			PID: p.Pid, Name: name, User: user,
			CPU: cpu, Memory: mem, Status: st, Command: cmd,
		})
	}

	listCacheMu.Lock()
	listCache = result
	listCacheAt = time.Now()
	listCacheMu.Unlock()
	return result, nil
}

func (s *Service) TopByCPU(limit int) ([]Info, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CPU > all[j].CPU
	})
	if limit <= 0 {
		limit = 10
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (s *Service) TopByMemory(limit int) ([]Info, error) {
	all, err := s.List()
	if err != nil {
		return nil, err
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Memory > all[j].Memory
	})
	if limit <= 0 {
		limit = 10
	}
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func (s *Service) InvalidateCache() {
	listCacheMu.Lock()
	listCache = nil
	listCacheAt = time.Time{}
	listCacheMu.Unlock()
}

func (s *Service) Kill(pid int32) error {
	if pid <= 1 {
		return ErrProtectedPID
	}
	if int(pid) == os.Getpid() {
		return fmt.Errorf("%w: panel", ErrProtectedPID)
	}
	p, err := process.NewProcess(pid)
	if err != nil {
		return ErrProcessGone
	}
	running, err := p.IsRunning()
	if err != nil || !running {
		return ErrProcessGone
	}
	if err := p.Kill(); err != nil {
		return err
	}
	s.InvalidateCache()
	return nil
}
