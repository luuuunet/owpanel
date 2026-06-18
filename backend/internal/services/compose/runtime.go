package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func composeFile(dir string) string {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml"} {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return filepath.Join(dir, "docker-compose.yml")
}

func runCompose(dir, composePath string, args ...string) (string, error) {
	if composePath == "" {
		composePath = composeFile(dir)
	}
	base := []string{"compose", "-f", composePath}
	base = append(base, args...)
	cmd := exec.Command("docker", base...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, fmt.Errorf("%s", text)
	}
	return text, nil
}

func dockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func ensureComposeDir(dir string, scaffold bool, templateID string) (string, error) {
	dir = filepath.Clean(dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	cf := composeFile(dir)
	if _, err := os.Stat(cf); os.IsNotExist(err) {
		if !scaffold {
			return "", fmt.Errorf("未找到 docker-compose.yml: %s", dir)
		}
		yaml, err := TemplateYAML(templateID)
		if err != nil {
			yaml, _ = TemplateYAML("nginx")
		}
		if err := os.WriteFile(cf, []byte(yaml), 0644); err != nil {
			return "", err
		}
	}
	return cf, nil
}

func detectComposeStatus(dir string) string {
	if !dockerAvailable() {
		return "unknown"
	}
	out, err := runCompose(dir, "", "ps", "--status", "running", "-q")
	if err != nil {
		return "stopped"
	}
	if strings.TrimSpace(out) != "" {
		return "running"
	}
	return "stopped"
}

func listComposeServices(dir string) []ServiceInfo {
	if !dockerAvailable() {
		return nil
	}
	out, err := runCompose(dir, "", "ps", "--format", "json")
	if err != nil {
		return nil
	}
	var list []ServiceInfo
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Service string `json:"Service"`
			Name    string `json:"Name"`
			Image   string `json:"Image"`
			State   string `json:"State"`
			Status  string `json:"Status"`
			Ports   string `json:"Ports"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		list = append(list, ServiceInfo{
			Name:      row.Service,
			Container: row.Name,
			Image:     row.Image,
			State:     row.State,
			Status:    row.Status,
			Ports:     row.Ports,
		})
	}
	return list
}
