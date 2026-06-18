package kafkaaccel

import (
	"fmt"
	"strings"
)

type PresetResult struct {
	Name        string            `json:"name"`
	Config      *KafkaAccelConfig `json:"config,omitempty"`
	AutoEnable  *AutoEnableResult `json:"auto_enable,omitempty"`
	Apply       *ApplyResult      `json:"apply,omitempty"`
	Message     string            `json:"message"`
}

func presetByKey(key string) (*KafkaAccelConfig, bool, bool) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "optimize":
		return &KafkaAccelConfig{
			Enabled:           true,
			BootstrapServers:  "127.0.0.1:9092",
			TopicPrefix:       "opanel.db",
			Mode:              "write_async",
			ConsumerGroup:     "opanel-db-accel",
			TopicPartitions:   3,
			ReplicationFactor: 1,
			RetentionHours:    168,
			ProducerBatchSize: 32768,
			ProducerLingerMs:  5,
			CompressionType:   "lz4",
			FetchMinBytes:     1,
		}, true, true
	case "high_throughput":
		return &KafkaAccelConfig{
			TopicPartitions:   6,
			ReplicationFactor: 1,
			RetentionHours:    168,
			ProducerBatchSize: 65536,
			ProducerLingerMs:  20,
			CompressionType:   "lz4",
			FetchMinBytes:     1,
		}, true, false
	case "low_latency":
		return &KafkaAccelConfig{
			TopicPartitions:   3,
			ReplicationFactor: 1,
			RetentionHours:    168,
			ProducerBatchSize: 16384,
			ProducerLingerMs:  0,
			CompressionType:   "none",
			FetchMinBytes:     1,
		}, true, false
	default:
		return nil, false, false
	}
}

func (s *Service) ApplyPreset(key string) (*PresetResult, error) {
	preset, ok, runAutoEnable := presetByKey(key)
	if !ok {
		return nil, fmt.Errorf("unknown preset: %s", key)
	}

	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}

	patch := *cfg
	if runAutoEnable {
		patch.Enabled = preset.Enabled
		if v := strings.TrimSpace(preset.BootstrapServers); v != "" {
			patch.BootstrapServers = v
		}
		if v := strings.TrimSpace(preset.TopicPrefix); v != "" {
			patch.TopicPrefix = v
		}
		if v := strings.TrimSpace(preset.Mode); v != "" {
			patch.Mode = v
		}
		if v := strings.TrimSpace(preset.ConsumerGroup); v != "" {
			patch.ConsumerGroup = v
		}
		ids, err := s.eligibleDatabaseIDs()
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return nil, fmt.Errorf("未找到可加速的本地数据库，请先添加 MySQL 或 PostgreSQL")
		}
		patch.LinkedDatabaseIDs = FormatLinkedIDs(ids)
	}
	patch.TopicPartitions = preset.TopicPartitions
	patch.ReplicationFactor = preset.ReplicationFactor
	patch.RetentionHours = preset.RetentionHours
	patch.ProducerBatchSize = preset.ProducerBatchSize
	patch.ProducerLingerMs = preset.ProducerLingerMs
	patch.CompressionType = preset.CompressionType
	patch.FetchMinBytes = preset.FetchMinBytes

	updated, err := s.UpdateConfig(&patch)
	if err != nil {
		return nil, err
	}

	result := &PresetResult{
		Name:    key,
		Config:  updated,
		Message: fmt.Sprintf("已应用预设 %s", key),
	}

	if runAutoEnable {
		autoRes, err := s.AutoEnable(true)
		if err != nil {
			return nil, err
		}
		result.AutoEnable = autoRes
		result.Apply = autoRes.Apply
		if autoRes.Message != "" {
			result.Message = autoRes.Message
		}
		return result, nil
	}

	if s.kafkaRunning() && len(ParseLinkedIDs(updated.LinkedDatabaseIDs)) > 0 {
		applyRes, err := s.Apply()
		if err != nil {
			result.Message = fmt.Sprintf("预设已保存，Topic 应用失败: %v", err)
		} else {
			result.Apply = applyRes
			result.Message = applyRes.Message
		}
	}

	return result, nil
}
