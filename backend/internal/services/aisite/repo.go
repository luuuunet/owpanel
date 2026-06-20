package aisite

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitCloneTimeout = 120 * time.Second

type RepoSnapshot struct {
	RepoURL       string   `json:"repo_url"`
	Branch        string   `json:"branch"`
	ClonePath     string   `json:"-"`
	FileList      []string `json:"file_list"`
	HasCargo       bool     `json:"has_cargo"`
	HasComposer   bool     `json:"has_composer"`
	HasPackageJSON bool    `json:"has_package_json"`
	HasDockerfile bool     `json:"has_dockerfile"`
	HasDockerCompose bool  `json:"has_docker_compose"`
	HasNodeServer bool     `json:"has_node_server"`
	HasIndexPHP   bool     `json:"has_index_php"`
	HasIndexHTML  bool     `json:"has_index_html"`
	HasArtisan    bool     `json:"has_artisan"`
	HasWPConfig   bool     `json:"has_wp_config"`
	ComposerJSON  string   `json:"composer_json,omitempty"`
	PackageJSON   string   `json:"package_json,omitempty"`
	Dockerfile    string   `json:"dockerfile,omitempty"`
	Readme        string   `json:"readme,omitempty"`
	FrameworkHint string   `json:"framework_hint"`
	// Runtime hints from clone analysis (temp dir removed after fetch).
	HasPnpmLock        bool   `json:"has_pnpm_lock"`
	HasYarnLock        bool   `json:"has_yarn_lock"`
	HasPackageLock     bool   `json:"has_package_lock"`
	LockfileKind       string `json:"lockfile_kind,omitempty"`
	UsesCatalog        bool   `json:"uses_catalog"`
	PackageManager     string `json:"package_manager,omitempty"`
	NodeMajorRequired  int    `json:"node_major_required,omitempty"`
	PHPVersionRequired string `json:"php_version_required,omitempty"`
	IsMonorepo         bool   `json:"is_monorepo"`
	HasTurbo           bool   `json:"has_turbo"`
	PrimaryAppFilter   string `json:"primary_app_filter,omitempty"`
	PrimaryAppPath     string `json:"primary_app_path,omitempty"`
	PrimaryAppOutDir   string `json:"primary_app_out_dir,omitempty"`
	BuildEnvKeys       []string `json:"build_env_keys,omitempty"`
}

func normalizeRepoURL(raw string) string {
	u := strings.TrimSpace(raw)
	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, ".git")
	if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
		if strings.HasPrefix(strings.ToLower(u), "github.com/") {
			u = "https://" + u
		} else {
			u = "https://github.com/" + strings.TrimPrefix(u, "/")
		}
	}
	return u + ".git"
}

func cloneURLWithToken(repoURL, token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return repoURL
	}
	u := strings.TrimSuffix(repoURL, ".git")
	if strings.HasPrefix(u, "https://github.com/") {
		return "https://oauth2:" + token + "@github.com/" + strings.TrimPrefix(u, "https://github.com/") + ".git"
	}
	if strings.HasPrefix(u, "https://") {
		return strings.Replace(u, "https://", "https://oauth2:"+token+"@", 1) + ".git"
	}
	return repoURL
}

