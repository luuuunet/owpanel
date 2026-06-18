package stack

// Definition describes a one-click runtime stack (LNMP/LAMP).
type Definition struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Components  []string `json:"components"`
}

var catalog = []Definition{
	{
		Key:         "lnmp",
		Name:        "LNMP",
		Description: "Nginx + MySQL + PHP 8.3（WordPress / 通用 PHP 站）",
		Components:  []string{"nginx", "mysql", "php83", "certbot", "composer"},
	},
	{
		Key:         "lamp",
		Name:        "LAMP",
		Description: "Apache + MySQL + PHP 8.3",
		Components:  []string{"apache", "mysql", "php83", "certbot"},
	},
	{
		Key:         "lnmp-full",
		Name:        "LNMP 完整版",
		Description: "LNMP + Redis + Docker + Fail2ban",
		Components:  []string{"nginx", "mysql", "php83", "redis", "docker", "certbot", "composer", "fail2ban"},
	},
	{
		Key:         "web",
		Name:        "Web 仅 Nginx",
		Description: "仅安装并配置 Nginx",
		Components:  []string{"nginx"},
	},
}

func List() []Definition {
	out := make([]Definition, len(catalog))
	copy(out, catalog)
	return out
}

func Get(key string) (Definition, bool) {
	for _, d := range catalog {
		if d.Key == key {
			return d, true
		}
	}
	return Definition{}, false
}

func Components(key string) []string {
	if d, ok := Get(key); ok {
		return append([]string(nil), d.Components...)
	}
	return nil
}
