package aichat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type ModelOption struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
}

func (s *Service) ListModels(provider, apiKeyOverride, baseURLOverride string) ([]ModelOption, error) {
	cfg, err := s.configForModels(provider, apiKeyOverride, baseURLOverride)
	if err != nil {
		return nil, err
	}
	cfg.Provider = normalizeProvider(cfg.Provider)

	if isCursorAPI(cfg) {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("请先配置 Cursor API Key")
		}
		return fetchCursorModels(cfg)
	}
	if isAnthropicAPI(cfg) {
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("请先配置 Claude API Key")
		}
		return fetchAnthropicModels(cfg)
	}

	switch cfg.Provider {
	case "ollama":
		return fetchOllamaModels(cfg)
	case "huggingface":
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("请先部署 Hugging Face AI 或配置 API 地址")
		}
		return fetchOpenAICompatibleModels(cfg)
	case "openai", "deepseek", "custom", "claude", "cursor":
		if cfg.Provider != "custom" && cfg.Provider != "ollama" && cfg.Provider != "huggingface" && cfg.APIKey == "" {
			return nil, fmt.Errorf("请先配置 API Key")
		}
		if cfg.Provider == "custom" && cfg.BaseURL == "" {
			return nil, fmt.Errorf("请先配置 API 地址")
		}
		return fetchOpenAICompatibleModels(cfg)
	default:
		return nil, fmt.Errorf("不支持同步该服务商的模型列表")
	}
}

func (s *Service) ListCursorModels(apiKeyOverride, baseURLOverride string) ([]ModelOption, error) {
	return s.ListModels("cursor", apiKeyOverride, baseURLOverride)
}

func (s *Service) configForModels(provider, apiKeyOverride, baseURLOverride string) (aiConfig, error) {
	cfg, err := s.loadConfig()
	if err != nil {
		return aiConfig{}, err
	}
	if p := strings.TrimSpace(provider); p != "" {
		cfg.Provider = p
	}
	if apiKeyOverride != "" {
		cfg.APIKey = apiKeyOverride
	}
	if baseURLOverride != "" {
		cfg.BaseURL = strings.TrimRight(baseURLOverride, "/")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL(cfg.Provider)
	}
	return cfg, nil
}

type openAIModelsResponse struct {
	Data []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
}

type anthropicModelsResponse struct {
	Data []struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		Type        string `json:"type"`
	} `json:"data"`
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func fetchOpenAICompatibleModels(cfg aiConfig) ([]ModelOption, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		return nil, fmt.Errorf("API 地址未配置")
	}

	req, err := http.NewRequest(http.MethodGet, base+"/models", nil)
	if err != nil {
		return nil, err
	}
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("模型列表请求失败: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("模型列表错误 (%d): %s", resp.StatusCode, parseAPIError(raw))
	}

	var parsed openAIModelsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("无效的模型列表响应: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("官方 API 未返回可用模型")
	}

	filter := chatModelFilter(cfg.Provider)
	seen := make(map[string]struct{})
	out := make([]ModelOption, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" || !filter(id) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, ModelOption{ID: id, DisplayName: id})
	}
	if len(out) == 0 {
		for _, item := range parsed.Data {
			id := strings.TrimSpace(item.ID)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, ModelOption{ID: id, DisplayName: id})
		}
	}
	sortModels(out)
	return out, nil
}

func fetchAnthropicModels(cfg aiConfig) ([]ModelOption, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com/v1"
	}

	req, err := http.NewRequest(http.MethodGet, base+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Claude 模型列表请求失败: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Claude 模型列表错误 (%d): %s", resp.StatusCode, parseAPIError(raw))
	}

	var parsed anthropicModelsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("无效的 Claude 模型列表响应: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("Claude API 未返回可用模型")
	}

	seen := make(map[string]struct{})
	out := make([]ModelOption, 0, len(parsed.Data))
	for _, item := range parsed.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(item.DisplayName)
		if name == "" {
			name = id
		}
		out = append(out, ModelOption{ID: id, DisplayName: name})
	}
	sortModels(out)
	return out, nil
}

