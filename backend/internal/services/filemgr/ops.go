package filemgr

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func copyPath(src, dest string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dest string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, srcInfo.Mode()); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode())
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func uniqueSiblingPath(dir, baseName string) (string, error) {
	ext := filepath.Ext(baseName)
	stem := strings.TrimSuffix(baseName, ext)
	candidates := []string{stem + " copy" + ext}
	for i := 2; i < 1000; i++ {
		candidates = append(candidates, fmt.Sprintf("%s copy %d%s", stem, i, ext))
	}
	for _, name := range candidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p, nil
		}
	}
	return "", fmt.Errorf("could not find unique name for %s", baseName)
}

// CopyItems copies paths into destDir (keeps basenames).
func (s *Service) CopyItems(paths []string, destDir string) error {
	destP, err := s.resolvePath(destDir)
	if err != nil {
		return err
	}
	destInfo, err := os.Stat(destP)
	if err != nil {
		return err
	}
	if !destInfo.IsDir() {
		return fmt.Errorf("destination is not a directory")
	}
	for _, p := range paths {
		srcP, err := s.resolvePath(p)
		if err != nil {
			return err
		}
		target := filepath.Join(destP, filepath.Base(srcP))
		if err := copyPath(srcP, target); err != nil {
			return err
		}
	}
	return nil
}

// MoveItems moves paths into destDir.
func (s *Service) MoveItems(paths []string, destDir string) error {
	destP, err := s.resolvePath(destDir)
	if err != nil {
		return err
	}
	destInfo, err := os.Stat(destP)
	if err != nil {
		return err
	}
	if !destInfo.IsDir() {
		return fmt.Errorf("destination is not a directory")
	}
	for _, p := range paths {
		srcP, err := s.resolvePath(p)
		if err != nil {
			return err
		}
		target := filepath.Join(destP, filepath.Base(srcP))
		if err := os.Rename(srcP, target); err != nil {
			if err := copyPath(srcP, target); err != nil {
				return err
			}
			if err := os.RemoveAll(srcP); err != nil {
				return err
			}
		}
	}
	return nil
}

// Duplicate creates a copy alongside the original with a unique name.
func (s *Service) Duplicate(path string) (string, error) {
	srcP, err := s.resolvePath(path)
	if err != nil {
		return "", err
	}
	dest, err := uniqueSiblingPath(filepath.Dir(srcP), filepath.Base(srcP))
	if err != nil {
		return "", err
	}
	if err := copyPath(srcP, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// SearchNames finds entries whose names contain query (case-insensitive) under dir, max depth 8.
func (s *Service) SearchNames(dir, query string, maxResults int) ([]Entry, error) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}
	if maxResults <= 0 {
		maxResults = 200
	}
	root, err := s.resolvePath(dir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory")
	}
	var result []Entry
	const maxDepth = 8
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if len(result) >= maxResults {
			return filepath.SkipAll
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}
		depth := strings.Count(rel, string(os.PathSeparator)) + 1
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), query) {
			fi, err := d.Info()
			if err != nil {
				return nil
			}
			result = append(result, Entry{
				Name:    d.Name(),
				Path:    path,
				IsDir:   d.IsDir(),
				Size:    fi.Size(),
				ModTime: fi.ModTime().Unix(),
				Mode:    formatMode(fi.Mode()),
			})
		}
		return nil
	})
	return result, err
}
