package docker

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type ContainerStats struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	CPUPerc       string  `json:"cpu_perc"`
	MemUsage      string  `json:"mem_usage"`
	MemPerc       string  `json:"mem_perc"`
	NetIO         string  `json:"net_io"`
	BlockIO       string  `json:"block_io"`
	PIDs          string  `json:"pids"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemPercent    float64 `json:"mem_percent"`
}

type DockerEvent struct {
	Time   int64  `json:"time"`
	Type   string `json:"type"`
	Action string `json:"action"`
	Actor  string `json:"actor"`
	ID     string `json:"id"`
	From   string `json:"from,omitempty"`
	Status string `json:"status,omitempty"`
}

type SystemOverview struct {
	Version        string `json:"version"`
	Containers     int    `json:"containers"`
	ContainersRun  int    `json:"containers_running"`
	ContainersStop int    `json:"containers_stopped"`
	Images         int    `json:"images"`
	Driver         string `json:"driver"`
	MemoryTotal    string `json:"memory_total"`
	CPUs           int    `json:"cpus"`
	OS             string `json:"operating_system"`
	Architecture   string `json:"architecture"`
	ServerVersion  string `json:"server_version"`
	ImagesSize     string `json:"images_size,omitempty"`
	VolumesSize    string `json:"volumes_size,omitempty"`
	ContainersSize string `json:"containers_size,omitempty"`
}

func (s *Service) Pause(id string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	_, err := runDocker("pause", id)
	return err
}

func (s *Service) Unpause(id string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	_, err := runDocker("unpause", id)
	return err
}

func (s *Service) Kill(id, signal string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	signal = strings.TrimSpace(signal)
	if signal == "" {
		signal = "SIGKILL"
	}
	_, err := runDocker("kill", "--signal", signal, id)
	return err
}

func (s *Service) Rename(id, name string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name required")
	}
	_, err := runDocker("rename", id, name)
	return err
}

func (s *Service) RawInspect(id string) (json.RawMessage, error) {
	if err := s.dockerOK(); err != nil {
		return nil, err
	}
	out, err := runDocker("inspect", id)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) ContainerStats(id string) (*ContainerStats, error) {
	if err := s.dockerOK(); err != nil {
		return nil, err
	}
	out, err := runDocker("stats", "--no-stream", "--format", "{{json .}}", id)
	if err != nil {
		return nil, err
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return nil, fmt.Errorf("no stats")
	}
	// docker may print multiple lines; take first
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	var row struct {
		ID       string `json:"ID"`
		Name     string `json:"Name"`
		CPUPerc  string `json:"CPUPerc"`
		MemUsage string `json:"MemUsage"`
		MemPerc  string `json:"MemPerc"`
		NetIO    string `json:"NetIO"`
		BlockIO  string `json:"BlockIO"`
		PIDs     string `json:"PIDs"`
	}
	if err := json.Unmarshal([]byte(line), &row); err != nil {
		return nil, err
	}
	st := &ContainerStats{
		ID: row.ID, Name: row.Name, CPUPerc: row.CPUPerc, MemUsage: row.MemUsage,
		MemPerc: row.MemPerc, NetIO: row.NetIO, BlockIO: row.BlockIO, PIDs: row.PIDs,
		CPUPercent: parsePerc(row.CPUPerc), MemPercent: parsePerc(row.MemPerc),
	}
	return st, nil
}

func parsePerc(s string) float64 {
	s = strings.TrimSpace(strings.TrimSuffix(s, "%"))
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func (s *Service) ContainerLogsEx(id string, tail int, timestamps bool) (string, error) {
	if err := s.dockerOK(); err != nil {
		return "", err
	}
	if tail <= 0 {
		tail = 200
	}
	if tail > 5000 {
		tail = 5000
	}
	args := []string{"logs", "--tail", strconv.Itoa(tail)}
	if timestamps {
		args = append(args, "--timestamps")
	}
	args = append(args, id)
	out, err := runDocker(args...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *Service) ExecOnce(id string, command []string) (string, error) {
	if err := s.dockerOK(); err != nil {
		return "", err
	}
	if len(command) == 0 {
		return "", fmt.Errorf("command required")
	}
	args := append([]string{"exec", id}, command...)
	out, err := runDocker(args...)
	return string(out), err
}

func (s *Service) ListEvents(sinceSec int) ([]DockerEvent, error) {
	if err := s.dockerOK(); err != nil {
		return nil, err
	}
	if sinceSec <= 0 {
		sinceSec = 3600
	}
	if sinceSec > 86400*7 {
		sinceSec = 86400 * 7
	}
	since := time.Now().Add(-time.Duration(sinceSec) * time.Second).Format(time.RFC3339)
	until := time.Now().Format(time.RFC3339)
	out, err := runDocker("events", "--since", since, "--until", until, "--format", "{{json .}}")
	if err != nil {
		// empty range often still exits 0; treat parse below
		if strings.TrimSpace(string(out)) == "" {
			return []DockerEvent{}, nil
		}
	}
	var events []DockerEvent
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		ev := DockerEvent{
			Type:   fmt.Sprint(raw["Type"]),
			Action: fmt.Sprint(raw["Action"]),
			Status: fmt.Sprint(raw["status"]),
			From:   fmt.Sprint(raw["from"]),
			ID:     fmt.Sprint(raw["id"]),
		}
		if t, ok := raw["time"].(float64); ok {
			ev.Time = int64(t)
		}
		if actor, ok := raw["Actor"].(map[string]interface{}); ok {
			if attrs, ok := actor["Attributes"].(map[string]interface{}); ok {
				if n, ok := attrs["name"]; ok {
					ev.Actor = fmt.Sprint(n)
				}
			}
			if ev.Actor == "" {
				ev.Actor = fmt.Sprint(actor["ID"])
			}
		}
		events = append(events, ev)
	}
	// newest first, cap
	if len(events) > 200 {
		events = events[len(events)-200:]
	}
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	return events, nil
}

func (s *Service) SystemOverview() (*SystemOverview, error) {
	if err := s.dockerOK(); err != nil {
		return nil, err
	}
	out, err := runDocker("info", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	var info map[string]interface{}
	if err := json.Unmarshal(out, &info); err != nil {
		return nil, err
	}
	ov := &SystemOverview{
		Version:       s.dockerVersion(),
		ServerVersion: fmt.Sprint(info["ServerVersion"]),
		Driver:        fmt.Sprint(info["Driver"]),
		OS:            fmt.Sprint(info["OperatingSystem"]),
		Architecture:  fmt.Sprint(info["Architecture"]),
	}
	if n, ok := info["Containers"].(float64); ok {
		ov.Containers = int(n)
	}
	if n, ok := info["ContainersRunning"].(float64); ok {
		ov.ContainersRun = int(n)
	}
	if n, ok := info["ContainersStopped"].(float64); ok {
		ov.ContainersStop = int(n)
	}
	if n, ok := info["Images"].(float64); ok {
		ov.Images = int(n)
	}
	if n, ok := info["NCPU"].(float64); ok {
		ov.CPUs = int(n)
	}
	if n, ok := info["MemTotal"].(float64); ok {
		ov.MemoryTotal = formatBytes(int64(n))
	}

	if dfOut, err := runDocker("system", "df", "--format", "{{json .}}"); err == nil {
		for _, line := range strings.Split(string(dfOut), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var row map[string]interface{}
			if json.Unmarshal([]byte(line), &row) != nil {
				continue
			}
			typ := strings.ToLower(fmt.Sprint(row["Type"]))
			size := fmt.Sprint(row["Size"])
			switch {
			case strings.Contains(typ, "image"):
				ov.ImagesSize = size
			case strings.Contains(typ, "container"):
				ov.ContainersSize = size
			case strings.Contains(typ, "volume"):
				ov.VolumesSize = size
			}
		}
	}
	return ov, nil
}

func formatBytes(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f := float64(n)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}

func (s *Service) InspectImage(id string) (json.RawMessage, error) {
	if err := s.dockerOK(); err != nil {
		return nil, err
	}
	out, err := runDocker("image", "inspect", id)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) TagImage(source, target string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	source = strings.TrimSpace(source)
	target = strings.TrimSpace(target)
	if source == "" || target == "" {
		return fmt.Errorf("source and target required")
	}
	_, err := runDocker("tag", source, target)
	return err
}

func (s *Service) InspectVolume(name string) (json.RawMessage, error) {
	if err := s.dockerOK(); err != nil {
		return nil, err
	}
	out, err := runDocker("volume", "inspect", name)
	if err != nil {
		return nil, err
	}
	var raw json.RawMessage
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) ConnectNetwork(networkID, containerID string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	_, err := runDocker("network", "connect", networkID, containerID)
	return err
}

func (s *Service) DisconnectNetwork(networkID, containerID string, force bool) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	args := []string{"network", "disconnect"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, networkID, containerID)
	_, err := runDocker(args...)
	return err
}
