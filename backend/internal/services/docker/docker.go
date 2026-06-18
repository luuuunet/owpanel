package docker

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"

	"gorm.io/gorm"
)

type Container struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Image      string `json:"image"`
	Status     string `json:"status"`
	Ports      string `json:"ports"`
	Created    string `json:"created"`
	BindDomain string `json:"bind_domain,omitempty"`
	AccessURL  string `json:"access_url,omitempty"`
	HostPort   int    `json:"host_port,omitempty"`
}

type Image struct {
	ID       string `json:"id"`
	RepoTags string `json:"repo_tags"`
	Size     string `json:"size"`
	Created  string `json:"created"`
}

type Volume struct {
	Name       string   `json:"name"`
	Driver     string   `json:"driver"`
	Mountpoint string   `json:"mountpoint"`
	Containers []string `json:"containers"`
	InUse      bool     `json:"in_use"`
	Category   string   `json:"category"`
}

type NetworkEndpoint struct {
	Name string `json:"name"`
	IPv4 string `json:"ipv4"`
}

type Network struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Driver         string            `json:"driver"`
	Scope          string            `json:"scope"`
	Subnet         string            `json:"subnet,omitempty"`
	Gateway        string            `json:"gateway,omitempty"`
	Endpoints      []NetworkEndpoint `json:"endpoints,omitempty"`
	ContainerCount int               `json:"container_count"`
	InUse          bool              `json:"in_use"`
	IsSystem       bool              `json:"is_system"`
}

type Service struct {
	db      *gorm.DB
	dataDir string
	ws      WebServerHooks
}

func NewService(db *gorm.DB, dataDir string) *Service {
	return &Service{db: db, dataDir: dataDir}
}

type DockerStatus struct {
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Version   string `json:"version"`
	DaemonOK  bool   `json:"daemon_ok"`
}

func (s *Service) Status() DockerStatus {
	st := DockerStatus{}
	if !s.dockerAvailable() {
		return st
	}
	st.Installed = true
	st.Version = s.dockerVersion()
	st.DaemonOK = s.dockerDaemonOK()
	st.Running = st.DaemonOK
	if !st.DaemonOK && runtime.GOOS != "windows" {
		st.Running = s.dockerServiceActive()
	}
	return st
}

func (s *Service) dockerVersion() string {
	out, err := exec.Command("docker", "version", "--format", "{{.Server.Version}}").Output()
	if err == nil {
		v := strings.TrimSpace(string(out))
		if v != "" {
			return v
		}
	}
	out, err = exec.Command("docker", "--version").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

func (s *Service) dockerDaemonOK() bool {
	err := exec.Command("docker", "info").Run()
	return err == nil
}

func (s *Service) dockerServiceActive() bool {
	out, err := exec.Command("systemctl", "is-active", "docker").Output()
	return err == nil && strings.TrimSpace(string(out)) == "active"
}

func (s *Service) dockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func (s *Service) ListContainers() ([]Container, error) {
	list, err := s.listContainersRaw()
	if err != nil {
		return nil, err
	}
	return s.enrichContainers(list), nil
}

func (s *Service) listContainersRaw() ([]Container, error) {
	if !s.dockerAvailable() {
		return []Container{}, nil
	}
	out, err := exec.Command("docker", "ps", "-a", "--format", "{{json .}}").Output()
	if err != nil {
		return nil, err
	}
	var list []Container
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			ID      string `json:"ID"`
			Names   string `json:"Names"`
			Image   string `json:"Image"`
			Status  string `json:"Status"`
			Ports   string `json:"Ports"`
			Created string `json:"CreatedAt"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		list = append(list, Container{
			ID: row.ID, Name: strings.TrimPrefix(row.Names, "/"),
			Image: row.Image, Status: row.Status, Ports: row.Ports, Created: row.Created,
		})
	}
	return list, nil
}

func (s *Service) ListImages() ([]Image, error) {
	if !s.dockerAvailable() {
		return []Image{}, nil
	}
	out, err := exec.Command("docker", "images", "--format", "{{json .}}").Output()
	if err != nil {
		return nil, err
	}
	var list []Image
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			ID       string `json:"ID"`
			RepoTags string `json:"Repository"`
			Tag      string `json:"Tag"`
			Size     string `json:"Size"`
			Created  string `json:"CreatedSince"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		tag := row.RepoTags
		if row.Tag != "" && row.Tag != "<none>" {
			tag = row.RepoTags + ":" + row.Tag
		}
		list = append(list, Image{ID: row.ID, RepoTags: tag, Size: row.Size, Created: row.Created})
	}
	return list, nil
}

func (s *Service) ListVolumes() ([]Volume, error) {
	if !s.dockerAvailable() {
		return []Volume{}, nil
	}
	out, err := exec.Command("docker", "volume", "ls", "--format", "{{json .}}").Output()
	if err != nil {
		return nil, err
	}
	var list []Volume
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			Name       string `json:"Name"`
			Driver     string `json:"Driver"`
			Mountpoint string `json:"Mountpoint"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		list = append(list, Volume{Name: row.Name, Driver: row.Driver, Mountpoint: row.Mountpoint})
	}
	return s.enrichVolumeUsage(list), nil
}

func (s *Service) ListNetworks() ([]Network, error) {
	if !s.dockerAvailable() {
		return []Network{}, nil
	}
	out, err := exec.Command("docker", "network", "ls", "--format", "{{json .}}").Output()
	if err != nil {
		return nil, err
	}
	var list []Network
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			ID     string `json:"ID"`
			Name   string `json:"Name"`
			Driver string `json:"Driver"`
			Scope  string `json:"Scope"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		list = append(list, Network{ID: row.ID, Name: row.Name, Driver: row.Driver, Scope: row.Scope})
	}
	return s.enrichNetworks(list), nil
}

func (s *Service) Start(id string) error {
	return exec.Command("docker", "start", id).Run()
}

func (s *Service) Stop(id string) error {
	return exec.Command("docker", "stop", id).Run()
}

func (s *Service) RemoveContainer(id string) error {
	_ = s.UnbindDomain(id)
	_ = exec.Command("docker", "stop", id).Run()
	return exec.Command("docker", "rm", "-f", id).Run()
}
