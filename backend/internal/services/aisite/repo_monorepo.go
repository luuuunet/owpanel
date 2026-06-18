package aisite

import (
	"path/filepath"
	"regexp"
	"strings"
)

var nextOutputExportRe = regexp.MustCompile(`output\s*:\s*['"]export['"]`)
var nextOutputStandaloneRe = regexp.MustCompile(`output\s*:\s*['"]standalone['"]`)

// nextNeedsPM2 reports whether Next.js output must run via PM2 (not nginx static root).
func nextNeedsPM2(outDir string) bool {
	outDir = strings.Trim(strings.TrimSpace(outDir), "/")
	if outDir == "" {
		return true
	}
	if strings.HasSuffix(outDir, ".next") && !strings.Contains(outDir, "standalone") {
		return true
	}
	return false
}

func detectMonorepoLayout(root string) (isMonorepo bool, hasTurbo bool, filter, appPath, outDir string, buildEnv []string) {
	hasTurbo = fileExists(filepath.Join(root, "turbo.json"))
	hasWorkspace := fileExists(filepath.Join(root, "pnpm-workspace.yaml")) ||
		fileExists(filepath.Join(root, "package-lock.json")) && fileExists(filepath.Join(root, "packages"))
	isMonorepo = hasTurbo || hasWorkspace || fileExists(filepath.Join(root, "apps"))

	if !isMonorepo {
		return false, hasTurbo, "", "", "", nil
	}

	// Prefer marketing/main site apps for single-domain deploy.
	for _, name := range []string{"site", "web", "www", "app", "main", "frontend"} {
		rel := filepath.Join("apps", name)
		if fileExists(filepath.Join(root, rel, "package.json")) {
			filter, appPath = name, rel
			outDir = detectNextOutDir(root, rel)
			buildEnv = scanBuildEnvHints(root, rel)
			return true, hasTurbo, filter, appPath, outDir, buildEnv
		}
	}

	// First app under apps/ with a build script.
	appsDir := filepath.Join(root, "apps")
	if entries, err := filepath.Glob(filepath.Join(appsDir, "*", "package.json")); err == nil && len(entries) > 0 {
		rel, _ := filepath.Rel(root, filepath.Dir(entries[0]))
		rel = filepath.ToSlash(rel)
		filter = filepath.Base(rel)
		appPath = rel
		outDir = detectNextOutDir(root, rel)
		buildEnv = scanBuildEnvHints(root, rel)
		return true, hasTurbo, filter, appPath, outDir, buildEnv
	}

	return isMonorepo, hasTurbo, "", "", "", scanBuildEnvHints(root, "")
}

func detectNextOutDir(root, appRel string) string {
	appRel = filepath.ToSlash(appRel)
	cfgPaths := []string{
		filepath.Join(root, appRel, "next.config.mjs"),
		filepath.Join(root, appRel, "next.config.js"),
		filepath.Join(root, appRel, "next.config.ts"),
	}
	for _, p := range cfgPaths {
		cfg := readTrunc(p, 8000)
		if cfg == "" {
			continue
		}
		if nextOutputExportRe.MatchString(cfg) {
			return appRel + "/out"
		}
		if nextOutputStandaloneRe.MatchString(cfg) {
			return appRel + "/.next/standalone"
		}
		if strings.Contains(strings.ToLower(cfg), "next") {
			return "" // SSR/hybrid — serve via PM2 + next start, not nginx static root
		}
	}
	if fileExists(filepath.Join(root, appRel, "package.json")) {
		pkg := readTrunc(filepath.Join(root, appRel, "package.json"), 4000)
		if strings.Contains(strings.ToLower(pkg), `"next"`) {
			return "" // default Next.js build is not static export
		}
	}
	return appRel + "/dist"
}

