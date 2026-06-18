package aisite

import "strings"

type EnvRequirement struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Installed   bool   `json:"installed"`
	Selected    bool   `json:"selected"`
}

func envAppLabel(key string) string {
	switch key {
	case "git":
		return "Git"
	case "nginx":
		return "Nginx"
	case "openresty":
		return "OpenResty"
	case "composer":
		return "Composer"
	case "nodejs20":
		return "Node.js 20"
	case "nodejs18":
		return "Node.js 18"
	case "nodejs":
		return "Node.js"
	case "mysql":
		return "MySQL"
	case "mariadb":
		return "MariaDB"
	case "docker":
		return "Docker"
	case "pm2":
		return "PM2"
	default:
		if strings.HasPrefix(key, "php") {
			ver := strings.TrimPrefix(key, "php")
			if ver != "" {
				return "PHP " + ver[0:1] + "." + ver[1:]
			}
		}
		return key
	}
}

func envAppDescription(key string) string {
	switch key {
	case "git":
		return "克隆 GitHub 仓库"
	case "nginx", "openresty":
		return "网站虚拟主机与反向代理"
	case "composer":
		return "PHP 依赖安装（Laravel / Symfony 等）"
	case "nodejs20", "nodejs18", "nodejs":
		return "前端构建 npm install / npm run build"
	case "mysql", "mariadb":
		return "应用数据库（Laravel 等）"
	case "docker":
		return "Docker Compose 部署"
	case "pm2":
		return "Node.js 进程守护"
	default:
		if strings.HasPrefix(key, "php") {
			return "PHP 运行时"
		}
		return ""
	}
}

func (s *Service) buildEnvRequirements(plan DeployPlan, snap *RepoSnapshot) []EnvRequirement {
	panel := s.collectPanelContext()
	keys := s.requiredDeployApps(plan, snap)
	if !gitAvailable() && !panel.GitAvailable {
		keys = append([]string{"git"}, keys...)
	}
	keys = dedupeKeys(keys)
	out := make([]EnvRequirement, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		installed := deployAppReady(s, key)
		if key == "git" {
			installed = gitAvailable() || panel.GitAvailable
		}
		required := true
		if key == "docker" && !plan.UseDocker {
			required = false
		}
		if key == "pm2" && !plan.UsePM2 {
			required = false
		}
		out = append(out, EnvRequirement{
			Key:         key,
			Label:       envAppLabel(key),
			Description: envAppDescription(key),
			Required:    required,
			Installed:   installed,
			Selected:    !installed || required,
		})
	}
	return out
}

func filterAppsBySelection(all []string, selected []string) []string {
	if len(selected) == 0 {
		return all
	}
	allow := make(map[string]struct{}, len(selected))
	for _, k := range selected {
		allow[strings.TrimSpace(k)] = struct{}{}
	}
	var out []string
	for _, k := range all {
		if _, ok := allow[k]; ok {
			out = append(out, k)
		}
	}
	return out
}
