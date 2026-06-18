package dashboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type OptimizeStep struct {
	Key     string `json:"key"`
	Title   string `json:"title"`
	Status  string `json:"status"` // skipped | success | failed | partial
	Message string `json:"message"`
}

type OptimizeResult struct {
	StartedAt   string         `json:"started_at"`
	FinishedAt  string         `json:"finished_at"`
	BeforeLoad  float64        `json:"before_load"`
	AfterLoad   float64        `json:"after_load"`
	BeforeMem   float64        `json:"before_mem_pct"`
	AfterMem    float64        `json:"after_mem_pct"`
	Steps       []OptimizeStep `json:"steps"`
	Summary     string         `json:"summary"`
	Improved    bool           `json:"improved"`
}

func (s *Service) OneClickOptimize() (*OptimizeResult, error) {
	start := time.Now()
	stats, err := s.GetStats()
	if err != nil || stats == nil {
		return nil, fmt.Errorf("无法读取系统状态")
	}

	beforeLoad := stats.Load.Load1
	beforeMem := stats.Memory.UsedPercent
	cores := stats.CPU.Cores
	if cores <= 0 {
		cores = 1
	}
	loadPct := beforeLoad / float64(cores) * 100

	result := &OptimizeResult{
		StartedAt:  start.Format(time.RFC3339),
		BeforeLoad: round2(beforeLoad),
		BeforeMem:  round2(beforeMem),
		Steps:      []OptimizeStep{},
	}

	add := func(key, title, status, msg string) {
		result.Steps = append(result.Steps, OptimizeStep{Key: key, Title: title, Status: status, Message: msg})
	}

	if runtime.GOOS != "linux" {
		add("platform", "系统优化", "skipped", "仅 Linux 服务器支持一键优化")
		result.Summary = "当前系统不支持自动优化"
		result.FinishedAt = time.Now().Format(time.RFC3339)
		return result, nil
	}

	maxDiskPct := maxDiskUsed(stats.Disk)
	swapUsedMB := stats.Swap.Used / 1024 / 1024

	// 1. 释放页面缓存
	if beforeMem >= 65 || loadPct >= 70 {
		memRes, err := s.FreeMemory()
		if err != nil {
			add("drop_cache", "释放页面缓存", "failed", err.Error())
		} else if memRes != nil && memRes.Supported {
			msg := memRes.Message
			if msg == "" && memRes.FreedBytes > 0 {
				msg = fmt.Sprintf("释放约 %s 内存", formatBytesShort(memRes.FreedBytes))
			}
			if msg == "" {
				msg = "已清理页面缓存"
			}
			add("drop_cache", "释放页面缓存", "success", msg)
		} else {
			add("drop_cache", "释放页面缓存", "skipped", "不支持或无需执行")
		}
	} else {
		add("drop_cache", "释放页面缓存", "skipped", fmt.Sprintf("内存使用率 %.0f%%，暂不需要", beforeMem))
	}

	// 2. Swap 刷新
	if stats.Swap.Total > 0 && swapUsedMB >= 256 {
		out, err := runShell("sync; echo 3 > /proc/sys/vm/drop_caches 2>/dev/null; swapoff -a 2>/dev/null && swapon -a 2>/dev/null; echo OK")
		if err != nil {
			add("swap", "刷新 Swap", "failed", trimOut(out, err))
		} else {
			add("swap", "刷新 Swap", "success", fmt.Sprintf("Swap 已刷新（此前占用约 %d MB）", swapUsedMB))
		}
	} else {
		add("swap", "刷新 Swap", "skipped", "Swap 占用较低或未启用")
	}

	// 3. Docker 清理
	if beforeMem >= 70 || maxDiskPct >= 80 {
		if _, err := exec.LookPath("docker"); err == nil {
			out, err := runShell("docker system prune -f 2>&1")
			if err != nil {
				add("docker_prune", "Docker 资源清理", "partial", trimOut(out, err))
			} else {
				msg := strings.TrimSpace(out)
				if msg == "" {
					msg = "已清理未使用的镜像/容器/网络"
				} else if len(msg) > 200 {
					msg = msg[:200] + "…"
				}
				add("docker_prune", "Docker 资源清理", "success", msg)
			}
		} else {
			add("docker_prune", "Docker 资源清理", "skipped", "未安装 Docker")
		}
	} else {
		add("docker_prune", "Docker 资源清理", "skipped", "负载与磁盘正常，跳过")
	}

	// 4. 系统日志压缩
	if maxDiskPct >= 85 {
		out, err := runShell("journalctl --vacuum-time=3d 2>&1 || true")
		add("journal", "压缩系统日志", "success", firstLine(out, "已清理 3 天前的 journal 日志"))
		_ = err
	} else {
		add("journal", "压缩系统日志", "skipped", fmt.Sprintf("磁盘使用率 %.0f%%", maxDiskPct))
	}

	// 5. 包管理器缓存
	if maxDiskPct >= 80 {
		out, err := runShell("apt-get clean 2>/dev/null || yum clean all 2>/dev/null || dnf clean all 2>/dev/null || true")
		if err != nil {
			add("pkg_cache", "清理软件包缓存", "partial", trimOut(out, err))
		} else {
			add("pkg_cache", "清理软件包缓存", "success", "已清理 apt/yum/dnf 缓存")
		}
	} else {
		add("pkg_cache", "清理软件包缓存", "skipped", "磁盘空间充足")
	}

	time.Sleep(400 * time.Millisecond)
	afterStats := s.currentStats(false)
	if afterStats != nil {
		result.AfterLoad = round2(afterStats.Load.Load1)
		result.AfterMem = round2(afterStats.Memory.UsedPercent)
	}
	result.FinishedAt = time.Now().Format(time.RFC3339)

	loadDown := result.AfterLoad < beforeLoad
	memDown := result.AfterMem < beforeMem
	result.Improved = loadDown || memDown
	if result.Improved {
		parts := []string{}
		if memDown {
			parts = append(parts, fmt.Sprintf("内存 %.0f%% → %.0f%%", beforeMem, result.AfterMem))
		}
		if loadDown {
			parts = append(parts, fmt.Sprintf("负载 %.2f → %.2f", beforeLoad, result.AfterLoad))
		}
		result.Summary = "优化完成：" + strings.Join(parts, "，")
	} else {
		result.Summary = "优化完成，系统状态已检查并执行必要清理"
	}
	return result, nil
}

func maxDiskUsed(disks []DiskStats) float64 {
	max := 0.0
	for _, d := range disks {
		if d.UsedPercent > max {
			max = d.UsedPercent
		}
	}
	return max
}

func runShell(script string) (string, error) {
	cmd := exec.Command("sh", "-c", script)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func trimOut(out string, err error) string {
	msg := strings.TrimSpace(out)
	if err != nil {
		if msg != "" {
			return msg
		}
		return err.Error()
	}
	return msg
}

func firstLine(out, fallback string) string {
	out = strings.TrimSpace(out)
	if out == "" {
		return fallback
	}
	if i := strings.IndexByte(out, '\n'); i >= 0 {
		return strings.TrimSpace(out[:i])
	}
	return out
}

func formatBytesShort(b uint64) string {
	if b >= 1024*1024*1024 {
		return fmt.Sprintf("%.1f GB", float64(b)/1024/1024/1024)
	}
	if b >= 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(b)/1024/1024)
	}
	return fmt.Sprintf("%d KB", b/1024)
}
