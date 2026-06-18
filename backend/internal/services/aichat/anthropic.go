package aichat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type anthropicChatRequest struct {
	Model     string         `json:"model"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system,omitempty"`
	Messages  []anthropicMsg `json:"messages"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicChatResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Service) callAnthropicMessages(cfg aiConfig, messages []Message) (string, error) {
	return s.callAnthropicMessagesVision(cfg, messages)
}

func (s *Service) callAnthropicMessagesVision(cfg aiConfig, messages []Message) (string, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com/v1"
	}
	if cfg.APIKey == "" {
		return "", fmt.Errorf("Claude API Key 未配置")
	}

	var system string
	chatMsgs := make([]anthropicMsg, 0, len(messages))
	useMultimodal := messagesHaveImages(messages)
	if useMultimodal {
		system, mm := toAnthropicMessages(messages)
		if len(mm) == 0 {
			return "", fmt.Errorf("没有可发送的消息")
		}
		body, _ := json.Marshal(anthropicChatRequestMultimodal{
			Model:     cfg.Model,
			MaxTokens: 8192,
			System:    system,
			Messages:  mm,
		})
		return s.doAnthropicRequest(base, cfg.APIKey, body)
	}
	for _, m := range messages {
		switch m.Role {
		case "system":
			if system == "" {
				system = m.Content
			} else {
				system += "\n\n" + m.Content
			}
		case "user", "assistant":
			chatMsgs = append(chatMsgs, anthropicMsg{Role: m.Role, Content: m.Content})
		}
	}
	if len(chatMsgs) == 0 {
		return "", fmt.Errorf("没有可发送的消息")
	}

	body, _ := json.Marshal(anthropicChatRequest{
		Model:     cfg.Model,
		MaxTokens: 8192,
		System:    system,
		Messages:  chatMsgs,
	})
	return s.doAnthropicRequest(base, cfg.APIKey, body)
}

func (s *Service) doAnthropicRequest(base, apiKey string, body []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, base+"/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Claude 请求失败: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		msg := parseAPIError(raw)
		return "", fmt.Errorf("Claude API error (%d): %s", resp.StatusCode, msg)
	}

	var parsed anthropicChatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("invalid Claude response: %w", err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("Claude API error: %s", parsed.Error.Message)
	}
	var parts []string
	for _, block := range parsed.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("Claude returned empty response")
	}
	return strings.TrimSpace(strings.Join(parts, "\n")), nil
}
