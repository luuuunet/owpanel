package aichat

import (
	"encoding/base64"
	"fmt"
	"strings"
)

const maxChatImages = 4

type parsedDataURL struct {
	MediaType string
	Data      string
	RawURL    string
}

func normalizeChatImages(images []string) ([]string, error) {
	if len(images) == 0 {
		return nil, nil
	}
	if len(images) > maxChatImages {
		return nil, fmt.Errorf("最多上传 %d 张图片", maxChatImages)
	}
	out := make([]string, 0, len(images))
	for _, img := range images {
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}
		parsed, err := parseDataURL(img)
		if err != nil {
			return nil, err
		}
		if len(parsed.Data) > 4*1024*1024 {
			return nil, fmt.Errorf("单张图片不能超过 4MB")
		}
		out = append(out, parsed.RawURL)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func parseDataURL(raw string) (*parsedDataURL, error) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "data:") {
		return nil, fmt.Errorf("无效的图片数据")
	}
	comma := strings.Index(raw, ",")
	if comma < 0 {
		return nil, fmt.Errorf("无效的图片数据")
	}
	meta := raw[5:comma]
	data := raw[comma+1:]
	if !strings.Contains(meta, ";base64") {
		return nil, fmt.Errorf("仅支持 base64 图片")
	}
	mediaType := strings.TrimSuffix(meta, ";base64")
	switch mediaType {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
	default:
		return nil, fmt.Errorf("不支持的图片格式: %s", mediaType)
	}
	if _, err := base64.StdEncoding.DecodeString(data); err != nil {
		return nil, fmt.Errorf("图片 base64 解码失败")
	}
	return &parsedDataURL{MediaType: mediaType, Data: data, RawURL: raw}, nil
}

func messagesHaveImages(messages []Message) bool {
	for _, m := range messages {
		if len(m.Images) > 0 {
			return true
		}
	}
	return false
}

type openAIContentPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL *struct {
		URL string `json:"url"`
	} `json:"image_url,omitempty"`
}

type openAIMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

func toOpenAIMessages(messages []Message) []openAIMessage {
	out := make([]openAIMessage, 0, len(messages))
	for _, m := range messages {
		if len(m.Images) == 0 {
			out = append(out, openAIMessage{Role: m.Role, Content: m.Content})
			continue
		}
		parts := make([]openAIContentPart, 0, len(m.Images)+1)
		if strings.TrimSpace(m.Content) != "" {
			parts = append(parts, openAIContentPart{Type: "text", Text: m.Content})
		}
		for _, img := range m.Images {
			parts = append(parts, openAIContentPart{
				Type: "image_url",
				ImageURL: &struct {
					URL string `json:"url"`
				}{URL: img},
			})
		}
		out = append(out, openAIMessage{Role: m.Role, Content: parts})
	}
	return out
}

type anthropicContentBlock struct {
	Type   string `json:"type"`
	Text   string `json:"text,omitempty"`
	Source *struct {
		Type      string `json:"type"`
		MediaType string `json:"media_type"`
		Data      string `json:"data"`
	} `json:"source,omitempty"`
}

func toAnthropicMessages(messages []Message) (system string, chat []anthropicMsgMultimodal) {
	for _, m := range messages {
		switch m.Role {
		case "system":
			if system == "" {
				system = m.Content
			} else {
				system += "\n\n" + m.Content
			}
		case "user", "assistant":
			if len(m.Images) == 0 || m.Role == "assistant" {
				chat = append(chat, anthropicMsgMultimodal{Role: m.Role, Content: m.Content})
				continue
			}
			blocks := make([]anthropicContentBlock, 0, len(m.Images)+1)
			if strings.TrimSpace(m.Content) != "" {
				blocks = append(blocks, anthropicContentBlock{Type: "text", Text: m.Content})
			}
			for _, img := range m.Images {
				parsed, err := parseDataURL(img)
				if err != nil {
					continue
				}
				blocks = append(blocks, anthropicContentBlock{
					Type: "image",
					Source: &struct {
						Type      string `json:"type"`
						MediaType string `json:"media_type"`
						Data      string `json:"data"`
					}{Type: "base64", MediaType: parsed.MediaType, Data: parsed.Data},
				})
			}
			chat = append(chat, anthropicMsgMultimodal{Role: m.Role, Content: blocks})
		}
	}
	return system, chat
}

type anthropicMsgMultimodal struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type anthropicChatRequestMultimodal struct {
	Model     string                   `json:"model"`
	MaxTokens int                      `json:"max_tokens"`
	System    string                   `json:"system,omitempty"`
	Messages  []anthropicMsgMultimodal `json:"messages"`
}

func appendVisionFallbackNote(messages []Message) []Message {
	out := make([]Message, len(messages))
	copy(out, messages)
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Role != "user" || len(out[i].Images) == 0 {
			continue
		}
		n := len(out[i].Images)
		note := fmt.Sprintf("\n\n[用户上传了 %d 张参考图片；当前 AI 接口不支持图像理解，请根据文字描述作答，或建议切换到支持视觉的 OpenAI / Claude 模型]", n)
		out[i].Content += note
		out[i].Images = nil
		break
	}
	return out
}
