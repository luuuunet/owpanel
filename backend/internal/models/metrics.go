package models

import "time"

// MetricSnapshot stores periodic system monitor samples (CPU, memory, IO, etc.).
type MetricSnapshot struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
	CPU       float64   `json:"cpu"`
	Memory    float64   `json:"memory"`
	Load1     float64   `json:"load1"`
	NetUp     float64   `json:"net_up"`
	NetDown   float64   `json:"net_down"`
	DiskRead  float64   `json:"disk_read"`
	DiskWrite float64   `json:"disk_write"`
}
