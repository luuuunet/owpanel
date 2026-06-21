package dataplatform

type AutoDefenseRule struct {
	Key         string `json:"key"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Action      string `json:"action"`
	AutoApply   bool   `json:"auto_apply"`
}

type SecOpsSummary struct {
	SecurityIntelSummary
	ThreatLevel     string            `json:"threat_level"` // low | medium | high
	AutoDefenseRules []AutoDefenseRule  `json:"auto_defense_rules"`
	AuditScore      int               `json:"audit_score"`
	HubbleEnabled   bool              `json:"hubble_enabled"`
	AIAnalysis      string            `json:"ai_analysis"`
}

func (s *Service) SecOps() SecOpsSummary {
	base := s.SecurityIntel()
	out := SecOpsSummary{
		SecurityIntelSummary: base,
		AuditScore:           base.Score,
		ThreatLevel:          "low",
	}
	if s.cilium != nil {
		if cfg, err := s.cilium.GetConfig(); err == nil && cfg != nil {
			out.HubbleEnabled = cfg.HubbleEnabled
		}
	}
	if len(base.RecentEvents) > 8 {
		out.ThreatLevel = "high"
	} else if len(base.RecentEvents) > 3 {
		out.ThreatLevel = "medium"
	}
	out.AutoDefenseRules = buildAutoDefenseRules(base, out.HubbleEnabled)
	out.AIAnalysis = buildSecOpsAIAnalysis(base, out.ThreatLevel)
	return out
}

func buildAutoDefenseRules(base SecurityIntelSummary, hubble bool) []AutoDefenseRule {
	var rules []AutoDefenseRule
	if !base.CiliumReady {
		rules = append(rules, AutoDefenseRule{
			Key: "block_ai_exposure", Title: "Restrict AI API ingress",
			Description: "Limit ports 8095, 11434, 8000 to internal networks until Cilium is active.",
			Severity: "high", Action: "/protection?tab=cilium", AutoApply: false,
		})
	}
	rules = append(rules, AutoDefenseRule{
		Key: "ai_port_policy", Title: "AI inference network policy",
		Description: "Apply Cilium policy allowing only panel and trusted CIDRs to TGI/Ollama/vLLM ports.",
		Severity: "medium", Action: "/protection?tab=cilium", AutoApply: base.CiliumReady,
	})
	if !hubble && base.CiliumReady {
		rules = append(rules, AutoDefenseRule{
			Key: "enable_hubble", Title: "Enable Hubble flow analytics",
			Description: "Detect lateral movement and anomalous east-west traffic around AI workloads.",
			Severity: "medium", Action: "/protection?tab=cilium", AutoApply: false,
		})
	}
	if base.AuditMode {
		rules = append(rules, AutoDefenseRule{
			Key: "enforce_policies", Title: "Exit audit mode",
			Description: "Policies are logging only — disable audit mode to enforce blocks after review.",
			Severity: "low", Action: "/protection?tab=cilium", AutoApply: false,
		})
	}
	rules = append(rules, AutoDefenseRule{
		Key: "waf_ai_paths", Title: "WAF rate-limit on AI endpoints",
		Description: "Combine Nginx WAF with Cilium for DDoS protection on public inference routes.",
		Severity: "medium", Action: "/protection?tab=waf", AutoApply: false,
	})
	return rules
}

func buildSecOpsAIAnalysis(base SecurityIntelSummary, threat string) string {
	if threat == "high" {
		return "Elevated security signals detected. Prioritize Cilium baseline policies and review Fail2ban/WAF logs for coordinated probes against AI ports."
	}
	if !base.CiliumReady {
		return "SecOps recommendation: deploy Cilium before exposing inference APIs publicly. Treat network policies as versioned data assets alongside model weights."
	}
	if base.AuditMode {
		return "Policies are in audit-only mode — flows are logged but not blocked. Review Hubble, then enforce."
	}
	return "Security posture stable. Enable Hubble for AI traffic anomaly detection and auto-generate policies from observed attack patterns."
}
