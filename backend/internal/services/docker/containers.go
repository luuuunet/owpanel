package docker

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type PortMapping struct {
	HostIP        string `json:"host_ip"`
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type MountMapping struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	ReadOnly    bool   `json:"read_only"`
}

type ContainerDetail struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Image         string         `json:"image"`
	ImageID       string         `json:"image_id"`
	Status        string         `json:"status"`
	Created       string         `json:"created"`
	Ports         []PortMapping  `json:"ports"`
	Env           []string       `json:"env"`
	Mounts        []MountMapping `json:"mounts"`
	Networks      []string       `json:"networks"`
	RestartPolicy string         `json:"restart_policy"`
	Command       []string       `json:"command"`
	WorkingDir    string         `json:"working_dir"`
}

type RunContainerRequest struct {
	Name          string        `json:"name"`
	Image         string        `json:"image"`
	Ports         []PortMapping `json:"ports"`
	Env           []string      `json:"env"`
	Mounts        []MountMapping `json:"mounts"`
	Networks      []string      `json:"networks"`
	RestartPolicy string        `json:"restart_policy"`
	Command       []string      `json:"command"`
	WorkingDir    string        `json:"working_dir"`
}

type RecreateContainerRequest struct {
	Name          *string        `json:"name"`
	Ports         []PortMapping  `json:"ports"`
	Env           []string       `json:"env"`
	Mounts        []MountMapping `json:"mounts"`
	Networks      []string       `json:"networks"`
	RestartPolicy *string        `json:"restart_policy"`
	Command       []string       `json:"command"`
	WorkingDir    *string        `json:"working_dir"`
}

func (s *Service) InspectContainer(id string) (*ContainerDetail, error) {
	if err := s.dockerOK(); err != nil {
		return nil, err
	}
	out, err := runDocker("inspect", id)
	if err != nil {
		return nil, err
	}
	var rows []inspectRow
	if err := json.Unmarshal(out, &rows); err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("container not found")
	}
	return rowToDetail(&rows[0]), nil
}

