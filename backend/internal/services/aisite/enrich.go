package aisite

import (
	"fmt"
	"runtime"
	"strings"
)

func enrichPlanDefaults(plan *DeployPlan, snap *RepoSnapshot, panel PanelContext) {
	applyRepoEnvHints(plan, snap)
	if !plan.EnableWebhook {
		plan.EnableWebhook = true
	}
	if !plan.RollbackOnFail {
		plan.RollbackOnFail = true
	}
	if snap.HasNodeServer && !plan.UseDocker && !plan.UsePM2 {
		plan.UsePM2 = true
		plan.ProjectType = "node"
		plan.PhpVersion = "static"
	}
	if (snap.HasDockerfile || snap.HasDockerCompose) && panel.DockerAvailable && !plan.UsePM2 {
		if plan.ProjectType == "" || plan.ProjectType == "static" {
			plan.UseDocker = true
			plan.ProjectType = "docker"
		}
	}
	if plan.Framework == "laravel" && plan.DocumentRoot == "" {
		plan.DocumentRoot = "public"
	}
	if plan.UseDocker && plan.DeployScript == "" {
		plan.DeployScript = dockerDeployScript(snap)
	}
	if plan.NodePort <= 0 && plan.UsePM2 {
		plan.NodePort = 31080
	}
}

func dockerDeployScript(snap *RepoSnapshot) string {
	if runtime.GOOS == "windows" {
		if snap.HasDockerCompose {
			return "git clone --depth 1 -b {{branch}} {{repo}} .\r\ndocker compose up -d --build\r\n"
		}
		return "git clone --depth 1 -b {{branch}} {{repo}} .\r\ndocker build -t op-site . \r\ndocker run -d --restart unless-stopped --name op-site-{{domain}} -p 8080:80 op-site\r\n"
	}
	if snap.HasDockerCompose {
		return "set -e\n" + gitCloneIntoSiteRootBlock() + "docker compose up -d --build\n"
	}
	return "set -e\n" + gitCloneIntoSiteRootBlock() + "docker build -t op-site-{{domain}} .\ndocker run -d --restart unless-stopped --name op-site-{{domain}} -p 8080:80 op-site-{{domain}}\n"
}

func applyPlanPlaceholders(script, repo, branch, domain, siteRoot string) string {
	script = strings.ReplaceAll(script, "{{repo}}", repo)
	script = strings.ReplaceAll(script, "{{branch}}", branch)
	script = strings.ReplaceAll(script, "{{domain_host}}", strings.TrimSpace(domain))
	script = strings.ReplaceAll(script, "{{domain}}", sanitizeDockerName(domain))
	if siteRoot != "" {
		script = strings.ReplaceAll(script, "{{root}}", siteRoot)
	}
	return script
}

func sanitizeDockerName(domain string) string {
	s := strings.NewReplacer(".", "-", ":", "-", "/", "-").Replace(domain)
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

func webhookNote(savedHookURL string) string {
	if savedHookURL == "" {
		return ""
	}
	return fmt.Sprintf("GitHub WebHook 已启用，推送后自动部署: %s", savedHookURL)
}

func branchFromSnap(req DeployRequest, snap *RepoSnapshot) string {
	if snap != nil && strings.TrimSpace(snap.Branch) != "" {
		return snap.Branch
	}
	if b := strings.TrimSpace(req.Branch); b != "" {
		return b
	}
	return "main"
}

// mergeWizardPlanChoices keeps user-confirmed wizard settings after a fresh AI/heuristic analysis.
func mergeWizardPlanChoices(plan *DeployPlan, wizard DeployPlan) {
	if plan == nil {
		return
	}
	if d := strings.TrimSpace(wizard.Domain); d != "" {
		plan.Domain = d
	}
	if v := strings.TrimSpace(wizard.PhpVersion); v != "" {
		plan.PhpVersion = v
	}
	plan.NeedDatabase = wizard.NeedDatabase
	if dr := strings.TrimSpace(wizard.DocumentRoot); dr != "" {
		plan.DocumentRoot = dr
	}
	plan.UseDocker = wizard.UseDocker
	plan.UsePM2 = wizard.UsePM2
	if wizard.NodePort > 0 {
		plan.NodePort = wizard.NodePort
	}
	plan.EnableWebhook = wizard.EnableWebhook
	plan.RollbackOnFail = wizard.RollbackOnFail
	if ds := strings.TrimSpace(wizard.DeployScript); ds != "" {
		plan.DeployScript = ds
	}
	if notes := strings.TrimSpace(wizard.PostNotes); notes != "" {
		plan.PostNotes = notes
	}
	if fw := strings.TrimSpace(wizard.Framework); fw != "" && strings.TrimSpace(plan.Framework) == "" {
		plan.Framework = fw
	}
}
