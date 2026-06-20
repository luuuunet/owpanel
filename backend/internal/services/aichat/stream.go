package aichat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type StreamChunkHandler func(chunk string) error

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ChatStream streams OpenAI-compatible chat completions. Non-streaming providers fall back to one chunk.
func (s *Service) ChatStream(messages []Message, onChunk StreamChunkHandler) error {
	cfg, err := s.loadConfig()
	if err != nil {
		return err
	}
	if !cfg.Enabled {
		return fmt.Errorf("请先在面板设置中启用 AI 助手")
	}
	if cfg.APIKey == "" && cfg.Provider != "ollama" && cfg.Provider != "huggingface" {
		return fmt.Errorf("未配置 AI API Key")
	}
	cfg.Provider = normalizeProvider(cfg.Provider)

	if isCursorAPI(cfg) || isAnthropicAPI(cfg) {
		reply, err := s.callChatAPI(cfg, messages)
		if err != nil {
			return err
		}
		if reply != "" {
			return onChunk(reply)
		}
		return nil
	}

	body, _ := json.Marshal(openAIChatRequest{
		Model:    cfg.Model,
		Messages: toOpenAIMessages(messages),
		Stream:   true,
	})
	url := cfg.BaseURL + "/chat/completions"
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	if cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 360 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("AI stream failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("AI API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	ct := strings.ToLower(resp.Header.Get("Content-Type"))
	if !strings.Contains(ct, "text/event-stream") {
		raw, _ := io.ReadAll(resp.Body)
		var parsed openAIChatResponse
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return fmt.Errorf("invalid AI response: %w", err)
		}
		if len(parsed.Choices) == 0 {
			return fmt.Errorf("AI returned empty response")
		}
		return onChunk(strings.TrimSpace(parsed.Choices[0].Message.Content))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk openAIStreamResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return fmt.Errorf("%s", chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		text := chunk.Choices[0].Delta.Content
		if text == "" {
			continue
		}
		if err := onChunk(text); err != nil {
			return err
		}
	}
	return scanner.Err()
}