func (s *Service) fetchRepoSnapshot(repoURL, branch, token string) (*RepoSnapshot, error) {
	if !gitAvailable() {
		return nil, fmt.Errorf("未找到 git 命令，请先安装 Git")
	}
	repoURL = normalizeRepoURL(repoURL)
	branch = resolveGitBranch(repoURL, branch, token)
	tmp, err := os.MkdirTemp("", "op-aisite-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	cloneURL := cloneURLWithToken(repoURL, token)
	if err := gitClone(cloneURL, branch, tmp); err != nil {
		return nil, fmt.Errorf("git clone 失败: %w", err)
	}

	snap := &RepoSnapshot{
		RepoURL:   strings.TrimSuffix(repoURL, ".git"),
		Branch:    branch,
		ClonePath: tmp,
	}
	snap.FileList = listTopFiles(tmp, 80)
	snap.ComposerJSON = readTrunc(filepath.Join(tmp, "composer.json"), 4000)
	snap.PackageJSON = readTrunc(filepath.Join(tmp, "package.json"), 4000)
	snap.HasCargo = fileExists(filepath.Join(tmp, "Cargo.toml"))
	snap.Dockerfile = readTrunc(filepath.Join(tmp, "Dockerfile"), 3000)
	snap.Readme = readTrunc(firstExisting(tmp, "README.md", "readme.md", "README.MD"), 5000)
	snap.HasComposer = snap.ComposerJSON != ""
	snap.HasPackageJSON = snap.PackageJSON != ""
	snap.HasDockerfile = snap.Dockerfile != ""
	snap.HasDockerCompose = fileExists(filepath.Join(tmp, "docker-compose.yml")) ||
		fileExists(filepath.Join(tmp, "docker-compose.yaml")) ||
		fileExists(filepath.Join(tmp, "compose.yml"))
	snap.HasNodeServer = detectNodeServer(snap.PackageJSON)
	snap.HasIndexPHP = fileExists(filepath.Join(tmp, "index.php"))
	snap.HasIndexHTML = fileExists(filepath.Join(tmp, "index.html"))
	snap.HasArtisan = fileExists(filepath.Join(tmp, "artisan"))
	snap.HasWPConfig = fileExists(filepath.Join(tmp, "wp-config-sample.php")) || fileExists(filepath.Join(tmp, "wp-config.php"))
	snap.FrameworkHint = detectFramework(snap)
	enrichSnapshotFromClone(snap)
	return snap, nil
}

// resolveGitBranch picks a clone branch: user preference when it exists, else remote HEAD, else main/master.
func resolveGitBranch(repoURL, preferred, token string) string {
	repoURL = normalizeRepoURL(repoURL)
	cloneURL := cloneURLWithToken(repoURL, token)
	preferred = strings.TrimSpace(preferred)

	if preferred != "" && remoteBranchExists(cloneURL, preferred) {
		return preferred
	}
	if def := remoteDefaultBranch(cloneURL); def != "" {
		return def
	}
	for _, fallback := range []string{"main", "master"} {
		if remoteBranchExists(cloneURL, fallback) {
			return fallback
		}
	}
	if preferred != "" {
		return preferred
	}
	return "main"
}

func remoteDefaultBranch(cloneURL string) string {
	gitBin := resolveGitBinary()
	if gitBin == "" {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "ls-remote", "--symref", cloneURL, "HEAD")
	cmd.Env = shellEnvWithGit()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ref: ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.HasPrefix(fields[1], "refs/heads/") {
			return strings.TrimPrefix(fields[1], "refs/heads/")
		}
	}
	return ""
}

func remoteBranchExists(cloneURL, branch string) bool {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return false
	}
	gitBin := resolveGitBinary()
	if gitBin == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	ref := "refs/heads/" + branch
	cmd := exec.CommandContext(ctx, gitBin, "ls-remote", "--heads", cloneURL, ref)
	cmd.Env = shellEnvWithGit()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), ref)
}

func gitClone(url, branch, dest string) error {
	gitBin := resolveGitBinary()
	if gitBin == "" {
		return fmt.Errorf("git 不可用")
	}
	ctx, cancel := context.WithTimeout(context.Background(), gitCloneTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, gitBin, "clone", "--depth", "1", "-b", branch, url, dest)
	cmd.Env = shellEnvWithGit()
	out, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("git clone 超时（%v），请检查网络或 GitHub 访问", gitCloneTimeout)
		}
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func listTopFiles(root string, limit int) []string {
	var out []string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		if strings.Contains(rel, string(os.PathSeparator)+".git") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if strings.Count(rel, string(os.PathSeparator)) >= 2 {
				return filepath.SkipDir
			}
			return nil
		}
		out = append(out, rel)
		if len(out) >= limit {
			return filepath.SkipAll
		}
		return nil
	})
	return out
}

func readTrunc(path string, max int) string {
	if path == "" || !fileExists(path) {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := string(b)
	if len(s) > max {
		return s[:max] + "\n...(truncated)"
	}
	return s
}

func firstExisting(root string, names ...string) string {
	for _, n := range names {
		p := filepath.Join(root, n)
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func detectNodeServer(packageJSON string) bool {
	if packageJSON == "" {
		return false
	}
	var pkg struct {
		Scripts      map[string]string `json:"scripts"`
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(packageJSON), &pkg); err != nil {
		return false
	}
	if _, ok := pkg.Scripts["start"]; ok {
		return true
	}
	for dep := range pkg.Dependencies {
		switch strings.ToLower(dep) {
		case "express", "fastify", "koa", "@nestjs/core", "next", "nuxt":
			return true
		}
	}
	return false
}

func detectFramework(s *RepoSnapshot) string {
	c := strings.ToLower(s.ComposerJSON)
	p := strings.ToLower(s.PackageJSON)
	switch {
	case s.HasArtisan || strings.Contains(c, "laravel/framework"):
		return "laravel"
	case s.HasCargo:
		return "rust"
	case s.HasDockerCompose || s.HasDockerfile:
		return "docker"
	case strings.Contains(c, "symfony/framework"):
		return "symfony"
	case s.HasWPConfig || fileExists(filepath.Join(s.ClonePath, "wp-content")):
		return "wordpress"
	case strings.Contains(p, "next"):
		return "nextjs"
	case strings.Contains(p, "vue"):
		return "vue"
	case strings.Contains(p, "react"):
		return "react"
	case s.HasPackageJSON:
		return "nodejs"
	case s.HasComposer:
		return "php"
	case s.HasIndexHTML && !s.HasIndexPHP:
		return "static"
	default:
		return "unknown"
	}
}