func scanBuildEnvHints(root, appRel string) []string {
	seen := map[string]bool{}
	var keys []string
	add := func(k string) {
		k = strings.TrimSpace(k)
		if k == "" || seen[k] {
			return
		}
		seen[k] = true
		keys = append(keys, k)
	}

	for _, name := range []string{".env.example", ".env.production.example", "env.example"} {
		p := filepath.Join(root, appRel, name)
		if appRel == "" {
			p = filepath.Join(root, name)
		}
		for _, k := range parseEnvExampleKeys(readTrunc(p, 8000)) {
			add(k)
		}
	}

	// Common Next.js monorepo cross-origin env vars (Prisma docs site, etc.).
	for _, k := range []string{"DOCS_ORIGIN", "BLOG_ORIGIN", "NEXT_DOCS_ORIGIN", "NEXT_BLOG_ORIGIN", "NEXT_PUBLIC_SITE_URL", "NEXT_PUBLIC_APP_URL", "SITE_URL", "VERCEL_URL"} {
		add(k)
	}

	cfgPaths := []string{
		filepath.Join(root, appRel, "next.config.mjs"),
		filepath.Join(root, appRel, "next.config.js"),
	}
	for _, p := range cfgPaths {
		if appRel == "" {
			continue
		}
		cfg := readTrunc(p, 12000)
		for _, m := range regexp.MustCompile(`process\.env\.([A-Z0-9_]+)`).FindAllStringSubmatch(cfg, -1) {
			if len(m) > 1 {
				add(m[1])
			}
		}
	}
	return keys
}

func parseEnvExampleKeys(content string) []string {
	var keys []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "="); i > 0 {
			keys = append(keys, strings.TrimSpace(line[:i]))
		}
	}
	return keys
}

func defaultBuildEnvValue(key, domainHost string) string {
	domainHost = strings.TrimSpace(domainHost)
	if domainHost == "" {
		domainHost = "localhost"
	}
	base := "https://" + domainHost
	switch key {
	case "DOCS_ORIGIN", "NEXT_DOCS_ORIGIN":
		return "https://docs." + domainHost
	case "BLOG_ORIGIN", "NEXT_BLOG_ORIGIN":
		return "https://blog." + domainHost
	case "NEXT_PUBLIC_SITE_URL", "NEXT_PUBLIC_APP_URL", "SITE_URL", "APP_URL":
		return base
	case "NODE_ENV":
		return "production"
	default:
		if strings.HasSuffix(key, "_ORIGIN") || strings.HasSuffix(key, "_URL") {
			return base
		}
		return ""
	}
}

func enrichMonorepoSnapshot(snap *RepoSnapshot) {
	if snap == nil || snap.ClonePath == "" {
		return
	}
	isMono, hasTurbo, filter, appPath, outDir, buildEnv := detectMonorepoLayout(snap.ClonePath)
	snap.IsMonorepo = isMono
	snap.HasTurbo = hasTurbo
	snap.PrimaryAppFilter = filter
	snap.PrimaryAppPath = appPath
	snap.PrimaryAppOutDir = outDir
	snap.BuildEnvKeys = buildEnv
	if isMono && filter != "" {
		snap.FrameworkHint = "nextjs"
		snap.HasNodeServer = true
	}
}

func (snap *RepoSnapshot) monorepoBuildExportsBlock() string {
	if snap == nil || len(snap.BuildEnvKeys) == 0 {
		return `export NODE_ENV=production
SITE_DOMAIN="{{domain_host}}"
export DOCS_ORIGIN="${DOCS_ORIGIN:-https://docs.${SITE_DOMAIN}}"
export BLOG_ORIGIN="${BLOG_ORIGIN:-https://blog.${SITE_DOMAIN}}"
`
	}
	var b strings.Builder
	b.WriteString("export NODE_ENV=production\n")
	b.WriteString(`SITE_DOMAIN="{{domain_host}}"
`)
	for _, key := range snap.BuildEnvKeys {
		def := defaultBuildEnvValue(key, "{{domain_host}}")
		if def == "" {
			continue
		}
		b.WriteString("export ")
		b.WriteString(key)
		b.WriteString("=\"${")
		b.WriteString(key)
		b.WriteString(":-" + def + "}\"\n")
	}
	return b.String()
}
