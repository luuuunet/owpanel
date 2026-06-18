package platform

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Info describes the host OS and what the panel can manage on it.
type Info struct {
	GOOS            string            `json:"goos"`
	GOARCH          string            `json:"goarch"`
	OSFamily        string            `json:"os_family"`
	OSName          string            `json:"os_name"`
	OSVersion       string            `json:"os_version"`
	PackageManager  string            `json:"package_manager"`
	InitSystem      string            `json:"init_system"`
	DeployProfile   string            `json:"deploy_profile"`
	Features        FeatureFlags      `json:"features"`
	RecommendedNote string            `json:"recommended_note,omitempty"`
}

// FeatureFlags indicates which integrations work on this host.
type FeatureFlags struct {
	SystemPackages  bool `json:"system_packages"`
	SystemdServices bool `json:"systemd_services"`
	FirewallApply   bool `json:"firewall_apply"`
	FTPSync         bool `json:"ftp_sync"`
	MailStack       bool `json:"mail_stack"`
	PHPExtensionApt bool `json:"php_extension_apt"`
	WingetInstall   bool `json:"winget_install"`
	DockerApps      bool `json:"docker_apps"`
}

func Detect() Info {
	info := Info{
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
	switch runtime.GOOS {
	case "linux":
		info.PackageManager = detectLinuxPackageManager()
		info.InitSystem = detectInitSystem()
		parseOSRelease(&info)
		info.DeployProfile = "linux-server"
		info.Features = FeatureFlags{
			SystemPackages:  info.PackageManager != "",
			SystemdServices: info.InitSystem == "systemd",
			FirewallApply:   true,
			FTPSync:         true,
			MailStack:       true,
			PHPExtensionApt: info.PackageManager == "apt",
			DockerApps:      commandExists("docker"),
		}
		info.RecommendedNote = "生产环境推荐：Ubuntu / Debian / CentOS / Rocky / AlmaLinux"
	case "windows":
		info.OSFamily = "windows"
		info.OSName = "Windows"
		info.InitSystem = "windows"
		info.PackageManager = "winget"
		if !commandExists("winget") {
			info.PackageManager = "none"
		}
		info.DeployProfile = "windows-desktop"
		info.Features = FeatureFlags{
			SystemPackages:  info.PackageManager == "winget",
			WingetInstall:   info.PackageManager == "winget",
			DockerApps:      commandExists("docker"),
		}
		info.RecommendedNote = "Windows 适合本地开发；防火墙/FTP/邮件等系统级能力会降级"
	default:
		info.OSFamily = runtime.GOOS
		info.OSName = runtime.GOOS
		info.DeployProfile = "other"
	}
	return info
}

func PackageManager() string {
	if runtime.GOOS == "windows" {
		if commandExists("winget") {
			return "winget"
		}
		return ""
	}
	if runtime.GOOS == "linux" {
		return detectLinuxPackageManager()
	}
	return ""
}

func detectLinuxPackageManager() string {
	for _, c := range []struct{ bin, name string }{
		{"apt-get", "apt"},
		{"dnf", "dnf"},
		{"yum", "yum"},
	} {
		if commandExists(c.bin) {
			return c.name
		}
	}
	return ""
}

func detectInitSystem() string {
	if commandExists("systemctl") {
		return "systemd"
	}
	return ""
}

func parseOSRelease(info *Info) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		info.OSFamily = "linux"
		info.OSName = "Linux"
		return
	}
	defer f.Close()

	vals := map[string]string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		vals[parts[0]] = strings.Trim(parts[1], `"`)
	}

	id := strings.ToLower(vals["ID"])
	idLike := strings.ToLower(vals["ID_LIKE"])
	info.OSName = vals["PRETTY_NAME"]
	if info.OSName == "" {
		info.OSName = vals["NAME"]
	}
	info.OSVersion = vals["VERSION_ID"]

	switch {
	case id == "ubuntu":
		info.OSFamily = "ubuntu"
	case id == "debian":
		info.OSFamily = "debian"
	case id == "centos" || id == "rocky" || id == "almalinux" || id == "rhel":
		info.OSFamily = id
	case strings.Contains(idLike, "rhel") || strings.Contains(idLike, "centos") || strings.Contains(idLike, "fedora"):
		info.OSFamily = "rhel"
	case strings.Contains(idLike, "debian") || strings.Contains(idLike, "ubuntu"):
		info.OSFamily = "debian"
	default:
		info.OSFamily = id
		if info.OSFamily == "" {
			info.OSFamily = "linux"
		}
	}
}

func commandExists(name string) bool {
	if filepath.IsAbs(name) || strings.Contains(name, string(os.PathSeparator)) {
		_, err := os.Stat(name)
		return err == nil
	}
	_, err := exec.LookPath(name)
	return err == nil
}
