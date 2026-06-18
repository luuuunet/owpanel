package analytics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const offsetsFileName = "log_offsets.json"

type offsetStore struct {
	mu   sync.Mutex
	path string
	dirty bool
}

func newOffsetStore(dataDir string) *offsetStore {
	return &offsetStore{path: filepath.Join(dataDir, "analytics", offsetsFileName)}
}

func (o *offsetStore) load() map[string]int64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	data, err := os.ReadFile(o.path)
	if err != nil || len(data) == 0 {
		return nil
	}
	var out map[string]int64
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}

func (o *offsetStore) save(offsets map[string]int64) {
	if len(offsets) == 0 {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(o.path), 0755); err != nil {
		return
	}
	data, err := json.Marshal(offsets)
	if err != nil {
		return
	}
	_ = os.WriteFile(o.path, data, 0644)
	o.dirty = false
}

func (o *offsetStore) markDirty() {
	o.mu.Lock()
	o.dirty = true
	o.mu.Unlock()
}

func (o *offsetStore) flushIfDirty(offsets map[string]int64) {
	o.mu.Lock()
	dirty := o.dirty
	o.mu.Unlock()
	if dirty {
		o.save(offsets)
	}
}
