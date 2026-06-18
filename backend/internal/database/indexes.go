package database

import (
	"log"

	"gorm.io/gorm"
)

// EnsureTrafficIndexes creates composite indexes for traffic_hits query patterns.
// AutoMigrate may not add these on existing deployments; IF NOT EXISTS is safe to re-run.
func EnsureTrafficIndexes(db *gorm.DB) {
	if db == nil {
		return
	}
	stmts := []string{
		// Range on created_at + filter log_source (traffic-map, geo analytics)
		`CREATE INDEX IF NOT EXISTS idx_traffic_created_source ON traffic_hits(created_at, log_source)`,
		// DISTINCT ip within time window
		`CREATE INDEX IF NOT EXISTS idx_traffic_created_source_ip ON traffic_hits(created_at, log_source, ip)`,
		// Country aggregation within time window
		`CREATE INDEX IF NOT EXISTS idx_traffic_created_source_country ON traffic_hits(created_at, log_source, country_code)`,
	}
	for _, sql := range stmts {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("ensure traffic index: %v", err)
		}
	}
	if err := db.Exec("ANALYZE traffic_hits").Error; err != nil {
		log.Printf("analyze traffic_hits: %v", err)
	}
}
