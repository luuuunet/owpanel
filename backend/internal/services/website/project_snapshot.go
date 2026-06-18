package website

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"
)

type ProjectSnapshot struct {
	Domain       string            `json:"domain"`
	RootPath     string            `json:"root_path"`
	ProjectType  string            `json:"project_type"`
	PhpVersion   string            `json:"php_version"`
	FileList     []string          `json:"file_list"`
	FileContents map[string]string `json:"file_contents"`
}

type snapshotCacheEntry struct {
	snap *ProjectSnapshot
	at   time.Time
}

var (
	projectSnapshotCache sync.Map
	snapshotCacheTTL     = 2 * time.Minute
)

var projectSkipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".next": true,
	"dist": true, "build": true, ".turbo": true, "coverage": true,
	"__pycache__": true, ".cache": true, "storage": true, "cache": true,
}

var projectTextExts = map[string]bool{
	".php": true, ".html": true, ".htm": true, ".css": true, ".scss": true,
	".sass": true, ".less": true, ".js": true, ".jsx": true, ".ts": true,
	".tsx": true, ".vue": true, ".json": true, ".md": true, ".txt": true,
	".xml": true, ".yaml": true, ".yml": true, ".twig": true, ".blade.php": true,
	".env.example": true,
}

var projectKeyNames = map[string]bool{
	"package.json": true, "composer.json": true, "pnpm-lock.yaml": true,
	"style.css": true, "theme.json": true, "functions.php": true,
	"index.php": true, "index.html": true, "tailwind.config.js": true,
	"tailwind.config.ts": true, "next.config.js": true, "next.config.mjs": true,
	"next.config.ts": true, "vite.config.ts": true, "vite.config.js": true,
	"nuxt.config.ts": true, "app.css": true, "globals.css": true,
}

func snapshotCacheKey(siteID uint, scope, focusPath string) string {
	return fmt.Sprintf("%d|%s|%s", siteID, scope, strings.TrimSpace(focusPath))
}

func (s *Service) BuildProjectSnapshot(siteID uint, focusPath string) (*ProjectSnapshot, error) {
	return s.BuildProjectSnapshotForChat(siteID, focusPath, "project", "")
}

// BuildProjectSnapshotForChat builds a lightweight snapshot tuned for AI chat latency.
func (s *Service) BuildProjectSnapshotForChat(siteID uint, focusPath, scope, message string) (*ProjectSnapshot, error) {
	if scope != "file" {
		scope = "project"
	}
	key := snapshotCacheKey(siteID, scope, focusPath)
	if v, ok := projectSnapshotCache.Load(key); ok {
		if ent, ok := v.(snapshotCacheEntry); ok && time.Since(ent.at) < snapshotCacheTTL {
			return cloneSnapshot(ent.snap), nil
		}
	}

	snap, err := s.buildProjectSnapshot(siteID, focusPath, scope, message)
	if err != nil {
		return nil, err
	}
	projectSnapshotCache.Store(key, snapshotCacheEntry{snap: snap, at: time.Now()})
	return cloneSnapshot(snap), nil
}

func cloneSnapshot(src *ProjectSnapshot) *ProjectSnapshot {
	if src == nil {
		return nil
	}
	dup := *src
	dup.FileList = append([]string(nil), src.FileList...)
	dup.FileContents = make(map[string]string, len(src.FileContents))
	for k, v := range src.FileContents {
		dup.FileContents[k] = v
	}
	return &dup
}

