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

type cursorCreateResponse struct {
	Agent struct {
		ID string `json:"id"`
	} `json:"agent"`
	Run struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"run"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type cursorRunResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result string `json:"result"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (s *Service) callCursorAgent(cfg aiConfig, messages []Message) (string, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.cursor.com"
	}

	payload := map[string]interface{}{
		"prompt": map[string]string{"text": buildCursorPrompt(messages)},
		"mode":   "agent",
	}
	if cfg.Model != "" {
		payload["model"] = map[string]string{"id": cfg.Model}
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, base+"/v1/agents", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cursorAuthHeader(cfg.APIKey))

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Cursor API request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		msg := parseCursorError(raw)
		return "", fmt.Errorf("Cursor API error (%d): %s", resp.StatusCode, msg)
	}

	var created cursorCreateResponse
	if err := json.Unmarshal(raw, &created); err != nil {
		return "", fmt.Errorf("invalid Cursor create response: %w", err)
	}
	agentID := created.Agent.ID
	runID := created.Run.ID
	if agentID == "" || runID == "" {
		return "", fmt.Errorf("Cursor API returned empty agent or run id")
	}

	deadline := time.Now().Add(5 * time.Minute)
	for time.Now().Before(deadline) {
		run, err := s.fetchCursorRun(base, cfg.APIKey, agentID, runID)
		if err != nil {
			return "", err
		}
		switch strings.ToUpper(run.Status) {
		case "FINISHED":
			if strings.TrimSpace(run.Result) == "" {
				return "", fmt.Errorf("Cursor agent finished without a reply")
			}
			return strings.TrimSpace(run.Result), nil
		case "ERROR", "CANCELLED", "EXPIRED":
			msg := strings.TrimSpace(run.Result)
			if msg == "" {
				msg = run.Status
			}
			return "", fmt.Errorf("Cursor agent run failed: %s", msg)
		}
		time.Sleep(1200 * time.Millisecond)
	}
	return "", fmt.Errorf("Cursor agent timed out after 5 minutes")
}

func (s *Service) fetchCursorRun(base, apiKey, agentID, runID string) (*cursorRunResponse, error) {
	req, err := http.NewRequest(http.MethodGet, base+"/v1/agents/"+agentID+"/runs/"+runID, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", cursorAuthHeader(apiKey))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cursor run poll failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Cursor run poll error (%d): %s", resp.StatusCode, parseCursorError(raw))
	}

	var run cursorRunResponse
	if err := json.Unmarshal(raw, &run); err != nil {
		return nil, fmt.Errorf("invalid Cursor run response: %w", err)
	}
	return &run, nil
}

func (s *Service) TestCursorChat(apiKey, baseURL, model string) (string, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", fmt.Errorf("API Key 为空")
	}
	if model == "" {
		model = "composer-2.5"
	}
	return s.callCursorAgent(aiConfig{
		Provider: "cursor",
		APIKey:   apiKey,
		BaseURL:  baseURL,
		Model:    model,
	}, []Message{
		{Role: "user", Content: "Reply with exactly: OK"},
	})
}

func cursorAuthHeader(apiKey string) string {
	return "Bearer " + strings.TrimSpace(apiKey)
}

func buildCursorPrompt(messages []Message) string {
	var b strings.Builder
	for _, m := range messages {
		switch m.Role {
		case "system":
			b.WriteString("System:\n")
		case "user":
			b.WriteString("User:\n")
		case "assistant":
			b.WriteString("Assistant:\n")
		default:
			continue
		}
		b.WriteString(m.Content)
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

func parseCursorError(raw []byte) string {
	var nested struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	var wrapped struct {
		Message string          `json:"message"`
		Error   json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		if len(wrapped.Error) > 0 {
			if json.Unmarshal(wrapped.Error, &nested) == nil && nested.Message != "" {
				return cursorErrorMessage(nested.Code, nested.Message)
			}
			if msg := strings.TrimSpace(string(wrapped.Error)); msg != "" && msg != "null" {
				return msg
			}
		}
		if wrapped.Message != "" {
			return wrapped.Message
		}
	}
	var flat struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	_ = json.Unmarshal(raw, &flat)
	if flat.Message != "" {
		return flat.Message
	}
	if flat.Error != "" {
		return flat.Error
	}
	return strings.TrimSpace(string(raw))
}

func cursorErrorMessage(code, message string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "plan_required":
		return "Cursor Cloud Agent 需要 Pro 订阅（当前为免费账号）。请升级 cursor.com 订阅，或改用 OpenAI / DeepSeek 等 OpenAI 兼容 API"
	case "invalid_api_key", "authentication_failed":
		return "Cursor API Key 无效，请在 cursor.com/dashboard → Cloud Agents → API Keys 重新创建"
	default:
		if message != "" {
			return message
		}
		return code
	}
}

type cursorModelsResponse struct {
	Items []struct {
		ID          string   `json:"id"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Aliases     []string `json:"aliases"`
	} `json:"items"`
}

func fetchCursorModels(cfg aiConfig) ([]ModelOption, error) {
	base := strings.TrimRight(cfg.BaseURL, "/")
	if base == "" {
		base = "https://api.cursor.com"
	}

	req, err := http.NewRequest(http.MethodGet, base+"/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", cursorAuthHeader(cfg.APIKey))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Cursor models request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Cursor models error (%d): %s", resp.StatusCode, parseCursorError(raw))
	}

	var parsed cursorModelsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("invalid Cursor models response: %w", err)
	}
	if len(parsed.Items) == 0 {
		return nil, fmt.Errorf("Cursor API returned no models")
	}

	seen := make(map[string]struct{}, len(parsed.Items))
	out := make([]ModelOption, 0, len(parsed.Items))
	for _, item := range parsed.Items {
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
		out = append(out, ModelOption{
			ID:          id,
			DisplayName: name,
			Description: strings.TrimSpace(item.Description),
		})
	}
	sortModels(out)
	return out, nil
}
