package toolbox

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/shirou/gopsutil/v3/cpu"
)

type SystemOverview struct {
	Hostname    string      `json:"hostname"`
	OS          string      `json:"os"`
	Platform    string      `json:"platform"`
	Uptime      uint64      `json:"uptime"`
	UptimeHuman string      `json:"uptime_human"`
	Load1       float64     `json:"load1"`
	Load5       float64     `json:"load5"`
	Load15      float64     `json:"load15"`
	CPUCount    int         `json:"cpu_count"`
	Memory      MemBrief    `json:"memory"`
	Swap        MemBrief    `json:"swap"`
	Disks       []DiskBrief `json:"disks"`
}

type MemBrief struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskBrief struct {
	Mount       string  `json:"mount"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type ListeningPort struct {
	Protocol  string `json:"protocol"`
	Address   string `json:"address"`
	Port      uint32 `json:"port"`
	PID       int32  `json:"pid"`
	Process   string `json:"process"`
	User      string `json:"user"`
	Command   string `json:"command,omitempty"`
	Firewalled bool  `json:"firewalled,omitempty"`
}

type ProcessTop struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	User    string  `json:"user"`
	CPU     float64 `json:"cpu"`
	Memory  float32 `json:"memory"`
	Command string  `json:"command"`
}

func (s *Service) SystemOverview() (*SystemOverview, error) {
	info, err := host.Info()
	if err != nil {
		return nil, err
	}
	vm, _ := mem.VirtualMemory()
	sw, _ := mem.SwapMemory()
	ld, _ := load.Avg()
	cores, _ := cpu.Counts(true)

	out := &SystemOverview{
		Hostname:    info.Hostname,
		OS:          info.OS,
		Platform:    info.Platform,
		Uptime:      info.Uptime,
		UptimeHuman: formatUptime(info.Uptime),
		CPUCount:    cores,
	}
	if ld != nil {
		out.Load1, out.Load5, out.Load15 = ld.Load1, ld.Load5, ld.Load15
	}
	if vm != nil {
		out.Memory = MemBrief{Total: vm.Total, Used: vm.Used, Free: vm.Available, UsedPercent: vm.UsedPercent}
	}
	if sw != nil {
		out.Swap = MemBrief{Total: sw.Total, Used: sw.Used, Free: sw.Free, UsedPercent: sw.UsedPercent}
	}

	partitions, _ := disk.Partitions(false)
	for _, p := range partitions {
		if strings.HasPrefix(p.Mountpoint, "/snap") || strings.HasPrefix(p.Mountpoint, "/boot/efi") {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil || usage.Total == 0 {
			continue
		}
		out.Disks = append(out.Disks, DiskBrief{
			Mount: p.Mountpoint, Total: usage.Total, Used: usage.Used,
			Free: usage.Free, UsedPercent: usage.UsedPercent,
		})
	}
	sort.Slice(out.Disks, func(i, j int) bool { return out.Disks[i].UsedPercent > out.Disks[j].UsedPercent })
	return out, nil
}

func (s *Service) ListeningPorts() ([]ListeningPort, error) {
	kinds := []string{"tcp", "udp", "tcp6", "udp6"}
	seen := map[string]ListeningPort{}
	for _, kind := range kinds {
		conns, err := gnet.Connections(kind)
		if err != nil {
			continue
		}
		for _, c := range conns {
			if c.Status != "LISTEN" && !strings.HasPrefix(kind, "udp") {
				continue
			}
			if c.Laddr.Port == 0 {
				continue
			}
			proto := strings.TrimSuffix(kind, "6")
			addr := c.Laddr.IP
			if addr == "" || addr == "0.0.0.0" || addr == "::" || addr == "*" {
				addr = "*"
			}
			key := fmt.Sprintf("%s:%d:%s", proto, c.Laddr.Port, addr)
			if _, ok := seen[key]; ok {
				continue
			}
			lp := ListeningPort{
				Protocol: proto,
				Address:  addr,
				Port:     c.Laddr.Port,
				PID:      c.Pid,
			}
			if c.Pid > 0 {
				if p, err := process.NewProcess(c.Pid); err == nil {
					name, _ := p.Name()
					lp.Process = name
					lp.User, _ = p.Username()
					cmd, _ := p.Cmdline()
					if len(cmd) > 120 {
						cmd = cmd[:120] + "…"
					}
					lp.Command = cmd
				}
			}
			seen[key] = lp
		}
	}
	out := make([]ListeningPort, 0, len(seen))
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Port != out[j].Port {
			return out[i].Port < out[j].Port
		}
		return out[i].Protocol < out[j].Protocol
	})
	return out, nil
}

func (s *Service) TopProcesses(limit int) ([]ProcessTop, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 15
	}
	out := make([]ProcessTop, 0, limit*2)
	for _, p := range procs {
		name, _ := p.Name()
		user, _ := p.Username()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		cmd, _ := p.Cmdline()
		if len(cmd) > 160 {
			cmd = cmd[:160] + "…"
		}
		out = append(out, ProcessTop{
			PID: p.Pid, Name: name, User: user, CPU: cpu, Memory: mem, Command: cmd,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Memory != out[j].Memory {
			return out[i].Memory > out[j].Memory
		}
		return out[i].CPU > out[j].CPU
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *Service) DropCaches() (*DropCacheResult, error) {
	result := &DropCacheResult{}
	if runtime.GOOS != "linux" {
		result.Supported = false
		result.Message = "仅 Linux 服务器支持清理页面缓存"
		return result, nil
	}
	vmBefore, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	result.Supported = true
	result.BeforePct = vmBefore.UsedPercent
	beforeAvail := vmBefore.Available
	_ = exec.Command("sync").Run()
	if err := exec.Command("sh", "-c", "echo 3 > /proc/sys/vm/drop_caches").Run(); err != nil {
		return nil, fmt.Errorf("清理缓存失败（需要 root 权限）: %w", err)
	}
	time.Sleep(300 * time.Millisecond)
	vmAfter, _ := mem.VirtualMemory()
	if vmAfter != nil {
		result.AfterPct = vmAfter.UsedPercent
		if vmAfter.Available > beforeAvail {
			result.FreedBytes = vmAfter.Available - beforeAvail
		}
	}
	result.Message = "已清理页面缓存"
	return result, nil
}

type DropCacheResult struct {
	Supported  bool    `json:"supported"`
	FreedBytes uint64  `json:"freed_bytes"`
	BeforePct  float64 `json:"before_pct"`
	AfterPct   float64 `json:"after_pct"`
	Message    string  `json:"message"`
}

func formatUptime(sec uint64) string {
	d := sec / 86400
	h := (sec % 86400) / 3600
	m := (sec % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
