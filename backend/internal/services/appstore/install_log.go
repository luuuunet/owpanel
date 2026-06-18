package appstore

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type InstallLogSnapshot struct {
	Key          string   `json:"key"`
	Status       string   `json:"status"`
	Lines        []string `json:"lines"`
	InstallError string   `json:"install_error,omitempty"`
	StartedAt    int64    `json:"started_at,omitempty"`
	UpdatedAt    int64    `json:"updated_at,omitempty"`
}

type installSession struct {
	key     string
	status  string
	lines   []string
	errMsg  string
	started time.Time
	updated time.Time
	mu      sync.RWMutex
}

type InstallLogManager struct {
	dataDir  string
	mu       sync.RWMutex
	sessions map[string]*installSession

	goMu   sync.RWMutex
	goKeys map[int64]string
}

func NewInstallLogManager(dataDir string) *InstallLogManager {
	return &InstallLogManager{
		dataDir:  dataDir,
		sessions: make(map[string]*installSession),
		goKeys:   make(map[int64]string),
	}
}

func (m *InstallLogManager) Begin(key, version, name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := &installSession{
		key:     key,
		status:  "installing",
		started: time.Now(),
		updated: time.Now(),
	}
	m.sessions[key] = s
	m.appendLineLocked(s, fmt.Sprintf("[%s] 开始安装 %s (%s)", time.Now().Format("15:04:05"), name, version))
}

func (m *InstallLogManager) Finish(key string, installErr error) {
	m.mu.Lock()
	s, ok := m.sessions[key]
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
	m.writeDisk(key, s)
}

func (m *InstallLogManager) AppendLine(key, line string) {
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	m.mu.RLock()
	s, ok := m.sessions[key]
	m.mu.RUnlock()
	if !ok {
		return
	}
	m.appendLineLocked(s, line)
}

func (m *InstallLogManager) appendLineLocked(s *installSession, line string) {
	s.mu.Lock()
	s.lines = append(s.lines, line)
	if len(s.lines) > 4000 {
		s.lines = s.lines[len(s.lines)-4000:]
	}
	s.updated = time.Now()
	s.mu.Unlock()
}

func (m *InstallLogManager) writeDisk(key string, s *installSession) {
	if m.dataDir == "" || s == nil {
		return
	}
	s.mu.RLock()
	content := strings.Join(s.lines, "\n")
	s.mu.RUnlock()
	dir := filepath.Join(m.dataDir, "logs", "install")
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, key+".log"), []byte(content), 0644)
}

func (m *InstallLogManager) Snapshot(key string) InstallLogSnapshot {
	m.mu.RLock()
	s, ok := m.sessions[key]
	m.mu.RUnlock()
	if ok {
		s.mu.RLock()
		snap := InstallLogSnapshot{
			Key:          key,
			Status:       s.status,
			Lines:        append([]string(nil), s.lines...),
			InstallError: s.errMsg,
			StartedAt:    s.started.Unix(),
			UpdatedAt:    s.updated.Unix(),
		}
		s.mu.RUnlock()
		return snap
	}
	return m.loadFromDisk(key)
}

func (m *InstallLogManager) ClearSession(key string) {
	m.mu.Lock()
	delete(m.sessions, key)
	m.mu.Unlock()
}

func (m *InstallLogManager) loadFromDisk(key string) InstallLogSnapshot {
	path := filepath.Join(m.dataDir, "logs", "install", key+".log")
	b, err := os.ReadFile(path)
	if err != nil {
		return InstallLogSnapshot{Key: key, Status: "idle", Lines: []string{}}
	}
	lines := strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	return InstallLogSnapshot{
		Key:    key,
		Status: "idle",
		Lines:  lines,
	}
}

func (m *InstallLogManager) setGoroutineKey(key string) func() {
	gid := currentGoID()
	m.goMu.Lock()
	m.goKeys[gid] = key
	m.goMu.Unlock()
	return func() {
		m.goMu.Lock()
		delete(m.goKeys, gid)
		m.goMu.Unlock()
	}
}

func (m *InstallLogManager) appendLineKey(key, line string) {
	if key == "" {
		return
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return
	}
	m.AppendLine(key, line)
}

var globalInstallLogs *InstallLogManager

func initInstallLogs(dataDir string) {
	globalInstallLogs = NewInstallLogManager(dataDir)
}

func installLogScope(key string) func() {
	if globalInstallLogs == nil {
		return func() {}
	}
	return globalInstallLogs.setGoroutineKey(key)
}

func installLogKeyForGoroutine() string {
	if globalInstallLogs == nil {
		return ""
	}
	gid := currentGoID()
	globalInstallLogs.goMu.RLock()
	key := globalInstallLogs.goKeys[gid]
	globalInstallLogs.goMu.RUnlock()
	return key
}

func currentGoID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(string(buf[:n]))
	if len(idField) >= 2 {
		var id int64
		fmt.Sscanf(idField[1], "%d", &id)
		return id
	}
	return 0
}

func logInstallLine(line string) {
	logInstallLineKey(installLogKeyForGoroutine(), line)
}

func logInstallLineKey(key, line string) {
	if globalInstallLogs != nil {
		globalInstallLogs.appendLineKey(key, line)
	}
}

func streamCommandOutput(r io.Reader, prefix, logKey string) {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if prefix != "" {
			line = prefix + line
		}
		logInstallLineKey(logKey, line)
	}
}
