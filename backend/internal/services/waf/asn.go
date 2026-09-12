package waf

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Service) ASNCacheDir() string {
	dir := filepath.Join(s.confDir, "asn-cache")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func (s *Service) ensureASNPrefixes(asn int) error {
	_, err := s.fetchASNPrefixes(asn)
	return err
}

func (s *Service) cachedASNPrefixes(asn int) ([]string, error) {
	path := filepath.Join(s.ASNCacheDir(), fmt.Sprintf("AS%d.txt", asn))
	if fi, err := os.Stat(path); err == nil && time.Since(fi.ModTime()) < 24*time.Hour {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			return splitPrefixLines(string(data)), nil
		}
	}
	return s.fetchASNPrefixes(asn)
}

func (s *Service) fetchASNPrefixes(asn int) ([]string, error) {
	prefixes, err := fetchRIPEASPrefixes(asn)
	if err != nil || len(prefixes) == 0 {
		alt, altErr := fetchIPVerseASNPrefixes(asn)
		if altErr != nil {
			if err != nil {
				return nil, err
			}
			return nil, altErr
		}
		prefixes = alt
	}
	if len(prefixes) == 0 {
		return nil, fmt.Errorf("AS%d 未解析到网段", asn)
	}
	path := filepath.Join(s.ASNCacheDir(), fmt.Sprintf("AS%d.txt", asn))
	_ = os.WriteFile(path, []byte(strings.Join(prefixes, "\n")+"\n"), 0644)
	return prefixes, nil
}

func splitPrefixLines(raw string) []string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if _, _, err := net.ParseCIDR(line); err == nil {
			out = append(out, line)
		}
	}
	return out
}

func fetchRIPEASPrefixes(asn int) ([]string, error) {
	url := fmt.Sprintf("https://stat.ripe.net/data/announced-prefixes/data.json?resource=AS%d", asn)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("RIPE HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Data struct {
			Prefixes []struct {
				Prefix string `json:"prefix"`
			} `json:"prefixes"`
		} `json:"data"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	var out []string
	for _, p := range payload.Data.Prefixes {
		if _, _, err := net.ParseCIDR(p.Prefix); err == nil {
			out = append(out, p.Prefix)
		}
	}
	return out, nil
}

func fetchIPVerseASNPrefixes(asn int) ([]string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/ipverse/asn-ip/master/as/%d/aggregated.json", asn)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ipverse HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Subnets struct {
			IPv4 []string `json:"ipv4"`
			IPv6 []string `json:"ipv6"`
		} `json:"subnets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, err
	}
	var valid []string
	for _, p := range append(append([]string{}, payload.Subnets.IPv4...), payload.Subnets.IPv6...) {
		if _, _, err := net.ParseCIDR(p); err == nil {
			valid = append(valid, p)
		}
	}
	return valid, nil
}
