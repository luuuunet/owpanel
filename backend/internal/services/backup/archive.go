package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func zipDirectory(srcDir, destZip string) (int64, error) {
	srcDir = filepath.Clean(srcDir)
	if st, err := os.Stat(srcDir); err != nil || !st.IsDir() {
		return 0, fmt.Errorf("备份源目录不存在: %s", srcDir)
	}
	if err := os.MkdirAll(filepath.Dir(destZip), 0755); err != nil {
		return 0, err
	}
	f, err := os.Create(destZip)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	var total int64
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "../") {
			return nil
		}
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		rf, err := os.Open(path)
		if err != nil {
			return err
		}
		defer rf.Close()
		n, err := io.Copy(w, rf)
		total += n
		return err
	})
	if err != nil {
		_ = os.Remove(destZip)
		return 0, err
	}
	if err := zw.Close(); err != nil {
		_ = os.Remove(destZip)
		return 0, err
	}
	st, err := os.Stat(destZip)
	if err != nil {
		return total, err
	}
	return st.Size(), nil
}

func resolvePath(dataDir, p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return filepath.Join(dataDir, "backup")
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(dataDir, p))
}
