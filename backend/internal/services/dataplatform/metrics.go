package dataplatform

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type MetricsEngineStatus struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Status    string `json:"status"`
	Port      int    `json:"port"`
	Endpoint  string `json:"endpoint"`
	UseCase   string `json:"use_case"`
	Healthy   bool   `json:"healthy"`
	Series    int    `json:"series,omitempty"`
	Message   string `json:"message,omitempty"`
}

var metricsEngines = []struct {
	Key, Name, UseCase string
	Port               int
	HealthPath         string
}{
	{Key: "prometheus", Name: "Prometheus", UseCase: "Cluster metrics scraping & alerting", Port: 9090, HealthPath: "/-/healthy"},
	{Key: "victoria-metrics", Name: "VictoriaMetrics", UseCase: "High-performance long-term metrics storage", Port: 8428, HealthPath: "/health"},
	{Key: "grafana", Name: "Grafana", UseCase: "Metrics visualization & dashboards", Port: 3003, HealthPath: "/api/health"},
}

func (s *Service) MetricsEngines() []MetricsEngineStatus {
	out := make([]MetricsEngineStatus, 0, len(metricsEngines))
	client := &http.Client{Timeout: 3 * time.Second}
	for _, e := range metricsEngines {
		st := MetricsEngineStatus{
			Key:      e.Key,
			Name:     e.Name,
			UseCase:  e.UseCase,
			Port:     e.Port,
			Endpoint: fmt.Sprintf("http://127.0.0.1:%d", e.Port),
		}
		if s.appstore != nil {
			app, err := s.appstore.Get(e.Key)
			if err == nil && app.Installed {
				st.Installed = true
				live := s.appstore.LiveStatus(e.Key)
				st.Status = live
				st.Running = live == "running"
			}
		}
		if st.Running {
			url := st.Endpoint + e.HealthPath
			resp, err := client.Get(url)
			if err == nil {
				defer resp.Body.Close()
				st.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 300
			}
			if e.Key == "prometheus" && st.Healthy {
				st.Series = countPrometheusSeries(client, st.Endpoint)
			}
		}
		out = append(out, st)
	}
	return out
}

func countPrometheusSeries(client *http.Client, base string) int {
	resp, err := client.Get(base + "/api/v1/status/tsdb")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var parsed struct {
		Data struct {
			TotalSeries int `json:"totalSeries"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0
	}
	return parsed.Data.TotalSeries
}
