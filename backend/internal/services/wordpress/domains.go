package wordpress

import (
	"fmt"
	"log"
	"strings"

	"github.com/luuuunet/owpanel/internal/models"
	"github.com/luuuunet/owpanel/internal/services/domaincheck"
	"github.com/luuuunet/owpanel/internal/services/sitepurge"
	"gorm.io/gorm"
)

func (s *Service) loadDomains(site *models.WordPressSite) {
	s.ensurePrimaryDomain(site)
	var domains []models.WordPressDomain
	s.db.Where("site_id = ? AND enabled = ?", site.ID, true).Order("type desc, id asc").Find(&domains)
	site.Domains = domains
}

func (s *Service) ensurePrimaryDomain(site *models.WordPressSite) {
	if site.ID == 0 || site.Domain == "" {
		return
	}
	var existing models.WordPressDomain
	err := s.db.Where("site_id = ? AND type = ?", site.ID, "primary").First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		d := models.WordPressDomain{
			SiteID: site.ID, Domain: site.Domain, Type: "primary", Enabled: true,
		}
		_ = s.db.Create(&d).Error
	} else if err == nil && existing.Domain != site.Domain {
		s.db.Model(&existing).Update("domain", site.Domain)
	}
}

func (s *Service) allServerNames(site *models.WordPressSite) []string {
	s.loadDomains(site)
	seen := map[string]bool{}
	var names []string
	add := func(d string) {
		d = strings.TrimSpace(strings.ToLower(d))
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		names = append(names, d)
	}
	add(site.Domain)
	for _, d := range site.Domains {
		if d.Enabled {
			add(d.Domain)
		}
	}
	if len(names) == 0 {
		add(site.Domain)
	}
	return names
}

func (s *Service) ListDomains(siteID uint) ([]models.WordPressDomain, error) {
	site, err := s.Get(siteID)
	if err != nil {
		return nil, err
	}
	return site.Domains, nil
}

func (s *Service) AddDomain(siteID uint, domain string) (*models.WordPressDomain, error) {
	domain = normalizeDomain(domain)
	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}
	site, err := s.Get(siteID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(domaincheck.HostOnly(domain), domaincheck.HostOnly(site.Domain)) {
		return nil, fmt.Errorf("domain is already the primary domain")
	}
	if err := domaincheck.AssertAvailable(s.db, []string{domain}, domaincheck.Scope{IgnoreWPSiteID: siteID}); err != nil {
		return nil, err
	}
	entry := models.WordPressDomain{
		SiteID: siteID, Domain: domain, Type: "alias", Enabled: true,
	}
	if err := s.db.Create(&entry).Error; err != nil {
		return nil, err
	}
	if err := s.syncWebsiteAlias(site, domain); err != nil {
		return nil, err
	}
	if err := s.regenerateVhost(siteID); err != nil {
		return nil, err
	}
	if err := s.reissueSSLAfterDomainChange(site); err != nil {
		return &entry, fmt.Errorf("域名已绑定，但更新 SSL 证书失败（可能出现 525）: %w", err)
	}
	return &entry, nil
}

func (s *Service) RemoveDomain(siteID, domainID uint) error {
	var entry models.WordPressDomain
	if err := s.db.Where("id = ? AND site_id = ?", domainID, siteID).First(&entry).Error; err != nil {
		return err
	}
	if entry.Type == "primary" {
		return fmt.Errorf("cannot remove primary domain")
	}
	if err := s.db.Delete(&entry).Error; err != nil {
		return err
	}
	sitepurge.Domains(s.db, []string{entry.Domain}, sitepurge.Options{DataDir: s.dataDir})
	if err := s.regenerateVhost(siteID); err != nil {
		return err
	}
	site, _ := s.Get(siteID)
	if site != nil {
		if err := s.reissueSSLAfterDomainChange(site); err != nil {
			return fmt.Errorf("域名已删除，但更新 SSL 证书失败: %w", err)
		}
	}
	return nil
}

func (s *Service) BindDomains(siteID uint, domains []string) error {
	site, err := s.Get(siteID)
	if err != nil {
		return err
	}
	for _, d := range domains {
		d = normalizeDomain(d)
		if d == "" || strings.EqualFold(d, site.Domain) {
			continue
		}
		if err := domaincheck.AssertAvailable(s.db, []string{d}, domaincheck.Scope{IgnoreWPSiteID: siteID}); err != nil {
			return err
		}
		entry := models.WordPressDomain{SiteID: siteID, Domain: d, Type: "alias", Enabled: true}
		if err := s.db.Create(&entry).Error; err != nil {
			return err
		}
		_ = s.syncWebsiteAlias(site, d)
	}
	if err := s.regenerateVhost(siteID); err != nil {
		return err
	}
	site, _ = s.Get(siteID)
	if site != nil {
		if err := s.reissueSSLAfterDomainChange(site); err != nil {
			return fmt.Errorf("域名已绑定，但更新 SSL 证书失败（可能出现 525）: %w", err)
		}
	}
	return nil
}

func (s *Service) reissueSSLAfterDomainChange(site *models.WordPressSite) error {
	// Skip only when origin SSL is off. If SSL is enabled (incl. CDN + Full Strict),
	// aliases must be on the origin certificate SAN list.
	if site == nil || !site.SSL {
		return nil
	}
	if err := s.IssueSSLForSite(site.ID, ""); err != nil {
		log.Printf("[wordpress] reissue SSL after domain change (site %d): %v", site.ID, err)
		return err
	}
	log.Printf("[wordpress] reissued SSL with alias SANs for site %d", site.ID)
	return nil
}

func (s *Service) syncWebsiteAlias(site *models.WordPressSite, domain string) error {
	websiteID, err := s.ensureWebsite(site)
	if err != nil {
		return err
	}
	if site.WebsiteID == 0 {
		site.WebsiteID = websiteID
		_ = s.db.Model(site).Update("website_id", websiteID).Error
	}
	host := domaincheck.HostOnly(domain)
	port := 80
	var existing models.WebsiteAlias
	if s.db.Where("website_id = ? AND domain = ?", websiteID, host).First(&existing).Error == nil {
		return nil
	}
	aliasType := "alias"
	if strings.EqualFold(host, domaincheck.HostOnly(site.Domain)) {
		aliasType = "primary"
	}
	return s.db.Create(&models.WebsiteAlias{
		WebsiteID: websiteID, Domain: host, Port: port, Type: aliasType,
	}).Error
}

func (s *Service) regenerateVhost(siteID uint) error {
	site, err := s.Get(siteID)
	if err != nil {
		return err
	}
	conf, err := s.writeNginxVhost(site)
	if err != nil {
		return err
	}
	if err := s.db.Model(site).Update("nginx_conf", conf).Error; err != nil {
		return err
	}
	return reloadNginxIfAvailable()
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

func parseDomainList(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';' || r == ' '
	}) {
		if d := normalizeDomain(part); d != "" {
			out = append(out, d)
		}
	}
	return out
}
