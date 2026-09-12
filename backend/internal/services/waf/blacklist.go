package waf

import (
	"fmt"
	"os"
	"strings"

	"github.com/luuuunet/owpanel/internal/models"
)

type QuickBlockResult struct {
	Blocked int      `json:"blocked"`
	ASNs    int      `json:"asns"`
	Failed  []string `json:"failed"`
}

func (s *Service) ListBlacklist() ([]models.IPBlacklist, error) {
	var list []models.IPBlacklist
	return list, s.db.Where("enabled = ?", true).Order("id desc").Find(&list).Error
}

func (s *Service) AddBlacklist(ip, reason, source string) (*models.IPBlacklist, error) {
	entry, err := parseBlockEntry(ip)
	if err != nil {
		return nil, err
	}
	if entry.Kind == blockKindASN {
		if err := s.ensureASNPrefixes(entry.ASN); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Canonical, err)
		}
	}
	row := models.IPBlacklist{IP: entry.Canonical, Reason: reason, Source: source, Enabled: true}
	if source == "" {
		row.Source = "manual"
	}
	if err := s.db.Where("ip = ?", entry.Canonical).Assign(row).FirstOrCreate(&row).Error; err != nil {
		return nil, err
	}
	_ = s.writeBlacklistMap()
	_ = s.reloadNginxMaps()
	return &row, nil
}

func (s *Service) RemoveBlacklist(id uint) error {
	if err := s.db.Delete(&models.IPBlacklist{}, id).Error; err != nil {
		return err
	}
	_ = s.writeBlacklistMap()
	_ = s.reloadNginxMaps()
	return nil
}

func (s *Service) ImportBlacklist(ips []string, reason string) (int, error) {
	res := s.QuickBlock(strings.Join(ips, "\n"), reason, "batch")
	return res.Blocked, nil
}

func (s *Service) QuickBlock(text, reason, source string) *QuickBlockResult {
	out := &QuickBlockResult{}
	if source == "" {
		source = "manual"
	}
	if reason == "" {
		reason = "quick-block"
	}
	for _, line := range splitBlockLines(text) {
		entry, err := parseBlockEntry(line)
		if err != nil {
			out.Failed = append(out.Failed, line+": "+err.Error())
			continue
		}
		if entry.Kind == blockKindASN {
			if err := s.ensureASNPrefixes(entry.ASN); err != nil {
				out.Failed = append(out.Failed, entry.Canonical+": "+err.Error())
				continue
			}
			out.ASNs++
		}
		row := models.IPBlacklist{IP: entry.Canonical, Reason: reason, Source: source, Enabled: true}
		if err := s.db.Where("ip = ?", entry.Canonical).Assign(row).FirstOrCreate(&row).Error; err != nil {
			out.Failed = append(out.Failed, entry.Canonical+": "+err.Error())
			continue
		}
		out.Blocked++
	}
	_ = s.writeBlacklistMap()
	_ = s.reloadNginxMaps()
	return out
}

func splitBlockLines(text string) []string {
	var out []string
	for _, line := range strings.FieldsFunc(text, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	}) {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func (s *Service) writeBlacklistMap() error {
	list, err := s.ListBlacklist()
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# OWPanel IP/CIDR/range/ASN blacklist — auto generated\n")
	seen := map[string]bool{}
	for _, item := range list {
		keys := s.expandStoredBlock(item.IP)
		for _, key := range keys {
			if seen[key] {
				continue
			}
			seen[key] = true
			b.WriteString(fmt.Sprintf("%s 1;\n", key))
		}
	}
	return os.WriteFile(s.BlacklistMapPath(), []byte(b.String()), 0644)
}

func (s *Service) expandStoredBlock(stored string) []string {
	entry, err := parseBlockEntry(stored)
	if err != nil {
		return nil
	}
	if entry.Kind == blockKindASN {
		prefixes, err := s.cachedASNPrefixes(entry.ASN)
		if err != nil {
			return nil
		}
		var keys []string
		for _, p := range prefixes {
			if _, e := parseBlockEntry(p); e == nil {
				keys = append(keys, p)
			}
		}
		return keys
	}
	return entry.Keys
}
