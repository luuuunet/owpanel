package website

import (
	"fmt"
	"log"
	"strings"

	"github.com/luuuunet/owpanel/internal/models"
	"github.com/luuuunet/owpanel/internal/services/ssl"
)

func (s *Service) DeploySSLForDomain(domain string) error {
	domain = strings.TrimSpace(strings.ToLower(domain))
	var site models.Website
	if err := s.db.Where("domain = ?", domain).First(&site).Error; err != nil {
		return fmt.Errorf("未找到域名 %s 对应的网站", domain)
	}
	fc, _, ok := ssl.CertPaths(s.dataDir, domain)
	if !ok {
		return fmt.Errorf("证书文件不存在: %s", domain)
	}
	_ = fc
	// Behind Cloudflare orange-cloud, Forced HTTPS on origin causes Flexible loops / empty HTTP.
	forceHTTPS := !s.siteHasProxiedDNS(&site)
	if err := s.db.Model(&site).Updates(map[string]interface{}{
		"ssl": true, "force_https": forceHTTPS,
	}).Error; err != nil {
		return err
	}
	site.SSL = true
	site.ForceHTTPS = forceHTTPS
	return s.applyVhost(&site)
}

// siteHasProxiedDNS reports whether this site has any Cloudflare-proxied DNS records.
func (s *Service) siteHasProxiedDNS(site *models.Website) bool {
	if site == nil || site.ID == 0 || s.db == nil {
		return false
	}
	var n int64
	if err := s.db.Model(&models.DNSRecord{}).
		Where("website_id = ? AND proxied = ?", site.ID, true).
		Count(&n).Error; err != nil {
		return false
	}
	return n > 0
}

// collectSANDomains returns all non-primary bound aliases for certificate SAN coverage.
func (s *Service) collectSANDomains(site *models.Website) string {
	if site == nil {
		return ""
	}
	if len(site.Aliases) == 0 {
		s.db.Where("website_id = ?", site.ID).Find(&site.Aliases)
	}
	seen := map[string]bool{strings.ToLower(site.Domain): true}
	var extras []string
	for _, a := range site.Aliases {
		host := strings.TrimSpace(strings.ToLower(a.Domain))
		if host == "" || seen[host] || a.Type == "primary" {
			continue
		}
		seen[host] = true
		extras = append(extras, host)
	}
	return strings.Join(extras, ",")
}

func mergeSANLists(manual, auto string) string {
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
		}) {
			h := strings.TrimSpace(strings.ToLower(part))
			if h == "" || seen[h] {
				continue
			}
			seen[h] = true
			out = append(out, h)
		}
	}
	add(manual)
	add(auto)
	return strings.Join(out, ",")
}

func (s *Service) certEmailForSite(site *models.Website) string {
	var cert models.SSLCertificate
	if err := s.db.Where("domain = ? AND status = ?", site.Domain, "active").Order("id desc").First(&cert).Error; err == nil {
		if e := strings.TrimSpace(cert.Email); e != "" {
			return e
		}
	}
	if err := s.db.Where("domain = ?", site.Domain).Order("id desc").First(&cert).Error; err == nil {
		return strings.TrimSpace(cert.Email)
	}
	return ""
}

// ReissueSSLWithAliases re-issues the site certificate including all bound aliases as SANs.
// Needed after adding domains behind Cloudflare Full (strict), otherwise origin TLS fails (525/526).
func (s *Service) ReissueSSLWithAliases(siteID uint) error {
	site, err := s.Get(siteID)
	if err != nil {
		return err
	}
	if !site.SSL {
		return nil
	}
	email := s.certEmailForSite(site)
	return s.IssueSSL(siteID, email, s.collectSANDomains(site), true)
}

func (s *Service) IssueSSL(siteID uint, email string, sanDomains string, deploy bool) error {
	site, err := s.Get(siteID)
	if err != nil {
		return err
	}
	sanDomains = mergeSANLists(sanDomains, s.collectSANDomains(site))
	if email == "" {
		email = s.certEmailForSite(site)
	}
	autoRenew := true
	sslSvc := ssl.NewService(s.db, s.dataDir)
	sslSvc.SetDeployHook(s.DeploySSLForDomain)
	cert, err := sslSvc.Issue(&ssl.IssueRequest{
		Domain:     site.Domain,
		SanDomains: sanDomains,
		Webroot:    site.RootPath,
		Email:      email,
		AutoRenew:  &autoRenew,
		Deploy:     deploy,
	})
	if err != nil {
		return err
	}
	if cert.Status != "active" {
		return fmt.Errorf("证书申请未完成")
	}
	if !deploy {
		if err := s.db.Model(site).Update("ssl", true).Error; err != nil {
			return err
		}
		site.SSL = true
		return s.applyVhost(site)
	}
	return nil
}

func (s *Service) reissueSSLAfterDomainChange(siteID uint) {
	go func() {
		if err := s.ReissueSSLWithAliases(siteID); err != nil {
			log.Printf("[website] reissue SSL after domain change (site %d): %v", siteID, err)
		} else {
			log.Printf("[website] reissued SSL with alias SANs for site %d", siteID)
		}
	}()
}