func (s *Service) ContainerLogs(id string, tail int) (string, error) {
	if err := s.dockerOK(); err != nil {
		return "", err
	}
	if tail <= 0 {
		tail = 200
	}
	if tail > 5000 {
		tail = 5000
	}
	out, err := runDocker("logs", "--tail", strconv.Itoa(tail), id)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (s *Service) Restart(id string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	_, err := runDocker("restart", id)
	return err
}

func (s *Service) RunContainer(req RunContainerRequest) (string, error) {
	if err := s.dockerOK(); err != nil {
		return "", err
	}
	req.Image = strings.TrimSpace(req.Image)
	if req.Image == "" {
		return "", fmt.Errorf("image required")
	}
	args := buildRunArgs(req)
	out, err := runDocker(args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *Service) RecreateContainer(id string, req RecreateContainerRequest) (string, error) {
	detail, err := s.InspectContainer(id)
	if err != nil {
		return "", err
	}
	runReq := RunContainerRequest{
		Name:          detail.Name,
		Image:         firstNonEmpty(detail.ImageID, detail.Image),
		Ports:         detail.Ports,
		Env:           detail.Env,
		Mounts:        detail.Mounts,
		Networks:      detail.Networks,
		RestartPolicy: detail.RestartPolicy,
		Command:       detail.Command,
		WorkingDir:    detail.WorkingDir,
	}
	if req.Name != nil {
		runReq.Name = strings.TrimSpace(*req.Name)
	}
	if len(req.Ports) > 0 {
		runReq.Ports = req.Ports
	}
	if len(req.Env) > 0 {
		runReq.Env = req.Env
	}
	if len(req.Mounts) > 0 {
		runReq.Mounts = req.Mounts
	}
	if len(req.Networks) > 0 {
		runReq.Networks = req.Networks
	}
	if req.RestartPolicy != nil {
		runReq.RestartPolicy = strings.TrimSpace(*req.RestartPolicy)
	}
	if len(req.Command) > 0 {
		runReq.Command = req.Command
	}
	if req.WorkingDir != nil {
		runReq.WorkingDir = strings.TrimSpace(*req.WorkingDir)
	}
	_ = s.Stop(id)
	_ = s.RemoveContainer(id)
	return s.RunContainer(runReq)
}

func buildRunArgs(req RunContainerRequest) []string {
	args := []string{"run", "-d"}
	name := strings.TrimPrefix(strings.TrimSpace(req.Name), "/")
	if name != "" {
		args = append(args, "--name", name)
	}
	if rp := strings.TrimSpace(req.RestartPolicy); rp != "" && rp != "no" {
		args = append(args, "--restart", rp)
	}
	if wd := strings.TrimSpace(req.WorkingDir); wd != "" {
		args = append(args, "-w", wd)
	}
	for _, p := range req.Ports {
		host := strings.TrimSpace(p.HostPort)
		cport := normalizeContainerPort(p.ContainerPort, p.Protocol)
		if host == "" || cport == "" {
			continue
		}
		pub := host + ":" + cport
		if ip := strings.TrimSpace(p.HostIP); ip != "" && ip != "0.0.0.0" {
			pub = ip + ":" + pub
		}
		args = append(args, "-p", pub)
	}
	for _, e := range req.Env {
		e = strings.TrimSpace(e)
		if e != "" {
			args = append(args, "-e", e)
		}
	}
	for _, m := range req.Mounts {
		spec := mountSpec(m)
		if spec != "" {
			args = append(args, "-v", spec)
		}
	}
	networks := req.Networks
	if len(networks) == 0 {
		networks = []string{"bridge"}
	}
	args = append(args, "--network", networks[0])
	args = append(args, req.Image)
	args = append(args, req.Command...)
	return args
}

func mountSpec(m MountMapping) string {
	src := strings.TrimSpace(m.Source)
	dst := strings.TrimSpace(m.Destination)
	if dst == "" {
		return ""
	}
	if src == "" {
		return dst
	}
	spec := src + ":" + dst
	if m.ReadOnly {
		spec += ":ro"
	}
	return spec
}

func normalizeContainerPort(port, protocol string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ""
	}
	if strings.Contains(port, "/") {
		return port
	}
	proto := strings.TrimSpace(protocol)
	if proto == "" {
		proto = "tcp"
	}
	return port + "/" + proto
}

type inspectRow struct {
	ID      string `json:"Id"`
	Image   string `json:"Image"`
	Name    string `json:"Name"`
	Created string `json:"Created"`
	State  struct {
		Status string `json:"Status"`
	} `json:"State"`
	Config struct {
		Image      string   `json:"Image"`
		Env        []string `json:"Env"`
		Cmd        []string `json:"Cmd"`
		WorkingDir string   `json:"WorkingDir"`
	} `json:"Config"`
	HostConfig struct {
		PortBindings map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"PortBindings"`
		RestartPolicy struct {
			Name string `json:"Name"`
		} `json:"RestartPolicy"`
		Binds []string `json:"Binds"`
	} `json:"HostConfig"`
	Mounts []struct {
		Type        string `json:"Type"`
		Source      string `json:"Source"`
		Destination string `json:"Destination"`
		RW          bool   `json:"RW"`
	} `json:"Mounts"`
	NetworkSettings struct {
		Networks map[string]struct{} `json:"Networks"`
	} `json:"NetworkSettings"`
}

func rowToDetail(row *inspectRow) *ContainerDetail {
	d := &ContainerDetail{
		ID:            row.ID,
		Name:          strings.TrimPrefix(row.Name, "/"),
		Image:         firstNonEmpty(row.Config.Image, row.Image),
		ImageID:       firstNonEmpty(row.Image, row.Config.Image),
		Status:        row.State.Status,
		Created:       row.Created,
		Env:           row.Config.Env,
		Command:       row.Config.Cmd,
		WorkingDir:    row.Config.WorkingDir,
		RestartPolicy: row.HostConfig.RestartPolicy.Name,
	}
	if d.RestartPolicy == "" {
		d.RestartPolicy = "no"
	}
	for key, binds := range row.HostConfig.PortBindings {
		proto := "tcp"
		cport := key
		if parts := strings.SplitN(key, "/", 2); len(parts) == 2 {
			cport = parts[0]
			proto = parts[1]
		}
		if len(binds) == 0 {
			d.Ports = append(d.Ports, PortMapping{ContainerPort: cport, Protocol: proto})
			continue
		}
		for _, b := range binds {
			d.Ports = append(d.Ports, PortMapping{
				HostIP:        b.HostIP,
				HostPort:      b.HostPort,
				ContainerPort: cport,
				Protocol:      proto,
			})
		}
	}
	for _, m := range row.Mounts {
		d.Mounts = append(d.Mounts, MountMapping{
			Type:        m.Type,
			Source:      m.Source,
			Destination: m.Destination,
			ReadOnly:    !m.RW,
		})
	}
	if len(d.Mounts) == 0 {
		for _, bind := range row.HostConfig.Binds {
			parts := strings.SplitN(bind, ":", 3)
			if len(parts) < 2 {
				continue
			}
			ro := len(parts) == 3 && parts[2] == "ro"
			d.Mounts = append(d.Mounts, MountMapping{
				Type: "bind", Source: parts[0], Destination: parts[1], ReadOnly: ro,
			})
		}
	}
	for name := range row.NetworkSettings.Networks {
		d.Networks = append(d.Networks, name)
	}
	return d
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