func fetchOllamaModels(cfg aiConfig) ([]ModelOption, error) {
	base := ollamaRoot(cfg.BaseURL)
	if base == "" {
		base = "http://127.0.0.1:11434"
	}

	if models, err := fetchOllamaOpenAIModels(base + "/v1"); err == nil && len(models) > 0 {
		return models, nil
	}
	return fetchOllamaTags(base)
}

func fetchOllamaOpenAIModels(baseV1 string) ([]ModelOption, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseV1, "/")+"/models", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var parsed openAIModelsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	out := make([]ModelOption, 0, len(parsed.Data))
	seen := make(map[string]struct{})
	for _, item := range parsed.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, ModelOption{ID: id, DisplayName: id})
	}
	sortModels(out)
	return out, nil
}

func fetchOllamaTags(base string) ([]ModelOption, error) {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(base, "/")+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ollama 模型列表请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Ollama 模型列表错误 (%d): %s", resp.StatusCode, parseAPIError(raw))
	}
	var parsed ollamaTagsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("无效的 Ollama 模型列表响应: %w", err)
	}
	if len(parsed.Models) == 0 {
		return nil, fmt.Errorf("Ollama 未安装任何模型，请先执行 ollama pull")
	}
	out := make([]ModelOption, 0, len(parsed.Models))
	seen := make(map[string]struct{})
	for _, item := range parsed.Models {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			continue
		}
		id := strings.Split(name, ":")[0]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, ModelOption{ID: id, DisplayName: name})
	}
	sortModels(out)
	return out, nil
}

func ollamaRoot(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(base, "/v1") {
		return strings.TrimSuffix(base, "/v1")
	}
	return base
}

func chatModelFilter(provider string) func(string) bool {
	switch provider {
	case "deepseek":
		return func(id string) bool {
			id = strings.ToLower(id)
			return strings.Contains(id, "deepseek")
		}
	case "openai":
		return isOpenAIChatModel
	default:
		return isLikelyChatModel
	}
}

func isOpenAIChatModel(id string) bool {
	id = strings.ToLower(id)
	if isExcludedModel(id) {
		return false
	}
	return strings.HasPrefix(id, "gpt-") ||
		strings.HasPrefix(id, "o1") ||
		strings.HasPrefix(id, "o3") ||
		strings.HasPrefix(id, "o4") ||
		strings.HasPrefix(id, "chatgpt-")
}

func isLikelyChatModel(id string) bool {
	id = strings.ToLower(id)
	if isExcludedModel(id) {
		return false
	}
	return strings.HasPrefix(id, "gpt-") ||
		strings.HasPrefix(id, "claude") ||
		strings.HasPrefix(id, "deepseek") ||
		strings.HasPrefix(id, "gemini") ||
		strings.HasPrefix(id, "qwen") ||
		strings.HasPrefix(id, "llama") ||
		strings.HasPrefix(id, "mistral") ||
		strings.HasPrefix(id, "composer") ||
		strings.HasPrefix(id, "o1") ||
		strings.HasPrefix(id, "o3") ||
		strings.HasPrefix(id, "o4") ||
		strings.HasPrefix(id, "chatgpt-")
}

func isExcludedModel(id string) bool {
	excluded := []string{
		"embed", "embedding", "whisper", "tts", "dall-e", "moderation",
		"audio", "transcribe", "realtime", "davinci", "babbage", "curie", "ada",
	}
	for _, part := range excluded {
		if strings.Contains(id, part) {
			return true
		}
	}
	return false
}

func sortModels(models []ModelOption) {
	sort.Slice(models, func(i, j int) bool {
		return strings.ToLower(models[i].DisplayName) < strings.ToLower(models[j].DisplayName)
	})
}

func parseAPIError(raw []byte) string {
	var errResp struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.Unmarshal(raw, &errResp)
	if errResp.Error.Message != "" {
		return errResp.Error.Message
	}
	if errResp.Message != "" {
		return errResp.Message
	}
	msg := strings.TrimSpace(string(raw))
	if len(msg) > 300 {
		return msg[:300] + "..."
	}
	return msg
}
