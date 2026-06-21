package database

import (
	"os/exec"
	"strings"
)

const mongoDockerContainer = "owpanel-mongodb"

type MongoDBStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Running   bool   `json:"running"`
}

func mongoDockerRunning() bool {
	out, err := exec.Command("docker", "ps", "--filter", "name=^"+mongoDockerContainer+"$", "--format", "{{.Names}}").CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) == mongoDockerContainer
}

func mongoPing() (bool, []byte) {
	if shell, err := findBinary("mongosh", "mongo"); err == nil {
		out, err := exec.Command(
			shell, "--quiet", "mongodb://127.0.0.1:27017",
			"--eval", "db.runCommand({ping:1}).ok",
		).CombinedOutput()
		return err == nil && strings.TrimSpace(string(out)) == "1", out
	}
	if !mongoDockerRunning() {
		return false, nil
	}
	out, err := exec.Command(
		"docker", "exec", mongoDockerContainer,
		"mongosh", "--quiet", "mongodb://127.0.0.1:27017",
		"--eval", "db.runCommand({ping:1}).ok",
	).CombinedOutput()
	return err == nil && strings.TrimSpace(string(out)) == "1", out
}

func (s *Service) MongoDBStatus() MongoDBStatus {
	st := MongoDBStatus{}
	if shell, err := findBinary("mongosh", "mongo"); err == nil {
		st.Installed = true
		if out, err := exec.Command(shell, "--version").CombinedOutput(); err == nil {
			st.Version = strings.TrimSpace(string(out))
			if i := strings.Index(st.Version, "mongosh"); i >= 0 {
				st.Version = strings.TrimSpace(st.Version[i:])
			}
		}
	} else if md, err := findBinary("mongod"); err == nil {
		st.Installed = true
		if out, err := exec.Command(md, "--version").CombinedOutput(); err == nil {
			st.Version = strings.TrimSpace(string(out))
		}
	} else if mongoDockerRunning() {
		st.Installed = true
		st.Version = "Docker (mongo:7.0)"
	}
	st.Running, _ = mongoPing()
	return st
}

func (s *Service) mongoShellExec(args ...string) ([]byte, error) {
	if bin, err := findBinary("mongosh", "mongo"); err == nil {
		base := []string{"--quiet", "mongodb://127.0.0.1:27017"}
		return exec.Command(bin, append(base, args...)...).CombinedOutput()
	}
	if mongoDockerRunning() {
		base := []string{"exec", mongoDockerContainer, "mongosh", "--quiet", "mongodb://127.0.0.1:27017"}
		return exec.Command("docker", append(base, args...)...).CombinedOutput()
	}
	return nil, exec.ErrNotFound
}
