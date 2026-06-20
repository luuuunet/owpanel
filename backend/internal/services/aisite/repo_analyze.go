package aisite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// enrichSnapshotFromClone fills runtime requirements detected from the cloned tree.
func enrichSnapshotFromClone(snap *RepoSnapshot) {
	if snap == nil || snap.ClonePath == "" {
		return
	}
	root := snap.ClonePath
	snap.HasPnpmLock = fileExists(filepath.Join(root, "pnpm-lock.yaml"))
	snap.HasYarnLock = fileExists(filepath.Join(root, "yarn.lock"))
	snap.HasPackageLock = fileExists(filepath.Join(root, "package-lock.json"))
	snap.UsesCatalog = strings.Contains(snap.PackageJSON, "catalog:")
	snap.LockfileKind = detectLockfileKind(snap)
	snap.PackageManager = parsePackageManager(snap.PackageJSON)
	snap.NodeMajorRequired = parseNodeMajorRequired(root, snap.PackageJSON)
	snap.PHPVersionRequired = parsePHPVersionRequired(snap.ComposerJSON)
	enrichMonorepoSnapshot(snap)
	if snap.PHPVersionRequired != "" && snap.FrameworkHint != "nextjs" {
		snap.FrameworkHint = detectFramework(snap)
	} else if snap.FrameworkHint == "" {
		snap.FrameworkHint = detectFramework(snap)
	}
}

func detectLockfileKind(snap *RepoSnapshot) string {
	switch {
	case snap.HasPnpmLock:
		return "pnpm"
	case snap.HasYarnLock:
		return "yarn"
	case snap.HasPackageLock:
		return "npm"
	default:
		return "none"
	}
}

func parsePackageManager(packageJSON string) string {
	if packageJSON == "" {
		return ""
	}
	var pkg struct {
		PackageManager string `json:"packageManager"`
	}
	if err := json.Unmarshal([]byte(packageJSON), &pkg); err != nil {
		return ""
	}
	return strings.TrimSpace(pkg.PackageManager)
}

func parseNodeMajorRequired(root, packageJSON string) int {
	for _, name := range []string{".nvmrc", ".node-version"} {
		if v := readTrunc(filepath.Join(root, name), 32); v != "" {
			if major := majorFromVersionString(v); major > 0 {
				return major
			}
		}
	}
	if packageJSON == "" {
		return 0
	}
	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	if err := json.Unmarshal([]byte(packageJSON), &pkg); err == nil {
		if major := majorFromVersionString(pkg.Engines.Node); major > 0 {
			return major
		}
	}
	if pm := parsePackageManager(packageJSON); pm != "" {
		// e.g. pnpm@9.15.0, npm@10.9.0
		if major := majorFromVersionString(pm); major > 0 {
			return major
		}
	}
	return 0
}

var phpRequireRe = regexp.MustCompile(`(?i)"php"\s*:\s*"([^"]+)"`)

func parsePHPVersionRequired(composerJSON string) string {
	if composerJSON == "" {
		return ""
	}
	m := phpRequireRe.FindStringSubmatch(composerJSON)
	if len(m) < 2 {
		return ""
	}
	constraint := strings.TrimSpace(m[1])
	return minPHPFromConstraint(constraint)
}

func minPHPFromConstraint(constraint string) string {
	constraint = strings.TrimSpace(constraint)
	// ^8.4, >=8.4.1, 8.4.*
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`(\d+)\.(\d+)(?:\.(\d+))?`),
	} {
		m := re.FindStringSubmatch(constraint)
		if len(m) >= 3 {
			major, _ := strconv.Atoi(m[1])
			minor, _ := strconv.Atoi(m[2])
			if major >= 7 && major <= 9 {
				return strconv.Itoa(major) + "." + strconv.Itoa(minor)
			}
		}
	}
	return ""
}

