package aichat

import "strings"

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func isCursorAPI(cfg aiConfig) bool {
	if normalizeProvider(cfg.Provider) == "cursor" {
		return true
	}
	return strings.Contains(strings.ToLower(cfg.BaseURL), "cursor.com")
}

func isAnthropicAPI(cfg aiConfig) bool {
	if normalizeProvider(cfg.Provider) == "claude" {
		return true
	}
	return strings.Contains(strings.ToLower(cfg.BaseURL), "anthropic.com")
}

func chatProviderHint(cfg aiConfig) string {
	switch {
	case isCursorAPI(cfg):
		return "Cursor 请使用服务商「Cursor」，API 地址 https://api.cursor.com（不支持 /chat/completions）"
	case isAnthropicAPI(cfg):
		return "Claude 请使用服务商「Claude」，API 地址 https://api.anthropic.com/v1（不支持 /chat/completions）"
	default:
		return "请检查 AI 服务商与 API 地址是否匹配"
	}
}
