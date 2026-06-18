package edgeworker

import (
	"os/exec"
	"strings"
)

type RuntimeInfo struct {
	Runtime       string `json:"runtime"`        // openresty, nginx
	NjsAvailable  bool   `json:"njs_available"`
	LuaAvailable  bool   `json:"lua_available"`
	ActiveWebSrv  string `json:"active_web_server"`
	Message       string `json:"message,omitempty"`
	RecommendOpen bool   `json:"recommend_openresty"`
}

func (s *Service) DetectRuntime() RuntimeInfo {
	active := "nginx"
	if s.getActiveWS != nil {
		if v := strings.TrimSpace(s.getActiveWS()); v != "" {
			active = v
		}
	}
	info := RuntimeInfo{
		Runtime:      active,
		ActiveWebSrv: active,
	}
	switch active {
	case "openresty":
		info.LuaAvailable = true
		info.Runtime = "openresty"
		info.NjsAvailable = nginxModuleAvailable("http_js")
	case "nginx":
		info.Runtime = "nginx"
		info.LuaAvailable = false
		info.NjsAvailable = nginxModuleAvailable("http_js")
		if !info.NjsAvailable {
			info.RecommendOpen = true
			info.Message = "OpenResty (LuaJIT) is recommended for edge Workers. Install OpenResty from Software Store, or enable nginx njs module."
		}
	default:
		info.RecommendOpen = true
		info.Message = "Edge Workers require Nginx or OpenResty. Install OpenResty from Software Store for best Lua support."
	}
	return info
}

func nginxModuleAvailable(module string) bool {
	out, err := exec.Command("nginx", "-V").CombinedOutput()
	if err != nil {
		out, _ = exec.Command("openresty", "-V").CombinedOutput()
	}
	text := string(out)
	return strings.Contains(text, module) || strings.Contains(text, "--add-module") && strings.Contains(strings.ToLower(text), module)
}
