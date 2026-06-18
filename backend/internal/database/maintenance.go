package database

import (
	"log"
	"time"

	"gorm.io/gorm"
)

// TuneSQLite limits connection pooling for embedded SQLite.
func TuneSQLite(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)
}

// CheckpointWAL flushes the write-ahead log to keep WAL files from growing unbounded.
func CheckpointWAL(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
}

// Vacuum reclaims disk space after large deletes. Run in background only.
func Vacuum(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	return db.Exec("VACUUM").Error
}

// StartPeriodicMaintenance runs WAL checkpoints on a fixed interval.
func StartPeriodicMaintenance(db *gorm.DB, interval time.Duration) {
	if db == nil || interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			if err := CheckpointWAL(db); err != nil {
				log.Printf("sqlite wal checkpoint: %v", err)
			}
		}
	}()
}
