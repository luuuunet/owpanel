package docker

import (
	"encoding/json"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var execUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin == "" {
			return true
		}
		host := r.Host
		if host == "" {
			return true
		}
		return strings.Contains(origin, host) || strings.HasPrefix(origin, "http://localhost") || strings.HasPrefix(origin, "https://localhost")
	},
}

type execCtrlMsg struct {
	Type    string   `json:"type"`
	Command []string `json:"command,omitempty"`
	Shell   string   `json:"shell,omitempty"`
	Data    string   `json:"data,omitempty"`
}

// HandleExecWebSocket provides an interactive docker exec console over WebSocket.
// Protocol (JSON text frames for control, binary/text for data):
//   client -> {type:"start", shell:"/bin/sh"} or {type:"start", command:["bash"]}
//   client -> {type:"stdin", data:"..."}
//   client -> {type:"ping"}
//   server -> {type:"ready"} / {type:"exit", data:"..."} / {type:"error", data:"..."}
//   server -> binary/text stdout chunks
func HandleExecWebSocket(w http.ResponseWriter, r *http.Request, containerID string) {
	containerID = strings.TrimSpace(containerID)
	if containerID == "" {
		http.Error(w, "container id required", http.StatusBadRequest)
		return
	}

	conn, err := execUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	var (
		cmd    *exec.Cmd
		stdin  io.WriteCloser
		stdout io.ReadCloser
		wg     sync.WaitGroup
		writeMu sync.Mutex
		started bool
	)

	writeJSON := func(v any) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_ = conn.WriteJSON(v)
	}
	writeBytes := func(b []byte) {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_ = conn.WriteMessage(websocket.BinaryMessage, b)
	}

	cleanup := func() {
		if stdin != nil {
			_ = stdin.Close()
		}
		if stdout != nil {
			_ = stdout.Close()
		}
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}
	defer cleanup()

	for {
		mt, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		if mt == websocket.BinaryMessage {
			if started && stdin != nil && len(payload) > 0 {
				_, _ = stdin.Write(payload)
			}
			continue
		}

		var msg execCtrlMsg
		if err := json.Unmarshal(payload, &msg); err != nil {
			// treat as raw stdin text
			if started && stdin != nil {
				_, _ = stdin.Write(payload)
			}
			continue
		}

		switch strings.ToLower(msg.Type) {
		case "ping":
			writeJSON(map[string]string{"type": "pong"})
		case "stdin":
			if started && stdin != nil && msg.Data != "" {
				_, _ = stdin.Write([]byte(msg.Data))
			}
		case "start":
			if started {
				writeJSON(map[string]string{"type": "error", "data": "already started"})
				continue
			}
			args := []string{"exec", "-i", containerID}
			if len(msg.Command) > 0 {
				args = append(args, msg.Command...)
			} else {
				shell := strings.TrimSpace(msg.Shell)
				if shell == "" {
					shell = "/bin/sh"
				}
				args = append(args, shell)
			}
			cmd = exec.Command("docker", args...)
			stdin, err = cmd.StdinPipe()
			if err != nil {
				writeJSON(map[string]string{"type": "error", "data": err.Error()})
				continue
			}
			stdout, err = cmd.StdoutPipe()
			if err != nil {
				writeJSON(map[string]string{"type": "error", "data": err.Error()})
				continue
			}
			cmd.Stderr = cmd.Stdout
			if err := cmd.Start(); err != nil {
				writeJSON(map[string]string{"type": "error", "data": err.Error()})
				continue
			}
			started = true
			writeJSON(map[string]string{"type": "ready"})

			wg.Add(1)
			go func() {
				defer wg.Done()
				buf := make([]byte, 4096)
				for {
					n, readErr := stdout.Read(buf)
					if n > 0 {
						writeBytes(append([]byte{}, buf[:n]...))
					}
					if readErr != nil {
						break
					}
				}
				waitErr := cmd.Wait()
				msg := "exit"
				if waitErr != nil {
					msg = waitErr.Error()
				}
				writeJSON(map[string]string{"type": "exit", "data": msg})
			}()
		case "resize":
			// CLI exec without TTY: ignore resize
		default:
			// ignore unknown
		}
	}
	wg.Wait()
}
