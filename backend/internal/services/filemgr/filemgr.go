package filemgr

import (
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	ModTime int64  `json:"mod_time"`
	Mode    string `json:"mode"`
}

type Service struct {
	defaultRoot string
	dataDir     string
}

func NewService(dataDir string) *Service {
	root := filepath.Join(dataDir, "wwwroot")
	if dataDir == "" {
		root = "/"
	}
	return &Service{defaultRoot: root, dataDir: dataDir}
}

func (s *Service) DefaultRoot() string {
	return s.defaultRoot
}

func (s *Service) Roots() []map[string]string {
	roots := []map[string]string{
		{"label": "wwwroot", "path": s.defaultRoot},
	}
	if s.dataDir != "" {
		roots = append(roots, map[string]string{"label": "data", "path": s.dataDir})
	}
	if runtime.GOOS == "windows" {
		roots = append(roots, map[string]string{"label": "C:\\", "path": "C:\\"})
	} else {
		roots = append(roots, map[string]string{"label": "/", "path": "/"})
	}
	return roots
}

func (s *Service) resolvePath(path string) (string, error) {
	if path == "" || path == "/" {
		if runtime.GOOS == "windows" && (path == "/" || path == "") {
			return filepath.Clean(s.defaultRoot), nil
		}
		if path == "" {
			return filepath.Clean(s.defaultRoot), nil
		}
	}
	p := filepath.Clean(path)
	if strings.Contains(p, "..") {
		return "", fs.ErrPermission
	}
	return p, nil
}

func formatMode(mode fs.FileMode) string {
	return fmt.Sprintf("%04o", mode.Perm())
}

func (s *Service) List(dir string) ([]Entry, error) {
	p, err := s.resolvePath(dir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	result := make([]Entry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		result = append(result, Entry{
			Name:    e.Name(),
			Path:    filepath.Join(p, e.Name()),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
			Mode:    formatMode(info.Mode()),
		})
	}
	return result, nil
}

func (s *Service) Stat(path string) (*FileInfo, error) {
	p, err := s.resolvePath(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		Name: info.Name(), Path: p, IsDir: info.IsDir(),
		Size: info.Size(), ModTime: info.ModTime().Unix(),
		Mode: formatMode(info.Mode()),
	}, nil
}

func (s *Service) Read(path string) ([]byte, error) {
	p, err := s.resolvePath(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory as file: %s", p)
	}
	return os.ReadFile(p)
}

func (s *Service) Write(path string, content []byte) error {
	p, err := s.resolvePath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return os.WriteFile(p, content, 0644)
}

func (s *Service) Delete(path string) error {
	p, err := s.resolvePath(path)
	if err != nil {
		return err
	}
	return os.RemoveAll(p)
}

// TreeSize returns total bytes used by path (file or directory tree).
func (s *Service) TreeSize(path string) (int64, error) {
	p, err := s.resolvePath(path)
	if err != nil {
		return 0, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return info.Size(), nil
	}
	var total int64
	err = filepath.WalkDir(p, func(_ string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		total += fi.Size()
		return nil
	})
	return total, err
}

func (s *Service) Mkdir(path string) error {
	p, err := s.resolvePath(path)
	if err != nil {
		return err
	}
	return os.MkdirAll(p, 0755)
}

func (s *Service) CreateFile(path string, content []byte, isDir bool) error {
	p, err := s.resolvePath(path)
	if err != nil {
		return err
	}
	if isDir {
		return os.MkdirAll(p, 0755)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	return os.WriteFile(p, content, 0644)
}

func (s *Service) Rename(path, newName string) error {
	p, err := s.resolvePath(path)
	if err != nil {
		return err
	}
	newName = strings.TrimSpace(newName)
	if newName == "" || strings.Contains(newName, string(filepath.Separator)) || strings.Contains(newName, "/") {
		return fmt.Errorf("invalid name")
	}
	newPath := filepath.Join(filepath.Dir(p), newName)
	return os.Rename(p, newPath)
}

type ChmodStats struct {
	Updated int `json:"updated"`
	Failed  int `json:"failed"`
}

func parseMode(modeStr string) (fs.FileMode, error) {
	modeStr = strings.TrimSpace(modeStr)
	if strings.HasPrefix(modeStr, "0") {
		modeStr = modeStr[1:]
	}
	var mode uint32
	if _, err := fmt.Sscanf(modeStr, "%o", &mode); err != nil {
		return 0, fmt.Errorf("invalid mode: %s", modeStr)
	}
	return fs.FileMode(mode), nil
}

func (s *Service) Chmod(path, modeStr string) error {
	p, err := s.resolvePath(path)
	if err != nil {
		return err
	}
	mode, err := parseMode(modeStr)
	if err != nil {
		return err
	}
	return os.Chmod(p, mode)
}

// ChmodRecursive applies mode to path and, if path is a directory, all entries beneath it.
// Symlinked directories are not traversed; each symlink entry is chmodded itself.
func (s *Service) ChmodRecursive(path, modeStr string) (ChmodStats, error) {
	p, err := s.resolvePath(path)
	if err != nil {
		return ChmodStats{}, err
	}
	mode, err := parseMode(modeStr)
	if err != nil {
		return ChmodStats{}, err
	}
	info, err := os.Lstat(p)
	if err != nil {
		return ChmodStats{}, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		if err := os.Chmod(p, mode); err != nil {
			return ChmodStats{Failed: 1}, nil
		}
		return ChmodStats{Updated: 1}, nil
	}
	var stats ChmodStats
	err = filepath.WalkDir(p, func(walkPath string, _ os.DirEntry, walkErr error) error {
		if walkErr != nil {
			stats.Failed++
			return nil
		}
		if err := os.Chmod(walkPath, mode); err != nil {
			stats.Failed++
		} else {
			stats.Updated++
		}
		return nil
	})
	return stats, err
}

func (s *Service) Upload(dir string, file multipart.File, filename string) (string, error) {
	p, err := s.resolvePath(dir)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(p)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("upload target is not a directory")
	}
	filename = filepath.Base(filename)
	if filename == "" || filename == "." {
		return "", fmt.Errorf("invalid filename")
	}
	target := filepath.Join(p, filename)
	out, err := os.Create(target)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, file); err != nil {
		return "", err
	}
	return target, nil
}