func majorFromVersionString(raw string) int {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "v")
	raw = strings.TrimPrefix(raw, "^")
	raw = strings.TrimPrefix(raw, ">=")
	raw = strings.TrimPrefix(raw, "~")
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '.' || r == ' ' || r == '-' || r == '@'
	})
	if len(parts) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(parts[0])
	return n
}

// suggestedPHPVersion returns the best PHP version for this repo snapshot.
func (snap *RepoSnapshot) suggestedPHPVersion() string {
	if snap == nil {
		return ""
	}
	if v := strings.TrimSpace(snap.PHPVersionRequired); v != "" {
		return v
	}
	switch snap.FrameworkHint {
	case "laravel":
		return "8.4"
	case "wordpress":
		return "8.2"
	case "symfony", "php":
		return "8.3"
	}
	return ""
}

// suggestedNodeAppKey returns appstore node key (nodejs20, nodejs18).
func (snap *RepoSnapshot) suggestedNodeAppKey() string {
	if snap == nil {
		return "nodejs20"
	}
	major := snap.NodeMajorRequired
	if major <= 0 {
		if snap.UsesCatalog || strings.Contains(snap.PackageJSON, `"vite"`) {
			return "nodejs20"
		}
		return "nodejs20"
	}
	switch {
	case major >= 20:
		return "nodejs20"
	case major >= 18:
		return "nodejs18"
	default:
		return "nodejs20"
	}
}

// suggestedRustAppKey returns appstore rust key (rust184, rust183).
func (snap *RepoSnapshot) suggestedRustAppKey() string {
	if snap == nil {
		return "rust184"
	}
	if v := parseRustToolchainVersion(filepath.Join(snap.ClonePath, "rust-toolchain.toml")); v != "" {
		return rustAppKeyFromVersion(v)
	}
	return "rust184"
}

func rustAppKeyFromVersion(v string) string {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	parts := strings.Split(v, ".")
	if len(parts) >= 2 {
		return "rust" + parts[0] + parts[1]
	}
	return "rust184"
}

func parseRustToolchainVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "channel") {
			// channel = "1.84.0"
			if idx := strings.Index(line, `"`); idx >= 0 {
				rest := line[idx+1:]
				if end := strings.Index(rest, `"`); end >= 0 {
					return rest[:end]
				}
			}
		}
	}
	return ""
}

// applyRepoEnvHints adjusts deploy plan from clone analysis (deterministic, not AI).
func applyRepoEnvHints(plan *DeployPlan, snap *RepoSnapshot) {
	if plan == nil || snap == nil {
		return
	}
	if plan.ProjectType == "static" && snap.HasComposer {
		plan.ProjectType = "php"
	}
	if php := snap.suggestedPHPVersion(); php != "" && planNeedsPHP(*plan) {
		if plan.PhpVersion == "" || plan.PhpVersion == "8.3" || plan.Confidence == "heuristic" {
			plan.PhpVersion = php
		}
	}
	if snap.IsMonorepo && snap.PrimaryAppPath != "" {
		if snap.PrimaryAppOutDir != "" && !nextNeedsPM2(snap.PrimaryAppOutDir) {
			plan.ProjectType = "static"
			plan.PhpVersion = "static"
			plan.Framework = "nextjs"
			if plan.DocumentRoot == "" {
				plan.DocumentRoot = snap.PrimaryAppOutDir
			}
			plan.Summary = "Monorepo：仅构建主应用 " + snap.PrimaryAppFilter + "，Nginx 托管 " + snap.PrimaryAppOutDir
		} else if snap.HasNodeServer {
			plan.ProjectType = "node"
			plan.UsePM2 = true
			plan.PhpVersion = "static"
			plan.Framework = "nextjs"
			if plan.DocumentRoot == "" {
				plan.DocumentRoot = snap.PrimaryAppPath
			}
			plan.Summary = "Monorepo：构建 " + snap.PrimaryAppFilter + "，PM2 运行 next start + Nginx 反代"
		}
	}
	_ = snap.suggestedNodeAppKey()
}
