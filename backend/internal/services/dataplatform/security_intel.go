package dataplatform

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SecurityEvent struct {
	Time    string `json:"time"`
	Source  string `json:"source"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type SecurityRecommendation struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Action      string `json:"action"`
}

type SecurityIntelSummary struct {
	Score           int                      `json:"score"`
	CiliumReady     bool                     `json:"cilium_ready"`
	PolicyCount     int                      `json:"policy_count"`
	AuditMode       bool                     `json:"audit_mode"`
	HubbleEnabled   bool                     `json:"hubble_enabled"`
	RecentEvents    []SecurityEvent          `json:"recent_events"`
	Recommendations []SecurityRecommendation `json:"recommendations"`
	AIInsight       string                   `json:"ai_insight"`
}

func (s *Service) SecurityIntel() SecurityIntelSummary {
	sum := SecurityIntelSummary{
		Score:        40,
		RecentEvents: []SecurityEvent{},
	}
	if s.cilium == nil {
		sum.AIInsight = "Install Cilium from Protection Center to enable eBPF network policy intelligence."
		return sum
	}
	st, err := s.cilium.Status()
	if err == nil && st != nil {
		sum.CiliumReady = st.CiliumReady
		if st.CiliumReady {
			sum.Score += 25
		}
	}
	cfg, _ := s.cilium.GetConfig()
	if cfg != nil {
		sum.AuditMode = cfg.AuditMode
		sum.HubbleEnabled = cfg.HubbleEnabled
		if cfg.HostFirewallEnabled {
			sum.Score += 15
		}
		if cfg.AuditMode {
			sum.Score += 5
		}
	}
	policies, _ := s.cilium.ListPolicies()
	sum.PolicyCount = len(policies)
	if sum.PolicyCount > 0 {
		sum.Score += min(15, sum.PolicyCount*3)
	}
	sum.RecentEvents = tailSecurityLogs(s.dataDir, 20)
	sum.Recommendations = buildSecurityRecommendations(sum)
	sum.AIInsight = buildAIInsight(sum)
	if sum.Score > 100 {
		sum.Score = 100
	}
	return sum
}

func buildSecurityRecommendations(sum SecurityIntelSummary) []SecurityRecommendation {
	var recs []SecurityRecommendation
	if !sum.CiliumReady {
		recs = append(recs, SecurityRecommendation{
			Key: "install_cilium", Title: "Deploy Cilium stack",
			Description: "Enable eBPF network policies and Hubble observability for cluster nodes.",
			Severity: "high", Action: "/protection?tab=cilium",
		})
	}
	if sum.PolicyCount == 0 && sum.CiliumReady {
		recs = append(recs, SecurityRecommendation{
			Key: "baseline_policy", Title: "Apply baseline policies",
			Description: "Restrict SSH, HTTP/HTTPS and panel ports with auditable Cilium policies.",
			Severity: "high", Action: "/protection?tab=cilium",
		})
	}
	if sum.CiliumReady && !sum.AuditMode && sum.PolicyCount < 3 {
		recs = append(recs, SecurityRecommendation{
			Key: "enable_audit", Title: "Enable audit mode first",
			Description: "Run policies in audit mode before enforcement to avoid locking yourself out.",
			Severity: "medium", Action: "/protection?tab=cilium",
		})
	}
	if !sum.HubbleEnabled && sum.CiliumReady {
		recs = append(recs, SecurityRecommendation{
			Key: "enable_hubble", Title: "Enable Hubble flows",
			Description: "Flow telemetry helps detect lateral movement and anomalous east-west traffic.",
			Severity: "medium", Action: "/protection?tab=cilium",
		})
	}
	return recs
}

func buildAIInsight(sum SecurityIntelSummary) string {
	if !sum.CiliumReady {
		return "No Cilium deployment detected. Vector RAG and model endpoints should be isolated with network policies once Cilium is active."
	}
	if len(sum.RecentEvents) > 5 {
		return fmt.Sprintf("Detected %d recent security-related log lines. Review failed SSH/WAF events and consider tightening ingress policies for AI API ports (8095, 11434).", len(sum.RecentEvents))
	}
	if sum.AuditMode {
		return "Audit mode is on — policies are logged but not enforced. Review Hubble flows, then disable audit mode for production enforcement."
	}
	return "Network posture looks stable. Pair Cilium policies with Fail2ban/WAF rules for layered edge defense around AI and storage services."
}

func tailSecurityLogs(dataDir string, limit int) []SecurityEvent {
	paths := []string{
		filepath.Join(dataDir, "logs", "security.log"),
		"/var/log/fail2ban.log",
	}
	var events []SecurityEvent
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		var lines []string
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			lines = append(lines, sc.Text())
		}
		_ = f.Close()
		for i := len(lines) - 1; i >= 0 && len(events) < limit; i-- {
			line := lines[i]
			low := strings.ToLower(line)
			if !strings.Contains(low, "ban") && !strings.Contains(low, "attack") &&
				!strings.Contains(low, "denied") && !strings.Contains(low, "waf") &&
				!strings.Contains(low, "cilium") {
				continue
			}
			events = append(events, SecurityEvent{
				Time:    time.Now().Format(time.RFC3339),
				Source:  filepath.Base(p),
				Level:   "warn",
				Message: truncate(line, 240),
			})
		}
	}
	return events
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
