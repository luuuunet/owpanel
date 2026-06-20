package aichat

import "fmt"

type AssistantStatus struct {
	Enabled     bool   `json:"enabled"`
	Configured  bool   `json:"configured"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Message     string `json:"message,omitempty"`
}

func (s *Service) AssistantStatus() AssistantStatus {
	cfg, err := s.loadConfig()
	if err != nil {
		return AssistantStatus{Message: err.Error()}
	}
	st := AssistantStatus{
		Enabled:    cfg.Enabled,
		Provider: cfg.Provider,
		Model:    cfg.Model,
	}
	if !cfg.Enabled {
		st.Message = "请先在面板设置中启用 AI 助手"
		return st
	}
	if cfg.APIKey == "" && cfg.Provider != "ollama" && cfg.Provider != "huggingface" {
		st.Message = "请先在面板设置中绑定 AI API Key"
		return st
	}
	if cfg.Model == "" {
		st.Message = "请先在面板设置中选择 AI 模型"
		return st
	}
	st.Configured = true
	return st
}

func (s *Service) EnsureConfigured() error {
	st := s.AssistantStatus()
	if st.Configured {
		return nil
	}
	if st.Message != "" {
		return fmt.Errorf("%s", st.Message)
	}
	return fmt.Errorf("AI 助手未配置，请前往面板设置绑定 API")
}
