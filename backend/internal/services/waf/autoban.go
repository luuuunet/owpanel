package waf

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/luuuunet/owpanel/internal/models"
	"github.com/luuuunet/owpanel/internal/services/settings"
)

var (
	logIPRe     = regexp.MustCompile(`^(\d{1,3}(?:\.\d{1,3}){3})\s`)
	logStatusRe = regexp.MustCompile(`"\s(\d{3})\s`)
	autoBanMu   sync.Mutex
	lastAutoBan struct {
		LastRun time.Time        `json:"last_run"`
		Message string           `json:"message"`
		Banned  []string         `json:"banned"`
	}
)

func (s *Service) StartAutoBan() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cfg, err := s.GetConfig()
			if err != nil || cfg == nil || !cfg.AutoBanEnabled {
				continue
			}
			_, _ = s.RunAutoBan()
		}
	}()
}

func (s *Service) AutoBanStatus() map[string]interface{} {
	autoBanMu.Lock()
	defer autoBanMu.Unlock()
	return map[string]interface{}{
		"last_run": lastAutoBan.LastRun,
		"message":  lastAutoBan.Message,
		"banned":   lastAutoBan.Banned,
	}
}

func (s *Service) RunAutoBan() (map[string]interface{}, error) {
	autoBanMu.Lock()
	defer autoBanMu.Unlock()

	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	threshold := cfg.AutoBanThreshold
	if threshold <= 0 {
		threshold = 20
	}
	window := cfg.AutoBanWindowSec
	if window <= 0 {
		window = 300
	}
	hits, scanned, err := s.scanSecurityHits(cfg, time.Duration(window)*time.Second)
	if err != nil {
		lastAutoBan.LastRun = time.Now()
		lastAutoBan.Message = err.Error()
		return lastAutoBanMap(), err
	}
	var toBan []string
	for ip, n := range hits {
		if n >= threshold {
			toBan = append(toBan, ip)
		}
	}
	var banned []string
	if len(toBan) > 0 {
		res := s.QuickBlock(strings.Join(toBan, "\n"), fmt.Sprintf("auto: >=%d hits / %ds", threshold, window), "auto")
		banned = toBan
		if len(res.Failed) > 0 {
			banned = banned[:0]
			for _, ip := range toBan {
				ok := true
				for _, f := range res.Failed {
					if strings.HasPrefix(f, ip) {
						ok = false
						break
					}
				}
				if ok {
					banned = append(banned, ip)
				}
			}
		}
	}
	lastAutoBan.LastRun = time.Now()
	lastAutoBan.Banned = banned
	lastAutoBan.Message = fmt.Sprintf("扫描 %d 行，自动封禁 %d 个 IP", scanned, len(banned))
	return lastAutoBanMap(), nil
}

func lastAutoBanMap() map[string]interface{} {
	return map[string]interface{}{
		"last_run": lastAutoBan.LastRun,
		"message":  lastAutoBan.Message,
		"banned":   lastAutoBan.Banned,
	}
}

func (s *Service) scanSecurityHits(cfg *models.SecurityConfig, window time.Duration) (map[string]int, int, error) {
	path := strings.TrimSpace(cfg.SecurityLogPath)
	if path == "" {
		path = settings.DefaultSecurityLogPath(s.dataDir)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("安全日志不存在")
	}
	defer f.Close()
	cutoff := time.Now().Add(-window)
	hits := map[string]int{}
	scanned := 0
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			scanned++
			if ip, bad := parseHitLine(line, cutoff); bad {
				hits[ip]++
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
	}
	return hits, scanned, nil
}

func parseHitLine(line string, cutoff time.Time) (string, bool) {
	line = strings.TrimSpace(line)
	m := logIPRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return "", false
	}
	ip := m[1]
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.IsLoopback() || parsed.IsPrivate() {
		return ip, false
	}
	if t, ok := parseNginxLogTime(line); ok && t.Before(cutoff) {
		return ip, false
	}
	sm := logStatusRe.FindStringSubmatch(line)
	status := 0
	if len(sm) >= 2 {
		status, _ = strconv.Atoi(sm[1])
	}
	if status == 403 || status == 429 || status == 444 || status == 405 {
		return ip, true
	}
	if strings.Contains(line, "bad_uri=1") || strings.Contains(line, "blocked_ip=1") {
		return ip, true
	}
	return ip, false
}

func parseNginxLogTime(line string) (time.Time, bool) {
	start := strings.Index(line, "[")
	end := strings.Index(line, "]")
	if start < 0 || end <= start {
		return time.Time{}, false
	}
	raw := line[start+1 : end]
	if t, err := time.Parse("02/Jan/2006:15:04:05 -0700", raw); err == nil {
		return t, true
	}
	if t, err := time.Parse("02/Jan/2006:15:04:05 +0000", raw); err == nil {
		return t, true
	}
	return time.Time{}, false
}
