package cilium

import "time"

type CiliumConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Scope     string    `gorm:"uniqueIndex;size:32;default:global" json:"scope"`

	HostFirewallEnabled bool   `gorm:"default:true" json:"host_firewall_enabled"`
	HubbleEnabled       bool   `gorm:"default:true" json:"hubble_enabled"`
	HubbleUIEnabled     bool   `gorm:"default:true" json:"hubble_ui_enabled"`
	AuditMode           bool   `gorm:"default:true" json:"audit_mode"`
	NetworkDevice       string `gorm:"size:64" json:"network_device"`
}
