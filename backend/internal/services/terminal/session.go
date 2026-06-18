package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type Config struct {
	Host          string
	Port          int
	User          string
	Password      string
	PrivateKey    string
	KeyPassphrase string
	AuthMethod    string // password | key
	Cols          int
	Rows          int
}

type SessionContext struct {
	UserID    uint
	Username  string
	Role      string
	AssetID   uint
	AccountID uint
}

type SessionHooks struct {
	OnConnected func(cfg Config, ctx SessionContext) (recorder SessionRecorder, err error)
}

type SessionRecorder interface {
	WriteOutput(data []byte)
	ProcessInput(data []byte) ([]byte, error)
	SetKill(fn func())
	Close(status string)
}

type ConnectMessage struct {
	Type          string `json:"type"`
	Host          string `json:"host,omitempty"`
	Port          int    `json:"port,omitempty"`
	User          string `json:"user,omitempty"`
	Password      string `json:"password,omitempty"`
	PrivateKey    string `json:"private_key,omitempty"`
	KeyPassphrase string `json:"key_passphrase,omitempty"`
	KeyID         uint   `json:"key_id,omitempty"`
	AuthMethod    string `json:"auth_method,omitempty"`
	NodeID        uint   `json:"node_id,omitempty"`
	AssetID       uint   `json:"asset_id,omitempty"`
	AccountID     uint   `json:"account_id,omitempty"`
	Cols          int    `json:"cols,omitempty"`
	Rows          int    `json:"rows,omitempty"`
}

type ctrlMsg = ConnectMessage

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     checkWebSocketOrigin,
}

func checkWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	host := r.Host
	if host == "" {
		return false
	}
	originHost := origin
	if strings.HasPrefix(originHost, "http://") || strings.HasPrefix(originHost, "https://") {
		if u, err := url.Parse(origin); err == nil && u.Host != "" {
			originHost = u.Host
		}
	}
	originHost = strings.TrimSuffix(originHost, "/")
	host = strings.TrimSuffix(host, "/")
	if strings.EqualFold(originHost, host) {
		return true
	}
	if strings.HasPrefix(originHost, "127.0.0.1:") || strings.HasPrefix(originHost, "localhost:") {
		return strings.HasPrefix(host, "127.0.0.1:") || strings.HasPrefix(host, "localhost:")
	}
	return false
}

func Dial(cfg Config) (*ssh.Client, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 22
	}
	user := strings.TrimSpace(cfg.User)
	if user == "" {
		user = "root"
	}

	method := strings.TrimSpace(cfg.AuthMethod)
	if method == "" {
		if strings.TrimSpace(cfg.PrivateKey) != "" {
			method = "key"
		} else {
			method = "password"
		}
	}

	var auths []ssh.AuthMethod
	if method == "key" || strings.TrimSpace(cfg.PrivateKey) != "" {
		signer, err := parsePrivateKey(cfg.PrivateKey, cfg.KeyPassphrase)
		if err != nil {
			return nil, err
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	if method == "password" || strings.TrimSpace(cfg.Password) != "" {
		auths = append(auths, ssh.Password(cfg.Password))
	}
	if len(auths) == 0 {
		return nil, fmt.Errorf("请提供 SSH 密码或私钥")
	}

	clientCfg := &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	return ssh.Dial("tcp", net.JoinHostPort(host, fmt.Sprintf("%d", port)), clientCfg)
}

func parsePrivateKey(pem, passphrase string) (ssh.Signer, error) {
	raw := []byte(strings.TrimSpace(pem))
	if len(raw) == 0 {
		return nil, fmt.Errorf("私钥不能为空")
	}
	if strings.TrimSpace(passphrase) != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(raw, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("私钥解析失败: %w", err)
		}
		return signer, nil
	}
	signer, err := ssh.ParsePrivateKey(raw)
	if err != nil {
		return nil, fmt.Errorf("私钥解析失败（若私钥有密码请填写 passphrase）: %w", err)
	}
	return signer, nil
}

type Resolver struct {
	ResolveNode  func(nodeID uint) (Config, error)
	ResolveKey   func(keyID uint) (privateKey string, err error)
	ResolveAsset func(assetID uint, ctx SessionContext) (Config, error)
}

