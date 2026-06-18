package appstore

import (
	"strings"
)

// ConfigCapabilities describes what the software config UI can offer.
type ConfigCapabilities struct {
	IsPHP                    bool   `json:"is_php"`
	HasConfigFile            bool   `json:"has_config_file"`
	SupportsCommonConfig     bool   `json:"supports_common_config"`
	SupportsRawEdit          bool   `json:"supports_raw_edit"`
	SupportsExtensions       bool   `json:"supports_extensions"`
	SupportsPgExtensions     bool   `json:"supports_pg_extensions"`
	SupportsDisableFunctions bool   `json:"supports_disable_functions"`
	SupportsAI               bool   `json:"supports_ai"`
	IsDockerApp              bool   `json:"is_docker_app"`
	ConfigKind               string `json:"config_kind"` // php|ini|json|yaml|env|xml|raw
}

func (s *Service) ConfigCapabilities(key string) (ConfigCapabilities, error) {
	meta, err := s.ConfigMeta(key)
	if err != nil {
		return ConfigCapabilities{}, err
	}
	app, _ := s.Get(key)
	def := defaultConfigFor(key)
	_, isDocker := dockerSpec(key)

	cap := ConfigCapabilities{
		IsPHP:                    meta.IsPHP,
		HasConfigFile:            meta.HasConfigFile,
		SupportsCommonConfig:     len(def) > 0 || isDocker || meta.IsPHP,
		SupportsRawEdit:          true,
		SupportsExtensions:       meta.IsPHP,
		SupportsPgExtensions:     key == "postgresql",
		SupportsDisableFunctions: meta.IsPHP,
		SupportsAI:               true,
		IsDockerApp:              isDocker,
		ConfigKind:               detectConfigKind(meta.ResolvedConfigPath, key),
	}
	if cap.SupportsCommonConfig && len(def) == 0 && app != nil && app.Config != "" {
		cap.SupportsCommonConfig = true
	}
	if !cap.HasConfigFile && cap.ConfigKind == "env" {
		cap.SupportsRawEdit = true
	}
	return cap, nil
}

func detectConfigKind(path, key string) string {
	if IsPHPKey(key) {
		return "php"
	}
	p := strings.ToLower(path)
	switch {
	case strings.HasSuffix(p, ".json"):
		return "json"
	case strings.HasSuffix(p, ".yaml") || strings.HasSuffix(p, ".yml"):
		return "yaml"
	case strings.HasSuffix(p, ".env"):
		return "env"
	case strings.HasSuffix(p, ".xml"):
		return "xml"
	case strings.HasSuffix(p, ".conf") || strings.HasSuffix(p, ".ini") || strings.HasSuffix(p, ".cnf"):
		return "ini"
	default:
		if _, ok := dockerSpec(key); ok {
			return "env"
		}
		return "raw"
	}
}
