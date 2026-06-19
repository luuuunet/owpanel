package autops

import (
	"regexp"
)

var webStackKeyPattern = regexp.MustCompile(`(?i)^(nginx|openresty|apache|caddy|mysql|mariadb|postgresql|redis|php|memcached)`)

type SitePresetResult struct {
	ConfigUpdated bool     `json:"config_updated"`
	WatchKeys     []string `json:"watch_keys"`
	WatchCount    int      `json:"watch_count"`
}

// ApplySitePreset enables auto-ops protection defaults and bulk watch for common web stack services.
func (s *Service) ApplySitePreset() (*SitePresetResult, error) {
	cfg := s.loadConfig()
	cfg.Enabled = true
	cfg.SSLAutoRenew = true
	cfg.WebsiteScanEnabled = true
	cfg.AlertDaysSSL = 14
	cfg.AlertDaysSite = 14
	if err := s.UpdateConfig(cfg); err != nil {
		return nil, err
	}
	st, err := s.GetStatus()
	if err != nil {
		return nil, err
	}
	var keys []string
	for _, w := range st.Watches {
		if webStackKeyPattern.MatchString(w.Key) {
			keys = append(keys, w.Key)
		}
	}
	if len(keys) > 0 {
		if err := s.BulkUpdateWatch(keys, true, true); err != nil {
			return nil, err
		}
	}
	return &SitePresetResult{
		ConfigUpdated: true,
		WatchKeys:     keys,
		WatchCount:    len(keys),
	}, nil
}

// ApplyOpsPreset enables resource alerts and memory relief for daily operations.
func (s *Service) ApplyOpsPreset() error {
	cfg := s.loadConfig()
	cfg.Enabled = true
	cfg.ResourceEnabled = true
	cfg.CPUThreshold = 85
	cfg.MemThreshold = 85
	cfg.DiskThreshold = 90
	cfg.MemAutoRelief = true
	cfg.NotifyOnDown = true
	cfg.NotifyOnFail = true
	cfg.SSLAutoRenew = true
	cfg.WebsiteScanEnabled = true
	return s.UpdateConfig(cfg)
}
