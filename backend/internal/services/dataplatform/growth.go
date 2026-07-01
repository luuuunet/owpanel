package dataplatform

import "github.com/luuuunet/owpanel/internal/services/appstore"

type GrowthPlatform struct {
	Key       string   `json:"key"`
	Name      string   `json:"name"`
	Installed bool     `json:"installed"`
	Running   bool     `json:"running"`
	Port      int      `json:"port"`
	Features  []string `json:"features"`
	UseCase   string   `json:"use_case"`
}

func (s *Service) GrowthPlatforms() []GrowthPlatform {
	platforms := []GrowthPlatform{
		{
			Key: "posthog", Name: "PostHog", Port: 8020,
			UseCase: "Product analytics, session replay, feature flags, A/B, surveys, error tracking",
			Features: []string{"Product analytics", "Session replay", "Feature flags", "A/B tests", "Surveys", "Error tracking", "Heatmaps"},
		},
		{
			Key: "openpanel-analytics", Name: "OpenPanel Analytics", Port: 3300,
			UseCase: "Lightweight A/B testing, funnels, and session replay",
			Features: []string{"A/B tests", "Funnels", "Session replay", "Cohorts"},
		},
		{
			Key: "umami", Name: "Umami", Port: 3018,
			UseCase: "Privacy-friendly web analytics",
			Features: []string{"Page views", "Privacy"},
		},
	}
	for i := range platforms {
		p := &platforms[i]
		switch p.Key {
		case "posthog":
			p.Installed = appstore.PosthogInstalled(s.dataDir)
			if p.Installed {
				p.Running = appstore.PosthogComposeStatus(s.dataDir) == "running"
			}
		case "openpanel-analytics":
			p.Installed = appstore.OpenpanelInstalled(s.dataDir)
			if p.Installed {
				p.Running = appstore.OpenpanelComposeStatus(s.dataDir) == "running"
			}
		default:
			if s.appstore != nil {
				if app, err := s.appstore.Get(p.Key); err == nil && app.Installed {
					p.Installed = true
					p.Running = s.appstore.LiveStatus(p.Key) == "running"
				}
			}
		}
	}
	return platforms
}
