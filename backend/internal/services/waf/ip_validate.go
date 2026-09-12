package waf

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type blockKind string

const (
	blockKindIP    blockKind = "ip"
	blockKindCIDR  blockKind = "cidr"
	blockKindRange blockKind = "range"
	blockKindASN   blockKind = "asn"
)

type blockEntry struct {
	Canonical string
	Kind      blockKind
	ASN       int
	Keys      []string
}

// parseBlockEntry accepts a single IP, CIDR, IPv4 range, or ASN.
// Examples: 1.2.3.4 | 10.0.0.0/24 | 192.168.1.10-192.168.1.50 | AS57523
func parseBlockEntry(raw string) (*blockEntry, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ",")
	if raw == "" {
		return nil, fmt.Errorf("empty")
	}
	if strings.ContainsAny(raw, ";#\n\r\t\"'") {
		return nil, fmt.Errorf("非法条目: %s", raw)
	}

	upper := strings.ToUpper(raw)
	if strings.HasPrefix(upper, "AS") && !strings.Contains(raw, ".") {
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(upper, "AS")))
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("无效 ASN: %s", raw)
		}
		return &blockEntry{Canonical: "AS" + strconv.Itoa(n), Kind: blockKindASN, ASN: n}, nil
	}

	if strings.Contains(raw, "/") {
		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return nil, fmt.Errorf("无效 CIDR: %s", raw)
		}
		cidr := network.String()
		return &blockEntry{Canonical: cidr, Kind: blockKindCIDR, Keys: []string{cidr}}, nil
	}

	if start, end, ok := splitIPv4Range(raw); ok {
		if start > end {
			start, end = end, start
		}
		if end-start > 1<<20 { // more than ~1 million addresses — require CIDR
			return nil, fmt.Errorf("IP 段过大，请改用 CIDR（例如 10.0.0.0/8）")
		}
		keys := rangeToCIDRs(start, end)
		canonical := uint32ToIP(start).String() + "-" + uint32ToIP(end).String()
		return &blockEntry{Canonical: canonical, Kind: blockKindRange, Keys: keys}, nil
	}

	parsed := net.ParseIP(raw)
	if parsed == nil {
		return nil, fmt.Errorf("无效 IP/IP段/ASN: %s", raw)
	}
	return &blockEntry{Canonical: parsed.String(), Kind: blockKindIP, Keys: []string{parsed.String()}}, nil
}

func validateMapIP(raw string) (string, error) {
	entry, err := parseBlockEntry(raw)
	if err != nil {
		return "", err
	}
	return entry.Canonical, nil
}

func splitIPv4Range(raw string) (uint32, uint32, bool) {
	// Support 1.2.3.4-1.2.3.20 and 1.2.3.4 - 1.2.3.20
	parts := strings.Split(raw, "-")
	if len(parts) != 2 {
		return 0, 0, false
	}
	a := net.ParseIP(strings.TrimSpace(parts[0]))
	b := net.ParseIP(strings.TrimSpace(parts[1]))
	if a == nil || b == nil {
		return 0, 0, false
	}
	start, ok1 := ipToUint32(a)
	end, ok2 := ipToUint32(b)
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	return start, end, true
}

func ipToUint32(ip net.IP) (uint32, bool) {
	v4 := ip.To4()
	if v4 == nil {
		return 0, false
	}
	return binary.BigEndian.Uint32(v4), true
}

func uint32ToIP(n uint32) net.IP {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, n)
	return net.IP(b)
}

func rangeToCIDRs(start, end uint32) []string {
	var out []string
	cur := uint64(start)
	last := uint64(end)
	for cur <= last {
		maxAlign := uint64(1) << 32
		if cur > 0 {
			maxAlign = cur & -cur
		}
		remain := last - cur + 1
		size := largestPow2(minU64(maxAlign, remain))
		prefix := 32
		for s := size; s > 1; s >>= 1 {
			prefix--
		}
		out = append(out, fmt.Sprintf("%s/%d", uint32ToIP(uint32(cur)).String(), prefix))
		next := cur + size
		if next <= cur {
			break
		}
		cur = next
	}
	return out
}

func minU64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func largestPow2(n uint64) uint64 {
	if n == 0 {
		return 1
	}
	p := uint64(1)
	for p<<1 > p && p<<1 <= n {
		p <<= 1
	}
	return p
}
