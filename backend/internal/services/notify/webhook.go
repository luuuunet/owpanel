package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// PostJSON sends a JSON payload to a webhook URL. Empty URL is a no-op.
func PostJSON(url string, payload map[string]interface{}) {
	url = strings.TrimSpace(url)
	if url == "" {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
