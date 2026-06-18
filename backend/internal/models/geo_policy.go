package models

import "time"

// WebsiteGeoPolicy defines per-site country access rules (block or redirect).
type WebsiteGeoPolicy struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	WebsiteID    uint      `gorm:"index" json:"website_id"`
	CountryCode  string    `gorm:"size:8;index" json:"country_code"`
	CountryName  string    `gorm:"size:64" json:"country_name"`
	Action       string    `gorm:"size:16" json:"action"` // block | redirect
	RedirectURL  string    `gorm:"size:512" json:"redirect_url"`
	Enabled      bool      `gorm:"default:true" json:"enabled"`
	Remark       string    `gorm:"size:256" json:"remark"`
}
