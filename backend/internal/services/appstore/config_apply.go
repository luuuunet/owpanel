package appstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Service) applyCommonConfigToFile(key string, cfg map[string]interface{}) error {
	if IsPHPKey(key) {
		return nil
	}
	meta, err := s.ConfigMeta(key)
	if err != nil {
		return err
	}
	path := meta.ResolvedConfigPath
	if path == "" {
		path = s.resolveConfigPath(key, "")
	}
	if path == "" {
		path = filepath.Join(s.dataDir, "apps", key, ".env")
	}
	kind := detectConfigKind(path, key)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	switch kind {
	case "env":
		return writeEnvConfig(path, cfg)
	case "json":
		return writeJSONConfigFile(path, cfg)
	case "ini", "php":
		return mergeIniConfig(path, cfg)
	default:
		return nil
	}
}

func writeEnvConfig(path string, cfg map[string]interface{}) error {
	var lines []string
	if b, err := os.ReadFile(path); err == nil {
		existing := parseEnvLines(string(b))
		for k, v := range cfg {
			existing[k] = fmt.Sprint(v)
		}
		cfg = existing
	}
	keys := sortedKeys(cfg)
	for _, k := range keys {
		lines = append(lines, k+"="+fmt.Sprint(cfg[k]))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func parseEnvLines(content string) map[string]interface{} {
	out := map[string]interface{}{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}

func writeJSONConfigFile(path string, cfg map[string]interface{}) error {
	var merged map[string]interface{}
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &merged)
	}
	if merged == nil {
		merged = map[string]interface{}{}
	}
	for k, v := range cfg {
		merged[k] = v
	}
	b, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0644)
}

func mergeIniConfig(path string, cfg map[string]interface{}) error {
	content := ""
	if b, err := os.ReadFile(path); err == nil {
		content = string(b)
	}
	for k, v := range cfg {
		content = setIniLine(content, k, fmt.Sprint(v))
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func setIniLine(content, key, value string) string {
	lines := strings.Split(content, "\n")
	found := false
	prefixes := []string{key + "=", key + " =", key + "  ="}
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		for _, p := range prefixes {
			if strings.HasPrefix(trim, p) || strings.HasPrefix(trim, key+"=") {
				lines[i] = key + " = " + value
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + key + " = " + value + "\n"
	}
	return strings.Join(lines, "\n")
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

func loadEnvConfig(path string) map[string]interface{} {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseEnvLines(string(b))
}
