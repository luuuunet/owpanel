package website

import (
	"strconv"
	"strings"
)

type domainEntry struct {
	Host string
	Port int
}

func parseDomainLine(raw string) domainEntry {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return domainEntry{}
	}
	port := 80
	host := raw
	if strings.Contains(host, ":") {
		parts := strings.Split(host, ":")
		host = strings.TrimSpace(parts[0])
		if len(parts) > 1 {
			if p, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && p > 0 {
				port = p
			}
		}
	}
	host = normalizeDomain(host)
	return domainEntry{Host: host, Port: port}
}

func parseDomainList(raw string) []domainEntry {
	if raw == "" {
		return nil
	}
	var out []domainEntry
	seen := map[string]bool{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ','
	}) {
		entry := parseDomainLine(part)
		if entry.Host == "" || seen[entry.Host] {
			continue
		}
		seen[entry.Host] = true
		out = append(out, entry)
	}
	return out
}

func normalizeDomain(d string) string {
	d = strings.TrimSpace(strings.ToLower(d))
	d = strings.TrimPrefix(d, "http://")
	d = strings.TrimPrefix(d, "https://")
	if idx := strings.Index(d, "/"); idx >= 0 {
		d = d[:idx]
	}
	return d
}

func groupByPort(entries []domainEntry) map[int][]string {
	groups := map[int][]string{}
	for _, e := range entries {
		groups[e.Port] = append(groups[e.Port], e.Host)
	}
	return groups
}

func sanitizeName(domain string) string {
	s := strings.NewReplacer(".", "_", "-", "_", ":", "_", "*", "w").Replace(domain)
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}
