package appstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	dynamicCatalogMu    sync.RWMutex
	dynamicCatalogItems []catalogItem
)

func dynamicCatalogPath(dataDir string) string {
	return filepath.Join(dataDir, "appstore", "dynamic_catalog.json")
}

func loadDynamicCatalog(dataDir string) {
	dynamicCatalogMu.Lock()
	defer dynamicCatalogMu.Unlock()
	dynamicCatalogItems = nil
	data, err := os.ReadFile(dynamicCatalogPath(dataDir))
	if err != nil {
		return
	}
	var items []catalogItem
	if err := json.Unmarshal(data, &items); err != nil {
		return
	}
	dynamicCatalogItems = items
}

func saveDynamicCatalog(dataDir string, items []catalogItem) error {
	dir := filepath.Join(dataDir, "appstore")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	dynamicCatalogMu.Lock()
	dynamicCatalogItems = append([]catalogItem(nil), items...)
	dynamicCatalogMu.Unlock()
	return os.WriteFile(dynamicCatalogPath(dataDir), data, 0644)
}

func ensureDynamicPHPCatalog(dataDir string) {
	dynamicCatalogMu.RLock()
	loaded := len(dynamicCatalogItems) > 0
	dynamicCatalogMu.RUnlock()
	if loaded {
		return
	}
	if data, err := os.ReadFile(dynamicCatalogPath(dataDir)); err == nil && len(strings.TrimSpace(string(data))) > 2 {
		return
	}
	dynamic := buildDynamicPHPCatalog()
	if len(dynamic) == 0 {
		return
	}
	_ = saveDynamicCatalog(dataDir, dynamic)
}

func appendBuiltinPHPCatalog(out []catalogItem) []catalogItem {
	seen := make(map[string]struct{}, len(out))
	for _, item := range out {
		seen[item.Key] = struct{}{}
	}
	for _, item := range buildDynamicPHPCatalog() {
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func appendDynamicCatalog(out []catalogItem) []catalogItem {
	dynamicCatalogMu.RLock()
	defer dynamicCatalogMu.RUnlock()
	if len(dynamicCatalogItems) == 0 {
		return out
	}
	seen := make(map[string]struct{}, len(out))
	for _, item := range out {
		seen[item.Key] = struct{}{}
	}
	for _, item := range dynamicCatalogItems {
		if _, ok := seen[item.Key]; ok {
			continue
		}
		out = append(out, item)
	}
	return out
}
