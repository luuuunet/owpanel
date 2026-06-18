package models

import "time"

// TrafficHit stores parsed access log entries with GeoIP enrichment.
type TrafficHit struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `gorm:"index;index:idx_traffic_log_created,priority:2;index:idx_traffic_created_source,priority:1;index:idx_traffic_created_source_ip,priority:1;index:idx_traffic_created_source_country,priority:1" json:"created_at"`
	IP          string    `gorm:"size:64;index;index:idx_traffic_created_source_ip,priority:3" json:"ip"`
	CountryCode string    `gorm:"size:8;index;index:idx_traffic_created_source_country,priority:3" json:"country_code"`
	CountryName string    `gorm:"size:64" json:"country_name"`
	City        string    `gorm:"size:128" json:"city"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	Bytes       uint64    `json:"bytes"`
	Status      int       `json:"status"`
	Method      string    `gorm:"size:16" json:"method"`
	Path        string    `gorm:"size:512" json:"path"`
	Host        string    `gorm:"size:256;index" json:"host"`
	Referer     string    `gorm:"size:512" json:"referer"`
	UserAgent   string    `gorm:"size:512" json:"user_agent"`
	LogSource   string    `gorm:"size:256;index:idx_traffic_log_created,priority:1;index:idx_traffic_created_source,priority:2;index:idx_traffic_created_source_ip,priority:2;index:idx_traffic_created_source_country,priority:2" json:"log_source"`
}
