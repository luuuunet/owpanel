package dataplatform

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type WeightAsset struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Source      string    `json:"source"`
	Path        string    `json:"path"`
	SizeBytes   int64     `json:"size_bytes"`
	SizeHuman   string    `json:"size_human"`
	Version     string    `json:"version,omitempty"`
	ModifiedAt  time.Time `json:"modified_at"`
	SnapshotOf  string    `json:"snapshot_of,omitempty"`
}

type WeightsSummary struct {
	TotalBytes  int64         `json:"total_bytes"`
	TotalHuman  string        `json:"total_human"`
	Assets      []WeightAsset `json:"assets"`
	SyncHint    string        `json:"sync_hint"`
	BackupDir   string        `json:"backup_dir"`
}

func (s *Service) WeightsSummary() WeightsSummary {
	assets := s.listWeightAssets()
	var total int64
	for _, a := range assets {
		total += a.SizeBytes
	}
	backupDir := filepath.Join(s.dataDir, "ai", "weights-backups")
	return WeightsSummary{
		TotalBytes: total,
		TotalHuman: humanSize(total),
		Assets:     assets,
		SyncHint:   "rsync -avz " + filepath.Join(s.dataDir, "ai/") + " user@node:/opt/owpanel/data/ai/",
		BackupDir:  backupDir,
	}
}

func (s *Service) listWeightAssets() []WeightAsset {
	var assets []WeightAsset
	roots := []struct {
		source, rel string
	}{
		{"huggingface", "ai/huggingface"},
		{"ollama", "ai/ollama"},
		{"tgi-cache", "ai/huggingface/cache"},
	}
	for _, r := range roots {
		dir := filepath.Join(s.dataDir, r.rel)
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			// group by top-level model folder under models/
			rel, _ := filepath.Rel(s.dataDir, path)
			name := filepath.Base(path)
			if strings.HasPrefix(rel, "ai/huggingface/models/") {
				parts := strings.Split(rel, string(os.PathSeparator))
				if len(parts) >= 4 {
					name = parts[3]
				}
			}
			assets = append(assets, WeightAsset{
				ID:         rel,
				Name:       name,
				Source:     r.source,
				Path:       rel,
				SizeBytes:  info.Size(),
				SizeHuman:  humanSize(info.Size()),
				ModifiedAt: info.ModTime(),
			})
			return nil
		})
	}
	// Ollama models via docker if available
	if out, err := exec.Command("docker", "exec", "owpanel-ollama", "ollama", "list").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n")[1:] {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			assets = append(assets, WeightAsset{
				ID:     "ollama:" + fields[0],
				Name:   fields[0],
				Source: "ollama",
				Path:   "docker:owpanel-ollama",
				Version: fields[1],
			})
		}
	}
	// dedupe by ID keeping largest
	byID := map[string]WeightAsset{}
	for _, a := range assets {
		if prev, ok := byID[a.ID]; !ok || a.SizeBytes > prev.SizeBytes {
			byID[a.ID] = a
		}
	}
	out := make([]WeightAsset, 0, len(byID))
	for _, a := range byID {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SizeBytes > out[j].SizeBytes })
	if len(out) > 100 {
		out = out[:100]
	}
	return out
}

func (s *Service) SnapshotWeight(assetID string) (string, error) {
	src := filepath.Join(s.dataDir, assetID)
	if _, err := os.Stat(src); err != nil {
		return "", fmt.Errorf("asset not found: %s", assetID)
	}
	backupDir := filepath.Join(s.dataDir, "ai", "weights-backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	ts := time.Now().Format("20060102-150405")
	base := strings.ReplaceAll(filepath.Base(assetID), "/", "_")
	outPath := filepath.Join(backupDir, base+"-"+ts+".tar.gz")
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		hdr, _ := tar.FileInfoHeader(info, "")
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(tw, file)
		return err
	})
	if err != nil {
		_ = os.Remove(outPath)
		return "", err
	}
	return outPath, nil
}

func (s *Service) DeleteWeightCache(assetID string) error {
	if strings.HasPrefix(assetID, "ollama:") {
		model := strings.TrimPrefix(assetID, "ollama:")
		return exec.Command("docker", "exec", "owpanel-ollama", "ollama", "rm", model).Run()
	}
	target := filepath.Join(s.dataDir, assetID)
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("not found")
	}
	return os.RemoveAll(target)
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
