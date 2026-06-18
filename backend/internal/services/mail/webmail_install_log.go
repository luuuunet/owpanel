package mail

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const webmailInstallLogKey = "snappymail"

type WebmailInstallLogSnapshot struct {
	Key          string   `json:"key"`
	Status       string   `json:"status"`
	Lines        []string `json:"lines"`
	InstallError string   `json:"install_error,omitempty"`
	StartedAt    int64    `json:"started_at,omitempty"`
	UpdatedAt    int64    `json:"updated_at,omitempty"`
}

type webmailInstallSession struct {
	status  string
	lines   []string
	errMsg  string
	started time.Time
	updated time.Time
	mu      sync.RWMutex
}

type webmailInstallLogManager struct {
	dataDir  string
	mu       sync.RWMutex
	sessions map[string]*webmailInstallSession
}

var webmailInstallLogs *webmailInstallLogManager

func initWebmailInstallLogs(dataDir string) {
	webmailInstallLogs = &webmailInstallLogManager{
		dataDir:  dataDir,
		sessions: make(map[string]*webmailInstallSession),
	}
}

func (m *webmailInstallLogManager) Begin(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &webmailInstallSession{
		status:  "installing",
		started: time.Now(),
		updated: time.Now(),
	}
	m.sessions[webmailInstallLogKey] = s
	m.appendLineLocked(s, fmt.Sprintf("[%s] 开始安装 %s", time.Now().Format("15:04:05"), name))
}

func (m *webmailInstallLogManager) Finish(installErr error) {
	m.mu.Lock()
	s, ok := m.sessions[webmailInstallLogKey]
	if !ok {
		m.mu.Unlock()
		return
	}
	s.mu.Lock()
	if installErr != nil {
		s.status = "failed"
		s.errMsg = installErr.Error()
		s.lines = append(s.lines, fmt.Sprintf("[%s] 安装失败: %s", time.Now().Format("15:04:05"), installErr.Error()))
	} else {
		s.status = "success"
		s.lines = append(s.lines, fmt.Sprintf("[%s] 安装成功", time.Now().Format("15:04:05")))
	}
	s.updated = time.Now()
	s.mu.Unlock()
	m.mu.Unlock()
	m.writeDisk(s)
}

func (m *webmailInstallLogManager) AppendLine(line string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	m.mu.RLock()
	s, ok := m.sessions[webmailInstallLogKey]
	m.mu.RUnlock()
	if !ok {
		return
	}
	m.appendLineLocked(s, line)
}

func (m *webmailInstallLogManager) appendLineLocked(s *webmailInstallSession, line string) {
	s.mu.Lock()
	s.lines = append(s.lines, line)
	if len(s.lines) > 4000 {
		s.lines = s.lines[len(s.lines)-4000:]
	}
	s.updated = time.Now()
	s.mu.Unlock()
}

func (m *webmailInstallLogManager) writeDisk(s *webmailInstallSession) {
	if m.dataDir == "" || s == nil {
		return
	}
	s.mu.RLock()
	content := strings.Join(s.lines, "\n")
	s.mu.RUnlock()
	dir := filepath.Join(m.dataDir, "logs", "install")
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, webmailInstallLogKey+".log"), []byte(content), 0644)
}

func (m *webmailInstallLogManager) Snapshot() WebmailInstallLogSnapshot {
	m.mu.RLock()
	s, ok := m.sessions[webmailInstallLogKey]
	m.mu.RUnlock()
	if ok {
		s.mu.RLock()
		snap := WebmailInstallLogSnapshot{
			Key:          webmailInstallLogKey,
			Status:       s.status,
			Lines:        append([]string(nil), s.lines...),
			InstallError: s.errMsg,
			StartedAt:    s.started.Unix(),
			UpdatedAt:    s.updated.Unix(),
		}
		s.mu.RUnlock()
		return snap
	}
	return m.loadFromDisk()
}

func (m *webmailInstallLogManager) loadFromDisk() WebmailInstallLogSnapshot {
	path := filepath.Join(m.dataDir, "logs", "install", webmailInstallLogKey+".log")
	b, err := os.ReadFile(path)
	if err != nil {
		return WebmailInstallLogSnapshot{Key: webmailInstallLogKey, Status: "idle", Lines: []string{}}
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	status := "idle"
	if len(lines) > 0 {
		last := lines[len(lines)-1]
		if strings.Contains(last, "安装成功") {
			status = "success"
		} else if strings.Contains(last, "安装失败") {
			status = "failed"
		}
	}
	return WebmailInstallLogSnapshot{
		Key:    webmailInstallLogKey,
		Status: status,
		Lines:  lines,
	}
}

func (m *webmailInstallLogManager) IsInstalling() bool {
	snap := m.Snapshot()
	return snap.Status == "installing"
}

func (s *Service) webmailLog(line string) {
	if webmailInstallLogs != nil {
		webmailInstallLogs.AppendLine(line)
	}
}

func (s *Service) GetWebmailInstallLogs() WebmailInstallLogSnapshot {
	if webmailInstallLogs == nil {
		return WebmailInstallLogSnapshot{Key: webmailInstallLogKey, Status: "idle", Lines: []string{}}
	}
	return webmailInstallLogs.Snapshot()
}
