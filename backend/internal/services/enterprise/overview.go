package enterprise

type Overview struct {
	HA              HAStatus          `json:"ha"`
	Compliance      ComplianceReport  `json:"compliance"`
	AuditStats      AuditStats        `json:"audit_stats"`
	Monitoring      AdvancedMonitoring `json:"monitoring"`
	SecurityScore   int               `json:"security_score"`
	SecurityGrade   string            `json:"security_grade"`
	UptimeAlerts    int               `json:"uptime_alerts"`
}

func (s *Service) GetOverview() (Overview, error) {
	out := Overview{}
	var err error
	out.HA, err = s.GetHAStatus()
	if err != nil {
		return out, err
	}
	out.Compliance = s.RunComplianceChecks()
	out.AuditStats, _ = s.Stats(true)
	out.Monitoring, _ = s.GetAdvancedMonitoring()
	out.UptimeAlerts = out.Monitoring.Uptime.Down
	if s.security != nil {
		sc := s.security.ComputeScore()
		out.SecurityScore = sc.Score
		out.SecurityGrade = sc.Grade
	}
	return out, nil
}