type HandlerOptions struct {
	Resolver      Resolver
	Context       SessionContext
	Hooks         *SessionHooks
	BeforeConnect func(msg ConnectMessage, ctx SessionContext) error
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request, opts HandlerOptions) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	ws.SetReadLimit(1 << 20)
	_ = ws.SetReadDeadline(time.Now().Add(60 * time.Second))

	var msg ctrlMsg
	if err := ws.ReadJSON(&msg); err != nil || msg.Type != "connect" {
		_ = ws.WriteJSON(map[string]string{"type": "error", "message": "首条消息须为 connect"})
		return
	}

	cfg := Config{
		Host: msg.Host, Port: msg.Port, User: msg.User,
		Password: msg.Password, PrivateKey: msg.PrivateKey,
		KeyPassphrase: msg.KeyPassphrase, AuthMethod: msg.AuthMethod,
		Cols: msg.Cols, Rows: msg.Rows,
	}
	if cfg.Cols <= 0 {
		cfg.Cols = 120
	}
	if cfg.Rows <= 0 {
		cfg.Rows = 32
	}

	ctx := opts.Context
	if msg.AssetID > 0 {
		ctx.AssetID = msg.AssetID
	}
	if msg.AccountID > 0 {
		ctx.AccountID = msg.AccountID
	}

	if opts.BeforeConnect != nil {
		if err := opts.BeforeConnect(msg, ctx); err != nil {
			_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
			return
		}
	}

	if msg.KeyID > 0 && opts.Resolver.ResolveKey != nil {
		pk, err := opts.Resolver.ResolveKey(msg.KeyID)
		if err != nil {
			_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
			return
		}
		cfg.PrivateKey = pk
		cfg.AuthMethod = "key"
	}

	if msg.AssetID > 0 && opts.Resolver.ResolveAsset != nil {
		acfg, err := opts.Resolver.ResolveAsset(msg.AssetID, ctx)
		if err != nil {
			_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
			return
		}
		cfg = acfg
		if cfg.Cols <= 0 {
			cfg.Cols = msg.Cols
		}
		if cfg.Rows <= 0 {
			cfg.Rows = msg.Rows
		}
	}

	if msg.NodeID > 0 && opts.Resolver.ResolveNode != nil {
		ncfg, err := opts.Resolver.ResolveNode(msg.NodeID)
		if err != nil {
			_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
			return
		}
		if msg.Password != "" {
			ncfg.Password = msg.Password
		}
		if msg.PrivateKey != "" {
			ncfg.PrivateKey = msg.PrivateKey
		}
		if msg.KeyPassphrase != "" {
			ncfg.KeyPassphrase = msg.KeyPassphrase
		}
		if msg.AuthMethod != "" {
			ncfg.AuthMethod = msg.AuthMethod
		}
		if msg.User != "" {
			ncfg.User = msg.User
		}
		if msg.Host != "" {
			ncfg.Host = msg.Host
		}
		if msg.Port > 0 {
			ncfg.Port = msg.Port
		}
		if ncfg.Cols <= 0 {
			ncfg.Cols = cfg.Cols
		}
		if ncfg.Rows <= 0 {
			ncfg.Rows = cfg.Rows
		}
		cfg = ncfg
	}

	client, err := Dial(cfg)
	if err != nil {
		_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
		return
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", cfg.Rows, cfg.Cols, modes); err != nil {
		_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
		return
	}
	stdin, _ := session.StdinPipe()
	stdout, _ := session.StdoutPipe()
	stderr, _ := session.StderrPipe()
	if err := session.Shell(); err != nil {
		_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
		return
	}

	var recorder SessionRecorder
	if opts.Hooks != nil && opts.Hooks.OnConnected != nil {
		rec, err := opts.Hooks.OnConnected(cfg, ctx)
		if err != nil {
			_ = ws.WriteJSON(map[string]string{"type": "error", "message": err.Error()})
			return
		}
		recorder = rec
		if recorder != nil {
			defer recorder.Close("closed")
			recorder.SetKill(func() { ws.Close() })
		}
	}

	_ = ws.WriteJSON(map[string]string{"type": "connected"})
	_ = ws.SetReadDeadline(time.Time{})

	var writeMu sync.Mutex
	writeText := func(data []byte) {
		if recorder != nil {
			recorder.WriteOutput(data)
		}
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = ws.WriteMessage(websocket.BinaryMessage, data)
	}

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				writeText(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				writeText(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		mt, data, err := ws.ReadMessage()
		if err != nil {
			break
		}
		if mt == websocket.BinaryMessage || mt == websocket.TextMessage {
			if len(data) == 0 {
				continue
			}
			if mt == websocket.TextMessage && data[0] == '{' {
				var cm ctrlMsg
				if json.Unmarshal(data, &cm) == nil && cm.Type == "resize" && cm.Cols > 0 && cm.Rows > 0 {
					_ = session.WindowChange(cm.Rows, cm.Cols)
				}
				continue
			}
			if recorder != nil {
				filtered, perr := recorder.ProcessInput(data)
				if perr != nil {
					writeMu.Lock()
					_ = ws.WriteJSON(map[string]string{"type": "policy", "message": perr.Error()})
					writeMu.Unlock()
					continue
				}
				data = filtered
				if len(data) == 0 {
					continue
				}
			}
			_, _ = stdin.Write(data)
		}
	}

	_ = session.Close()
	_, _ = io.Copy(io.Discard, stdout)
}
