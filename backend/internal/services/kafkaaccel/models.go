package kafkaaccel

import "time"

// KafkaAccelConfig global singleton for Kafka-based database acceleration.
// Modes:
//   - write_async: buffer writes to Kafka before async DB persistence (LinkedIn-style)
//   - cache_invalidate: broadcast cache invalidation events to Redis/CDN consumers
//   - read_through: route read-heavy queries through Kafka-backed read replicas / cache warmers
type KafkaAccelConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Scope     string    `gorm:"uniqueIndex;size:32;default:global" json:"scope"`

	Enabled             bool   `gorm:"default:false" json:"enabled"`
	BootstrapServers    string `gorm:"size:256;default:127.0.0.1:9092" json:"bootstrap_servers"`
	TopicPrefix         string `gorm:"size:128;default:opanel.db" json:"topic_prefix"`
	Mode                string `gorm:"size:32;default:write_async" json:"mode"`
	LinkedDatabaseIDs   string `gorm:"size:512" json:"linked_database_ids"`
	ConsumerGroup       string `gorm:"size:128;default:opanel-db-accel" json:"consumer_group"`

	TopicPartitions   int    `gorm:"default:3" json:"topic_partitions"`
	ReplicationFactor int    `gorm:"default:1" json:"replication_factor"`
	RetentionHours    int    `gorm:"default:24" json:"retention_hours"`
	ProducerBatchSize int    `gorm:"default:32768" json:"producer_batch_size"`
	ProducerLingerMs  int    `gorm:"default:5" json:"producer_linger_ms"`
	CompressionType   string `gorm:"size:16;default:lz4" json:"compression_type"`
	FetchMinBytes     int    `gorm:"default:1" json:"fetch_min_bytes"`
}

// KafkaAccelRule optional per-database topic override.
type KafkaAccelRule struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DatabaseID  uint      `gorm:"index" json:"database_id"`
	Topic       string    `gorm:"size:256" json:"topic"`
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	Description string    `gorm:"size:512" json:"description"`
}
