package dataplatform

import (
	"strings"
)

type LogInsight struct {
	Source  string `json:"source"`
	Level   string `json:"level"`
	Message string `json:"message"`
	FixHint string `json:"fix_hint"`
}

type AIOpsSummary struct {
	MetricsStores   []MetricsEngineStatus `json:"metrics_stores"`
	HealthScore     int                   `json:"health_score"`
	PredictedRisk   string                `json:"predicted_risk"` // low | medium | high
	WatchCount      int                   `json:"watch_count"`
	ServicesDown    int                   `json:"services_down"`
	LogSources      int                   `json:"log_sources"`
	ErrorLines      int                   `json:"error_lines"`
	LogInsights     []LogInsight          `json:"log_insights"`
	GrowthPlatforms []GrowthPlatform      `json:"growth_platforms"`
	AlertHint       string                `json:"alert_hint"`
	Hint            string                `json:"hint,omitempty"`
}

func (s *Service) AIOps() AIOpsSummary {
	metrics := s.MetricsEngines()
	out := AIOpsSummary{
		MetricsStores: metrics,
		PredictedRisk: "low",
	}
	if s.autops != nil {
		if st, err := s.autops.GetStatus(); err == nil && st != nil {
			out.WatchCount = st.WatchCount
			out.ServicesDown = st.DownCount
			if st.DownCount > 2 {
				out.PredictedRisk = "high"
			} else if st.DownCount > 0 {
				out.PredictedRisk = "medium"
			}
		}
	}
	if s.logs != nil {
		sources := s.logs.DiscoverSources()
		out.LogSources = len(sources)
		entries, _ := s.logs.Combined(200)
		for _, e := range entries {
			low := strings.ToLower(e.Content)
			if strings.Contains(low, "error") || strings.Contains(low, "fatal") ||
				strings.Contains(low, "panic") || strings.Contains(low, "failed") {
				out.ErrorLines++
				if len(out.LogInsights) < 8 {
					out.LogInsights = append(out.LogInsights, LogInsight{
						Source:  e.Source,
						Level:   "error",
						Message: truncateInsight(e.Content, 200),
						FixHint: suggestLogFix(e.Source, e.Content),
					})
				}
			}
		}
	}
	out.GrowthPlatforms = s.GrowthPlatforms()
	metricsHealthy := 0
	for _, m := range metrics {
		if m.Running && m.Healthy {
			metricsHealthy++
		}
	}
	score := 50
	if metricsHealthy > 0 {
		score += 20
	}
	if out.ErrorLines == 0 {
		score += 15
	} else if out.ErrorLines < 5 {
		score += 5
	}
	if out.ServicesDown == 0 {
		score += 15
	}
	if score > 100 {
		score = 100
	}
	out.HealthScore = score
	switch out.PredictedRisk {
	case "high":
		out.AlertHint = "Multiple watched services are down — check Auto-Ops and Prometheus alerts."
	case "medium":
		out.AlertHint = "Some services need attention; review log insights below."
	default:
		out.AlertHint = "Telemetry stable; enable VictoriaMetrics for long-term trend prediction."
	}
	if metricsHealthy == 0 {
		out.Hint = "Install Prometheus or VictoriaMetrics for cluster telemetry and autoscaling signals."
	} else if out.ErrorLines > 10 {
		out.Hint = "High error rate in aggregated logs — open Log Center for AI-assisted analysis."
	} else {
		out.Hint = "AIOps pipeline active: metrics + log aggregation + health scoring."
	}
	return out
}

func suggestLogFix(source, msg string) string {
	low := strings.ToLower(msg + " " + source)
	switch {
	case strings.Contains(low, "oom") || strings.Contains(low, "memory"):
		return "Run dashboard memory tune or reduce model/vector DB memory limits."
	case strings.Contains(low, "connection refused"):
		return "Verify dependent service is running (DB, Redis, vector DB, inference port)."
	case strings.Contains(low, "cuda") || strings.Contains(low, "gpu"):
		return "Check nvidia-smi and Docker GPU runtime; fall back to CPU inference if needed."
	case strings.Contains(low, "ssl") || strings.Contains(low, "certificate"):
		return "Renew SSL certificate or fix nginx proxy_pass upstream."
	case strings.Contains(low, "waf") || strings.Contains(low, "403"):
		return "Review WAF rules and Cilium policies for false positives."
	default:
		return "Inspect full log in Log Center; correlate with metrics spike in Grafana."
	}
}

func truncateInsight(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
