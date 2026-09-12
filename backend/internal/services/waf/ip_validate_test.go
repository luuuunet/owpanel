package waf

import (
	"net"
	"strings"
	"testing"
)

func TestParseBlockEntryIPAndCIDR(t *testing.T) {
	ip, err := parseBlockEntry("8.8.8.8")
	if err != nil || ip.Canonical != "8.8.8.8" || ip.Kind != blockKindIP {
		t.Fatalf("ip: %#v %v", ip, err)
	}
	cidr, err := parseBlockEntry("10.0.0.1/24")
	if err != nil || cidr.Canonical != "10.0.0.0/24" {
		t.Fatalf("cidr: %#v %v", cidr, err)
	}
}

func TestParseBlockEntryRange(t *testing.T) {
	entry, err := parseBlockEntry("192.168.1.10 - 192.168.1.12")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Kind != blockKindRange {
		t.Fatalf("kind=%s", entry.Kind)
	}
	if entry.Canonical != "192.168.1.10-192.168.1.12" {
		t.Fatalf("canonical=%s", entry.Canonical)
	}
	if len(entry.Keys) == 0 {
		t.Fatal("expected cidr keys")
	}
	if !strings.Contains(strings.Join(entry.Keys, ","), "192.168.1.10") {
		t.Fatalf("keys=%v", entry.Keys)
	}
}

func TestRangeToCIDRsAligned(t *testing.T) {
	start, ok1 := ipToUint32(net.ParseIP("10.0.0.0"))
	end, ok2 := ipToUint32(net.ParseIP("10.0.0.255"))
	if !ok1 || !ok2 {
		t.Fatal("parse")
	}
	got := rangeToCIDRs(start, end)
	if len(got) != 1 || got[0] != "10.0.0.0/24" {
		t.Fatalf("got %v", got)
	}
}

func TestParseBlockEntryASN(t *testing.T) {
	entry, err := parseBlockEntry("AS57523")
	if err != nil || entry.ASN != 57523 || entry.Kind != blockKindASN {
		t.Fatalf("%#v %v", entry, err)
	}
}

func TestParseBlockEntryRejectsJunk(t *testing.T) {
	if _, err := parseBlockEntry("not-an-ip"); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateNginxKeepsIfOutOfHTTP(t *testing.T) {
	s := &Service{dataDir: t.TempDir(), confDir: t.TempDir()}
	cfg := defaultConfig(s.dataDir)
	httpConf := s.generateNginx(&cfg, nil)
	if strings.Contains(httpConf, "if ($op_deny") {
		t.Fatalf("http conf must not contain server if-directives:\n%s", httpConf)
	}
	snippet := s.generateServerSnippet(&cfg)
	if !strings.Contains(snippet, "if ($op_deny_ip)") {
		t.Fatalf("server snippet missing ip deny:\n%s", snippet)
	}
}
