package waf

import (
	"fmt"
	"net"
	"strings"
)

// validateMapIP accepts a single IP or CIDR suitable for nginx geo/map keys.
func validateMapIP(raw string) (string, error) {
	ip := strings.TrimSpace(raw)
	if ip == "" {
		return "", fmt.Errorf("ip is required")
	}
	if strings.ContainsAny(ip, ";#\n\r\t \"'") {
		return "", fmt.Errorf("非法 IP/CIDR: %s", raw)
	}
	if strings.Contains(ip, "/") {
		_, network, err := net.ParseCIDR(ip)
		if err != nil {
			return "", fmt.Errorf("无效 CIDR: %s", raw)
		}
		return network.String(), nil
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", fmt.Errorf("无效 IP: %s", raw)
	}
	return parsed.String(), nil
}
