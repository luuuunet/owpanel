package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	base := os.Getenv("PANEL_BASE")
	if base == "" {
		base = "http://127.0.0.1:8888/bb276bbd"
	}

	client := &http.Client{Timeout: 120 * time.Second}

	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "admin"})
	req, _ := http.NewRequest(http.MethodPost, base+"/api/v1/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("login error:", err)
		os.Exit(1)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("login status=%d body=%s\n", resp.StatusCode, trunc(string(body), 200))
	if resp.StatusCode != 200 {
		os.Exit(1)
	}

	var loginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(body, &loginResp)
	token := loginResp.Data.Token
	if token == "" {
		fmt.Println("no token")
		os.Exit(1)
	}

	for _, path := range []string{"/api/v1/phpmyadmin/access", "/api/v1/software/phpmyadmin"} {
		req, _ := http.NewRequest(http.MethodGet, base+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err = client.Do(req)
		if err != nil {
			fmt.Println(path, "error:", err)
			continue
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%s status=%d body=%s\n", path, resp.StatusCode, trunc(string(b), 400))
	}

	req, _ = http.NewRequest(http.MethodPost, base+"/api/v1/phpmyadmin/setup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = client.Do(req)
	if err != nil {
		fmt.Println("setup error:", err)
		os.Exit(1)
	}
	b, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("setup status=%d body=%s\n", resp.StatusCode, trunc(string(b), 500))
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
