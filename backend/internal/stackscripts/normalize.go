package stackscripts

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
)

// NormalizeLineEndings rewrites *.sh under dir to Unix LF (fixes CRLF breaking bash set -o pipefail).
func NormalizeLineEndings(dir string) error {
	dir = filepath.Clean(dir)
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".sh") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Contains(data, []byte("\r")) {
			return nil
		}
		normalized := bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		normalized = bytes.ReplaceAll(normalized, []byte("\r"), []byte("\n"))
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(path, normalized, info.Mode())
	})
}

func normalizeScriptFile(path string, data []byte, mode os.FileMode) error {
	if bytes.Contains(data, []byte("\r")) {
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
	}
	return os.WriteFile(path, data, mode)
}
