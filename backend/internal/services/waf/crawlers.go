package waf

// CrawlerPreset defines a known bot/crawler with UA match patterns.
type CrawlerPreset struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Icon          string   `json:"icon"`
	DefaultAction string   `json:"default_action"` // allow | block
	Patterns      []string `json:"patterns"`
}

var crawlerPresets = []CrawlerPreset{
	{
		ID: "google", Name: "Google", Icon: "google",
		DefaultAction: "allow",
		Patterns:      []string{"Googlebot", "Google-InspectionTool", "GoogleOther", "Mediapartners-Google", "AdsBot-Google"},
	},
	{
		ID: "apple", Name: "Apple", Icon: "apple",
		DefaultAction: "allow",
		Patterns:      []string{"Applebot", "Applebot-Extended"},
	},
	{
		ID: "openai", Name: "OpenAI", Icon: "openai",
		DefaultAction: "allow",
		Patterns:      []string{"GPTBot", "ChatGPT-User", "OAI-SearchBot"},
	},
	{
		ID: "anthropic", Name: "Anthropic", Icon: "anthropic",
		DefaultAction: "allow",
		Patterns:      []string{"ClaudeBot", "anthropic-ai", "Claude-Web"},
	},
	{
		ID: "bing", Name: "Bing", Icon: "bing",
		DefaultAction: "allow",
		Patterns:      []string{"bingbot", "BingPreview", "msnbot"},
	},
	{
		ID: "baidu", Name: "Baidu", Icon: "baidu",
		DefaultAction: "allow",
		Patterns:      []string{"Baiduspider", "Baiduspider-image", "Baiduspider-render"},
	},
	{
		ID: "meta", Name: "Meta", Icon: "meta",
		DefaultAction: "allow",
		Patterns:      []string{"facebookexternalhit", "Facebot", "Meta-ExternalAgent"},
	},
	{
		ID: "twitter", Name: "Twitter/X", Icon: "twitter",
		DefaultAction: "allow",
		Patterns:      []string{"Twitterbot", "X-Twitterbot"},
	},
	{
		ID: "yandex", Name: "Yandex", Icon: "yandex",
		DefaultAction: "allow",
		Patterns:      []string{"YandexBot", "YandexImages", "YandexRenderResourcesBot"},
	},
	{
		ID: "generic_scraper", Name: "Generic Scraper", Icon: "scraper",
		DefaultAction: "allow",
		Patterns:      []string{"Scrapy", "HttpClient", "python-requests", "curl/", "wget/", "libwww-perl", "Go-http-client"},
	},
}

func ListCrawlerPresets() []CrawlerPreset {
	out := make([]CrawlerPreset, len(crawlerPresets))
	copy(out, crawlerPresets)
	return out
}

func GetCrawlerPreset(id string) (CrawlerPreset, bool) {
	for _, p := range crawlerPresets {
		if p.ID == id {
			return p, true
		}
	}
	return CrawlerPreset{}, false
}

func CrawlerPatternRegex(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}
	parts := make([]string, len(patterns))
	for i, p := range patterns {
		parts[i] = escapeNginxRegex(p)
	}
	return "(?i)(" + joinPatterns(parts) + ")"
}

func joinPatterns(parts []string) string {
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "|" + parts[i]
	}
	return out
}
