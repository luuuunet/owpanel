package dashboard

import (
	"fmt"
	"strconv"
)

type AlertThresholds struct {
	CPU  int `json:"cpu"`
	Mem  int `json:"mem"`
	Disk int `json:"disk"`
}

type ResourceAlert struct {
	Type      string  `json:"type"`
	Level     string  `json:"level"`
	Value     float64 `json:"value"`
	Threshold int     `json:"threshold"`
	Message   string  `json:"message"`
}

func DefaultAlertThresholds() AlertThresholds {
	return AlertThresholds{CPU: 85, Mem: 85, Disk: 90}
}

func ParseAlertThresholds(settings map[string]string) AlertThresholds {
	th := DefaultAlertThresholds()
	if settings == nil {
		return th
	}
	if v, ok := parseThresholdSetting(settings["auto_ops_cpu_threshold"]); ok {
		th.CPU = v
	}
	if v, ok := parseThresholdSetting(settings["auto_ops_mem_threshold"]); ok {
		th.Mem = v
	}
	if v, ok := parseThresholdSetting(settings["auto_ops_disk_threshold"]); ok {
		th.Disk = v
	}
	return th
}

func parseThresholdSetting(raw string) (int, bool) {
	v, err := strconv.Atoi(raw)
	if err != nil || v < 50 || v > 100 {
		return 0, false
	}
	return v, true
}

func (s *Service) ComputeResourceAlerts(th AlertThresholds) []ResourceAlert {
	cur := s.currentStats(false)
	if cur == nil {
		return nil
	}
	var alerts []ResourceAlert
	if a := alertForMetric("cpu", cur.CPU.UsagePercent, th.CPU); a != nil {
		alerts = append(alerts, *a)
	}
	if a := alertForMetric("memory", cur.Memory.UsedPercent, th.Mem); a != nil {
		alerts = append(alerts, *a)
	}
	maxDisk := maxDiskPercent(cur.Disk)
	if a := alertForMetric("disk", maxDisk, th.Disk); a != nil {
		alerts = append(alerts, *a)
	}
	return alerts
}

func maxDiskPercent(disks []DiskStats) float64 {
	var max float64
	for _, d := range disks {
		if d.UsedPercent > max {
			max = d.UsedPercent
		}
	}
	return max
}

func alertForMetric(kind string, value float64, threshold int) *ResourceAlert {
	if threshold <= 0 {
		return nil
	}
	warnAt := float64(threshold) * 0.88
	if value < warnAt {
		return nil
	}
	level := "warning"
	if value >= float64(threshold) {
		level = "critical"
	}
	label := kind
	switch kind {
	case "cpu":
		label = "CPU"
	case "memory":
		label = "Memory"
	case "disk":
		label = "Disk"
	}
	return &ResourceAlert{
		Type:      kind,
		Level:     level,
		Value:     round2(value),
		Threshold: threshold,
		Message:   fmt.Sprintf("%s %.1f%% (threshold %d%%)", label, value, threshold),
	}
}
