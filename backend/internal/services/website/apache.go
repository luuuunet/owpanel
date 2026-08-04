package website

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/luuuunet/owpanel/internal/models"
	"github.com/luuuunet/owpanel/internal/services/php"
	sslpkg "github.com/luuuunet/owpanel/internal/services/ssl"
)

func (s *Service) writeApacheVhost(site *models.Website) (string, error) {
	if site.Status == "stopped" {
		return s.writeStoppedVhost(site, "apache")
	}
	confDir := s.vhostDir("apache")
	_ = os.MkdirAll(confDir, 0755)
	confPath := filepath.Join(confDir, confFileName(site.Domain))
	root := filepath.ToSlash(site.RootPath)

	entries := s.allDomainEntries(site)
	if len(entries) == 0 {
		entries = []domainEntry{{Host: site.Domain, Port: site.Port}}
	}
	groups := groupByPort(entries)

	certOK := false
	if site.SSL {
		_, _, certOK = sslpkg.CertPaths(s.dataDir, site.Domain)
		if !certOK {
			log.Printf("[website] SSL enabled for %s but certificate files missing; generating HTTP-only Apache vhost", site.Domain)
		}
	}
	forceHTTPS := site.SSL && site.ForceHTTPS && certOK

	var blocks []string
	for port, hosts := range groups {
		block, err := s.apacheVirtualHost(site, root, port, hosts, false, "", "", forceHTTPS)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, block)
	}
	if site.SSL && certOK {
		fullchain, privkey, ok := sslpkg.CertPaths(s.dataDir, site.Domain)
		if ok {
			var hosts []string
			for _, e := range entries {
				hosts = append(hosts, e.Host)
			}
			if len(hosts) == 0 {
				hosts = []string{site.Domain}
			}
			sslBlock, err := s.apacheVirtualHost(site, root, 443, hosts, true, filepath.ToSlash(fullchain), filepath.ToSlash(privkey), false)
			if err != nil {
				return "", err
			}
			blocks = append(blocks, sslBlock)
		}
	}
	content := fmt.Sprintf("# OWPanel — %s\n%s\n", site.Domain, strings.Join(blocks, "\n"))
	if err := os.WriteFile(confPath, []byte(content), 0644); err != nil {
		return "", err
	}
	return confPath, nil
}

func apacheServerNames(hosts []string) (serverName, serverAlias string) {
	if len(hosts) == 0 {
		return "", ""
	}
	serverName = hosts[0]
	if len(hosts) > 1 {
		serverAlias = strings.Join(hosts[1:], " ")
	}
	return serverName, serverAlias
}

func (s *Service) apacheVirtualHost(site *models.Website, root string, port int, hosts []string, ssl bool, fullchain, privkey string, forceHTTPS bool) (string, error) {
	if site.Status == "stopped" {
		return fmt.Sprintf(`# Site stopped: %s
<VirtualHost *:%d>
    ServerName %s
    DocumentRoot "%s"
    ErrorDocument 403 /503.html
</VirtualHost>`, site.Domain, port, site.Domain, root), nil
	}

	serverName, serverAlias := apacheServerNames(hosts)
	aliasLine := ""
	if serverAlias != "" {
		aliasLine = fmt.Sprintf("\n    ServerAlias %s", serverAlias)
	}

	logSuffix := ""
	if ssl {
		logSuffix = "_ssl"
	}
	accessLog := filepath.ToSlash(filepath.Join(s.dataDir, "logs", site.Domain+logSuffix+"_access.log"))
	errorLog := filepath.ToSlash(filepath.Join(s.dataDir, "logs", site.Domain+logSuffix+"_error.log"))

	indexLine := strings.TrimSpace(site.IndexFiles)
	if indexLine == "" {
		indexLine = "index.html"
		if site.PHP && site.PhpVersion != "static" {
			indexLine = "index.php index.html"
		}
	}

	phpBlock := ""
	if site.PHP && site.PhpVersion != "" && site.PhpVersion != "static" {
		handler := php.ApacheProxyFCGI(site.PhpVersion)
		phpBlock = fmt.Sprintf(`
    <FilesMatch \.php$>
        SetHandler "%s"
    </FilesMatch>`, handler)
	}

	rewriteBlock := ""
	if rules := strings.TrimSpace(site.RewriteRules); rules != "" {
		rewriteBlock = "\n    " + strings.ReplaceAll(rules, "\n", "\n    ") + "\n"
	}

	// Cloudflare-aware HTTP→HTTPS (skip when already proxied as HTTPS).
	if forceHTTPS && !ssl {
		rewriteBlock += `
    RewriteEngine On
    RewriteCond %{HTTP:CF-Ray} ^$
    RewriteCond %{HTTP:X-Forwarded-Proto} !https [NC]
    RewriteRule ^ https://%{HTTP_HOST}%{REQUEST_URI} [L,R=301]`
	}

	crossSiteBlock := ""
	if site.CrossSiteProtectEnabled {
		crossSiteBlock = `
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set Content-Security-Policy "frame-ancestors 'self'"
    Header always set Referrer-Policy "strict-origin-when-cross-origin"`
	}

	proxyBlock := ""
	if proxy := strings.TrimSpace(site.ProxyPass); proxy != "" {
		proxyBlock = fmt.Sprintf(`
    ProxyPreserveHost On
    ProxyPass / %s/
    ProxyPassReverse / %s/`, proxy, proxy)
	} else if redirect := strings.TrimSpace(site.RedirectURL); redirect != "" {
		proxyBlock = fmt.Sprintf(`
    RedirectMatch 301 ^/(.*)$ %s`, redirect)
	}

	sslBlock := ""
	if ssl {
		sslBlock = fmt.Sprintf(`
    SSLEngine on
    SSLCertificateFile "%s"
    SSLCertificateKeyFile "%s"`, fullchain, privkey)
	}

	return fmt.Sprintf(`<VirtualHost *:%d>
    ServerName %s%s
    DocumentRoot "%s"
    DirectoryIndex %s
    ErrorLog "%s"
    CustomLog "%s" combined%s
%s%s%s%s
    <Directory "%s">
        Options Indexes FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>
</VirtualHost>`, port, serverName, aliasLine, root, indexLine, errorLog, accessLog, sslBlock, rewriteBlock, crossSiteBlock, proxyBlock, phpBlock, root), nil
}

func (s *Service) removeApacheVhost(domain string) {
	confPath := filepath.Join(s.vhostDir("apache"), confFileName(domain))
	_ = os.Remove(confPath)
}