func (s *Service) buildProjectSnapshot(siteID uint, focusPath, scope, message string) (*ProjectSnapshot, error) {
	site, err := s.Get(siteID)
	if err != nil {
		return nil, err
	}
	if site.RootPath == "" {
		return nil, fmt.Errorf("未配置网站根目录")
	}
	rootAbs, err := filepath.Abs(site.RootPath)
	if err != nil {
		return nil, err
	}

	maxFiles := 350
	maxDepth := 5
	contentBudget := 48000
	maxPerFile := 8000
	if scope == "file" {
		maxFiles = 180
		maxDepth = 4
		contentBudget = 32000
		maxPerFile = 12000
	}

	snap := &ProjectSnapshot{
		Domain:       site.Domain,
		RootPath:     rootAbs,
		ProjectType:  site.ProjectType,
		PhpVersion:   site.PhpVersion,
		FileList:     []string{},
		FileContents: map[string]string{},
	}
	msgHints := messagePathHints(message)

	filepath.WalkDir(rootAbs, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if projectSkipDirs[d.Name()] {
				return filepath.SkipDir
			}
			rel, _ := filepath.Rel(rootAbs, path)
			depth := strings.Count(rel, string(os.PathSeparator))
			if depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if len(snap.FileList) >= maxFiles {
			return nil
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil || rel == "" || strings.HasPrefix(rel, "..") {
			return nil
		}
		rel = filepath.ToSlash(rel)
		snap.FileList = append(snap.FileList, rel)
		if contentBudget <= 0 {
			return nil
		}
		if !shouldIncludeProjectFile(rel, focusPath, scope, msgHints) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) == 0 {
			return nil
		}
		if len(data) > maxPerFile {
			data = data[:maxPerFile]
		}
		if len(data) > contentBudget {
			data = data[:contentBudget]
		}
		snap.FileContents[rel] = string(data)
		contentBudget -= len(data)
		return nil
	})
	return snap, nil
}

func messagePathHints(message string) []string {
	message = strings.ToLower(strings.TrimSpace(message))
	if message == "" {
		return nil
	}
	known := []string{
		"theme", "themes", "css", "style", "styles", "scss", "sass", "less",
		"layout", "header", "footer", "home", "index", "dark", "light",
		"主题", "配色", "颜色", "样式", "布局", "首页", "深色", "浅色", "导航", "页脚", "页眉",
		"apps/site", "tsx", "vue", "tailwind", "wordpress", "wp-content",
	}
	var hints []string
	for _, k := range known {
		if strings.Contains(message, k) {
			hints = append(hints, k)
		}
	}
	for _, tok := range strings.FieldsFunc(message, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r > 127
	}) {
		if len(tok) >= 3 && len(hints) < 12 {
			hints = append(hints, tok)
		}
	}
	return hints
}

func shouldIncludeProjectFile(rel, focusPath, scope string, msgHints []string) bool {
	base := filepath.Base(rel)
	lower := strings.ToLower(rel)
	focus := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(focusPath)), "/")

	if scope == "file" && focus != "" {
		if rel == focus {
			return true
		}
		if projectKeyNames[base] {
			return true
		}
		if focus != "" && (strings.HasPrefix(rel, focus+"/") || strings.HasPrefix(focus, rel+"/")) {
			return true
		}
		if strings.Contains(lower, "/themes/") || strings.Contains(lower, "/theme/") ||
			strings.Contains(lower, "/assets/") || strings.Contains(lower, "/css/") ||
			strings.Contains(lower, "/styles/") || strings.Contains(lower, "apps/site/") {
			ext := strings.ToLower(filepath.Ext(rel))
			if projectTextExts[ext] {
				return true
			}
		}
		return false
	}

	if projectKeyNames[base] {
		return true
	}
	if focus != "" && (rel == focus || strings.HasPrefix(rel, focus+"/") || strings.HasPrefix(focus, rel+"/")) {
		return true
	}
	for _, h := range msgHints {
		if h != "" && strings.Contains(lower, strings.ToLower(h)) {
			ext := strings.ToLower(filepath.Ext(rel))
			if projectTextExts[ext] || ext == ".php" {
				return true
			}
		}
	}
	if strings.Contains(lower, "/themes/") || strings.Contains(lower, "/theme/") {
		ext := strings.ToLower(filepath.Ext(rel))
		if projectTextExts[ext] || ext == ".php" {
			return true
		}
	}
	if strings.Contains(lower, "/assets/") || strings.Contains(lower, "/css/") || strings.Contains(lower, "/styles/") {
		ext := strings.ToLower(filepath.Ext(rel))
		if ext == ".css" || ext == ".scss" || ext == ".sass" || ext == ".less" {
			return true
		}
	}
	if strings.Contains(lower, "apps/site/") && (strings.HasSuffix(lower, ".tsx") || strings.HasSuffix(lower, ".css") || strings.HasSuffix(lower, ".json")) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if !projectTextExts[ext] {
		return false
	}
	if len(rel) > 120 {
		return false
	}
	parts := strings.Split(rel, "/")
	return len(parts) <= 3
}
