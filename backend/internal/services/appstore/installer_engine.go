package appstore

import (
	"fmt"
	"os/exec"
	"strings"
)

const mongoDockerContainer = "owpanel-mongodb"

func mongodbDockerRunning() bool {
	out, err := exec.Command(
		"docker", "ps", "--filter", "name=^"+mongoDockerContainer+"$", "--format", "{{.Names}}",
	).Output()
	return err == nil && strings.TrimSpace(string(out)) == mongoDockerContainer
}

func startLinuxEngineService(key string, spec packageSpec) error {
	svc := serviceName(spec)
	if svc == "" {
		return nil
	}
	_ = runCommand("systemctl", "enable", svc)
	if err := runCommand("systemctl", "start", svc); err != nil {
		if key == "mongodb" && mongodbDockerRunning() {
			logInstallLine("MongoDB 已由 Docker 容器 owpanel-mongodb 提供，跳过 systemd mongod")
			return nil
		}
		return fmt.Errorf("start service %s: %w", svc, err)
	}
	return nil
}
