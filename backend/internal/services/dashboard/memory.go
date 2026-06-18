package dashboard

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
)

type FreeMemoryResult struct {
	Supported  bool    `json:"supported"`
	Method     string  `json:"method,omitempty"`
	FreedBytes uint64  `json:"freed_bytes"`
	BeforeUsed uint64  `json:"before_used"`
	AfterUsed  uint64  `json:"after_used"`
	BeforePct  float64 `json:"before_pct"`
	AfterPct   float64 `json:"after_pct"`
	Message    string  `json:"message,omitempty"`
}

func (s *Service) FreeMemory() (*FreeMemoryResult, error) {
	vmBefore, err := mem.VirtualMemory()
	if err != nil || vmBefore == nil {
		return nil, fmt.Errorf("无法读取内存信息")
	}
	result := &FreeMemoryResult{
		BeforeUsed: vmBefore.Used,
		BeforePct:  vmBefore.UsedPercent,
	}

	switch runtime.GOOS {
	case "linux":
		result.Supported = true
		result.Method = "drop_caches"
		beforeAvail := vmBefore.Available
		_ = exec.Command("sync").Run()
		if err := os.WriteFile("/proc/sys/vm/drop_caches", []byte("3\n"), 0); err != nil {
			return nil, fmt.Errorf("释放内存失败: %w（需要 root 权限）", err)
		}
		time.Sleep(300 * time.Millisecond)
		vmAfter, err := mem.VirtualMemory()
		if err == nil && vmAfter != nil {
			result.AfterUsed = vmAfter.Used
			result.AfterPct = vmAfter.UsedPercent
			if vmAfter.Available > beforeAvail {
				result.FreedBytes = vmAfter.Available - beforeAvail
			}
		}
		if result.FreedBytes == 0 {
			result.Message = "已清理页面缓存（当前可用内存无明显变化）"
		}
	default:
		result.Supported = false
		result.Message = "当前系统暂不支持一键释放内存（仅 Linux 服务器可用）"
	}
	return result, nil
}
