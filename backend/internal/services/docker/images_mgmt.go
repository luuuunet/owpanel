package docker

import (
	"fmt"
	"strings"
)

func (s *Service) PullImage(ref string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("image reference required")
	}
	_, err := runDocker("pull", ref)
	return err
}

func (s *Service) RemoveImage(id string, force bool) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	args := []string{"rmi"}
	if force {
		args = append(args, "-f")
	}
	args = append(args, id)
	_, err := runDocker(args...)
	return err
}

func (s *Service) PruneImages() (string, error) {
	if err := s.dockerOK(); err != nil {
		return "", err
	}
	out, err := runDocker("image", "prune", "-f")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
