package website

import (
	"strconv"
	"strings"

	"github.com/luuuunet/owpanel/internal/services/domaincheck"
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
	return domaincheck.HostOnly(d)
}

// confFileName returns a safe nginx/apache vhost filename for a domain.
func confFileName(domain string) string {
	d := domaincheck.HostOnly(domain)
	d = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			return r
		default:
			return '_'
		}
	}, d)
	if d == "" || d == "." || d == ".." {
		d = "site"
	}
	return d + ".conf"
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
