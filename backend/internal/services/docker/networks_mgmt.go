package docker

import (
	"encoding/json"
	"fmt"
	"strings"
)

type CreateNetworkOpts struct {
	Name    string
	Driver  string
	Subnet  string
	Gateway string
}

func isSystemNetwork(name string) bool {
	switch name {
	case "bridge", "host", "none":
		return true
	default:
		return false
	}
}

func (s *Service) enrichNetworks(list []Network) []Network {
	if len(list) == 0 {
		return list
	}
	details := s.fetchNetworkInspectMap()
	for i := range list {
		list[i].IsSystem = isSystemNetwork(list[i].Name)
		if d, ok := details[list[i].ID]; ok {
			list[i].Subnet = d.Subnet
			list[i].Gateway = d.Gateway
			list[i].Endpoints = d.Endpoints
			list[i].ContainerCount = len(d.Endpoints)
			list[i].InUse = len(d.Endpoints) > 0
		}
	}
	return list
}

type networkInspectDetail struct {
	Subnet    string
	Gateway   string
	Endpoints []NetworkEndpoint
}

func (s *Service) fetchNetworkInspectMap() map[string]networkInspectDetail {
	out := map[string]networkInspectDetail{}
	if !s.dockerAvailable() {
		return out
	}
	idsRaw, err := runDocker("network", "ls", "-q")
	if err != nil {
		return out
	}
	ids := strings.Fields(strings.TrimSpace(string(idsRaw)))
	if len(ids) == 0 {
		return out
	}
	args := append([]string{"network", "inspect", "--format", "{{json .}}"}, ids...)
	inspectOut, err := runDocker(args...)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(inspectOut), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row struct {
			ID    string `json:"Id"`
			IPAM  struct {
				Config []struct {
					Subnet  string `json:"Subnet"`
					Gateway string `json:"Gateway"`
				} `json:"Config"`
			} `json:"IPAM"`
			Containers map[string]struct {
				Name        string `json:"Name"`
				IPv4Address string `json:"IPv4Address"`
			} `json:"Containers"`
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		d := networkInspectDetail{Endpoints: []NetworkEndpoint{}}
		if len(row.IPAM.Config) > 0 {
			d.Subnet = row.IPAM.Config[0].Subnet
			d.Gateway = row.IPAM.Config[0].Gateway
		}
		for _, c := range row.Containers {
			name := strings.TrimPrefix(c.Name, "/")
			ipv4 := strings.Split(c.IPv4Address, "/")[0]
			d.Endpoints = append(d.Endpoints, NetworkEndpoint{Name: name, IPv4: ipv4})
		}
		shortID := row.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		out[row.ID] = d
		out[shortID] = d
	}
	return out
}

func (s *Service) CreateNetwork(opts CreateNetworkOpts) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	name := strings.TrimSpace(opts.Name)
	if name == "" {
		return fmt.Errorf("network name required")
	}
	args := []string{"network", "create"}
	if d := strings.TrimSpace(opts.Driver); d != "" {
		args = append(args, "--driver", d)
	}
	if subnet := strings.TrimSpace(opts.Subnet); subnet != "" {
		args = append(args, "--subnet", subnet)
	}
	if gateway := strings.TrimSpace(opts.Gateway); gateway != "" {
		args = append(args, "--gateway", gateway)
	}
	args = append(args, name)
	_, err := runDocker(args...)
	return err
}

func (s *Service) RemoveNetwork(id string) error {
	if err := s.dockerOK(); err != nil {
		return err
	}
	_, err := runDocker("network", "rm", id)
	return err
}

func (s *Service) PruneNetworks() (string, error) {
	if err := s.dockerOK(); err != nil {
		return "", err
	}
	out, err := runDocker("network", "prune", "-f")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
