package posthog

import (
	"fmt"
	"html"
	"strings"

	"github.com/luuuunet/owpanel/internal/services/appstore"
	"github.com/luuuunet/owpanel/internal/services/settings"
)

const (
	DefaultDashboardURL = "http://localhost:8020"
	DefaultAPIHost     = "http://localhost:8020"
)

type Feature struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type Status struct {
	Installed    bool      `json:"installed"`
	Running      bool      `json:"running"`
	DashboardURL string    `json:"dashboard_url"`
	APIHost      string    `json:"api_host"`
	Features     []Feature `json:"features"`
	MinRAMGB     int       `json:"min_ram_gb"`
	Hint         string    `json:"hint,omitempty"`
}

type TrackingSnippet struct {
	Snippet string `json:"snippet"`
}

type Service struct {
	dataDir   string
	settings *settings.Service
}

func NewService(dataDir string, settingsSvc *settings.Service) *Service {
	return &Service{dataDir: dataDir, settings: settingsSvc}
}

func DefaultFeatures() []Feature {
	return []Feature{
		{Key: "product_analytics", Name: "Product analytics", Enabled: true},
		{Key: "web_analytics", Name: "Web analytics", Enabled: true},
		{Key: "session_replay", Name: "Session replay", Enabled: true},
		{Key: "feature_flags", Name: "Feature flags", Enabled: true},
		{Key: "experiments", Name: "A/B experiments", Enabled: true},
		{Key: "surveys", Name: "Surveys", Enabled: true},
		{Key: "error_tracking", Name: "Error tracking", Enabled: true},
		{Key: "heatmaps", Name: "Heatmaps", Enabled: true},
		{Key: "data_`.xpack", Name: "Data warehouse (SQL)", Enabled: true},
	}
}

func (s *Service) Status() Status {
	installed := appstore.PosthogInstalled(s.dataDir)
	running := appstore.PosthogComposeStatus(s.dataDir) == "running"
	dashboardURL := DefaultDashboardURL
	apiHost := DefaultAPIHost
	if s.settings != nil {
		if all, err := s.settings.GetAll(); err == nil {
			host := publicHost(all["server_public_ip"])
			if host != "" {
				dashboardURL = fmt.Sprintf("http://%s:8020", host)
				apiHost = dashboardURL
			}
		}
	}
	hint := "PostHog hobby stack needs ~8GB RAM (ClickHouse + Kafka). Use Product Analytics page to deploy and embed tracking."
	if !installed {
		hint = "Install PostHog from App Store for session replay, feature flags, surveys, and error tracking."
	} else if !running {
		hint = "PostHog is installed but stopped — start the stack from App Store or Product Analytics."
	}
	return Status{
		Installed:    installed,
		Running:      running,
		DashboardURL: dashboardURL,
		APIHost:      apiHost,
		Features:     DefaultFeatures(),
		MinRAMGB:     8,
		Hint:         hint,
	}
}

func (s *Service) TrackingSnippet(projectAPIKey, apiHost string) TrackingSnippet {
	projectAPIKey = strings.TrimSpace(projectAPIKey)
	if projectAPIKey == "" {
		projectAPIKey = "YOUR_PROJECT_API_KEY"
	}
	apiHost = strings.TrimSpace(apiHost)
	if apiHost == "" {
		apiHost = DefaultAPIHost
	}
	apiHost = strings.TrimRight(apiHost, "/")

	snippet := fmt.Sprintf(`<script>
  !function(t,e){var o,n,p,r;e.__SV||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.async=!0,p.src=s.api_host+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.capture=u.capture||function(){},u.identify=u.identify||function(){},e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);
  posthog.init('%s', {
    api_host: '%s',
    person_profiles: 'identified_only',
    capture_pageview: true,
    capture_pageleave: true,
  });
</script>`,
		html.EscapeString(projectAPIKey),
		html.EscapeString(apiHost),
	)
	return TrackingSnippet{Snippet: snippet}
}

func publicHost(raw string) string {
	host := strings.TrimSpace(raw)
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host
}
