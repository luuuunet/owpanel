package toolbox

import (
	"os/exec"
	"runtime"
	"strings"

	"gorm.io/gorm"
)

type Result struct {
	Command string `json:"command"`
	Output  string `json:"output"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (s *Service) Ping(host string) (*Result, error) {
	flag := "-c"
	if runtime.GOOS == "windows" {
		flag = "-n"
	}
	out, err := exec.Command("ping", flag, "4", host).CombinedOutput()
	return &Result{Command: "ping " + host, Output: string(out)}, err
}

func (s *Service) Traceroute(host string) (*Result, error) {
	cmd := "traceroute"
	if runtime.GOOS == "windows" {
		cmd = "tracert"
	}
	out, err := exec.Command(cmd, host).CombinedOutput()
	return &Result{Command: cmd + " " + host, Output: string(out)}, err
}

func (s *Service) DNSLookup(domain string) (*Result, error) {
	out, err := exec.Command("nslookup", domain).CombinedOutput()
	return &Result{Command: "nslookup " + domain, Output: string(out)}, err
}

func (s *Service) Whois(domain string) (*Result, error) {
	out, err := exec.Command("nslookup", "-type=any", domain).CombinedOutput()
	if err != nil {
		return &Result{Command: "whois " + domain, Output: strings.TrimSpace(string(out))}, err
	}
	return &Result{Command: "whois " + domain, Output: string(out)}, nil
}
