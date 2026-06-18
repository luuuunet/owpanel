package cli

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func ShowDefaultInfo(ctx *Context) error {
	all, err := ctx.Settings.GetAll()
	if err != nil {
		return err
	}

	port := ctx.Cfg.Port
	if p := all["panel_port"]; p != "" {
		if n, err := parsePort(p); err == nil {
			port = n
		}
	}

	safePath := strings.Trim(all["panel_safe_path"], "/")
	sslOn := all["panel_ssl"] == "true"
	scheme := "http"
	if sslOn {
		scheme = "https"
	}

	admin, _ := ctx.Auth.FirstAdmin()
	username := "admin"
	if admin != nil {
		username = admin.Username
	}

	basePath := func(host string) string {
		u := fmt.Sprintf("%s://%s:%d", scheme, host, port)
		if safePath != "" {
			u += "/" + safePath
		}
		return u
	}

	var urls []string
	for _, ip := range collectIPv4() {
		urls = append(urls, basePath(ip))
	}
	for _, ip := range collectIPv6() {
		urls = append(urls, basePath(ip))
	}
	if len(urls) == 0 {
		urls = append(urls, basePath("127.0.0.1"))
	}

	rows := []infoRow{
		{label: "Panel URL", value: urls[0]},
		{label: "Local URL", value: basePath("127.0.0.1")},
		{label: "Username", value: username},
		{label: "Password", value: dim("(hidden — use menu option 3 to reset)")},
		{label: "Port", value: fmt.Sprintf("%d", port)},
	}
	if safePath != "" {
		rows = append(rows, infoRow{label: "Entrance", value: "/" + safePath})
	}
	rows = append(rows,
		infoRow{label: "SSL", value: fmt.Sprintf("%v", sslOn)},
		infoRow{label: "Data dir", value: ctx.DataDir},
	)

	printMenuTitle("Panel Information")
	printInfoBox(rows)

	if len(urls) > 1 {
		fmt.Println(dim("  Other addresses:"))
		for _, u := range urls[1:] {
			fmt.Println(dim("    · " + u))
		}
		fmt.Println()
	}

	printHint("Open the URL in your browser and sign in with the credentials above.")
	return nil
}

func parsePort(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func collectIPv4() []string {
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			ip, ok := a.(*net.IPNet)
			if !ok || ip.IP.To4() == nil {
				continue
			}
			v := ip.IP.String()
			if !isPrivate(v) {
				out = append(out, v)
			}
		}
	}
	if len(out) == 0 {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			addrs, _ := iface.Addrs()
			for _, a := range addrs {
				ip, ok := a.(*net.IPNet)
				if !ok || ip.IP.To4() == nil {
					continue
				}
				out = append(out, ip.IP.String())
			}
		}
	}
	return dedupe(out)
}

func collectIPv6() []string {
	var out []string
	ifaces, _ := net.Interfaces()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			ip, ok := a.(*net.IPNet)
			if !ok || ip.IP.To4() != nil || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() {
				continue
			}
			out = append(out, fmt.Sprintf("[%s]", ip.IP.String()))
		}
	}
	return dedupe(out)
}

func isPrivate(ip string) bool {
	p := net.ParseIP(ip)
	if p == nil {
		return true
	}
	return p.IsPrivate() || p.IsLoopback()
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func hostnameIP() string {
	if h, err := os.Hostname(); err == nil {
		if addrs, err := net.LookupIP(h); err == nil {
			for _, a := range addrs {
				if v4 := a.To4(); v4 != nil {
					return v4.String()
				}
			}
		}
	}
	return "127.0.0.1"
}
