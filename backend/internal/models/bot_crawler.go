package models

import "time"

// BotCrawlerRule stores per-global or per-site crawler allow/block policy.
// WebsiteID 0 = global default; site rules may use action "inherit".
type BotCrawlerRule struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	WebsiteID  uint      `gorm:"uniqueIndex:idx_bot_crawler_site;default:0" json:"website_id"`
	CrawlerID  string    `gorm:"uniqueIndex:idx_bot_crawler_site;size:64" json:"crawler_id"`
	Action     string    `gorm:"size:16;default:inherit" json:"action"` // allow | block | inherit (site only)
}
