package docker

import (
	"fmt"
	"os/exec"
	"strings"
)

func runDocker(args ...string) ([]byte, error) {
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("%s", msg)
	}
	return out, nil
}

func (s *Service) dockerOK() error {
	if !s.dockerAvailable() {
		return fmt.Errorf("docker not installed")
	}
	return nil
}
