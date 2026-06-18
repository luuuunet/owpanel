package database

import (
	"os/exec"
	"strings"
)

type MongoDBStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Running   bool   `json:"running"`
}

func (s *Service) MongoDBStatus() MongoDBStatus {
	st := MongoDBStatus{}
	shell, err := findBinary("mongosh", "mongo")
	if err != nil {
		if md, err2 := findBinary("mongod"); err2 == nil {
			st.Installed = true
			if out, err := exec.Command(md, "--version").CombinedOutput(); err == nil {
				st.Version = strings.TrimSpace(string(out))
			}
		}
		return st
	}
	st.Installed = true
	if out, err := exec.Command(shell, "--version").CombinedOutput(); err == nil {
		st.Version = strings.TrimSpace(string(out))
		if i := strings.Index(st.Version, "mongosh"); i >= 0 {
			st.Version = strings.TrimSpace(st.Version[i:])
		}
	}
	out, err := exec.Command(
		shell, "--quiet", "mongodb://127.0.0.1:27017",
		"--eval", "db.runCommand({ping:1}).ok",
	).CombinedOutput()
	st.Running = err == nil && strings.TrimSpace(string(out)) == "1"
	return st
}

func (s *Service) mongoShellExec(args ...string) ([]byte, error) {
	bin, err := findBinary("mongosh", "mongo")
	if err != nil {
		return nil, err
	}
	base := []string{"--quiet", "mongodb://127.0.0.1:27017"}
	return exec.Command(bin, append(base, args...)...).CombinedOutput()
}
