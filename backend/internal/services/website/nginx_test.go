package website

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luuuunet/owpanel/internal/models"
)

func TestRenderServerBlockAccessLogPaths(t *testing.T) {
	s := &Service{dataDir: "/opt/owpanel/data"}
	site := &models.Website{
		Domain:     "example.com",
		RootPath:   "/opt/owpanel/data/wwwroot/example.com/public",
		Port:       80,
		PHP:        true,
		PhpVersion: "8.4",
	}
	features := &nginxFeatureBlocks{}
	block, err := s.renderServerBlock(site, site.RootPath, 80, []string{site.Domain}, sslOpts{}, features, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(block, "access_log ;") {
		t.Fatalf("malformed access_log in vhost:\n%s", block)
	}
	if strings.Contains(block, "%!(EXTRA") {
		t.Fatalf("fmt placeholder mismatch in vhost:\n%s", block)
	}
	wantAccess := "/opt/owpanel/data/logs/example.com_access.log"
	wantError := "/opt/owpanel/data/logs/example.com_error.log"
	if !strings.Contains(block, "access_log "+wantAccess+";") {
		t.Fatalf("missing access log path, got:\n%s", block)
	}
	if !strings.Contains(block, "error_log "+wantError+";") {
		t.Fatalf("missing error log path, got:\n%s", block)
	}
}

func TestForceHTTPSKeepsTryFilesBehindCloudflare(t *testing.T) {
	s := &Service{dataDir: "/opt/owpanel/data"}
	site := &models.Website{
		Domain:     "example.com",
		RootPath:   "/opt/owpanel/data/wwwroot/example.com/public",
		Port:       80,
		PHP:        true,
		PhpVersion: "8.4",
		SSL:        true,
		ForceHTTPS: true,
	}
	features := &nginxFeatureBlocks{}
	block, err := s.renderServerBlock(site, site.RootPath, 80, []string{site.Domain}, sslOpts{}, features, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"$http_cf_ray",
		"try_files",
		"/.well-known/acme-challenge/",
		"return 301 https://$host$request_uri",
	} {
		if !strings.Contains(block, want) {
			t.Fatalf("missing %q in force-HTTPS HTTP block:\n%s", want, block)
		}
	}
}

func TestApacheServerNames(t *testing.T) {
	name, alias := apacheServerNames([]string{"a.com", "www.a.com", "b.a.com"})
	if name != "a.com" {
		t.Fatalf("ServerName=%q", name)
	}
	if alias != "www.a.com b.a.com" {
		t.Fatalf("ServerAlias=%q", alias)
	}
}

func TestSslHostnamesIncludesAllAliases(t *testing.T) {
	site := &models.Website{
		Domain: "example.com",
		Aliases: []models.WebsiteAlias{
			{Domain: "example.com", Type: "primary", Port: 80},
			{Domain: "www.example.com", Type: "alias", Port: 80},
			{Domain: "tmt1.example.com", Type: "alias", Port: 80},
		},
	}
	hosts := sslHostnames(site)
	joined := strings.Join(hosts, " ")
	for _, want := range []string{"example.com", "www.example.com", "tmt1.example.com"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %s in %v", want, hosts)
		}
	}
}

func TestResolveSslCertFallsBackToPrimaryWhenAliasHasNone(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "ssl", "example.com")
	if err := os.MkdirAll(certDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "fullchain.pem"), []byte("c"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "privkey.pem"), []byte("k"), 0644); err != nil {
		t.Fatal(err)
	}
	s := &Service{dataDir: dir}
	site := &models.Website{
		Domain: "example.com",
		Aliases: []models.WebsiteAlias{
			{Domain: "tmt1.example.com", Type: "alias"},
		},
	}
	_, _, ok := s.resolveSslCert(site, "example.com")
	if !ok {
		t.Fatal("expected to reuse primary certificate for aliases")
	}
}
