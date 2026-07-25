package filemgr

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultChunkSize int64 = 2 * 1024 * 1024 // 2 MiB
	maxChunkSize     int64 = 16 * 1024 * 1024
	minChunkSize     int64 = 256 * 1024
)

type ChunkSessionMeta struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	DestDir     string `json:"dest_dir"`
	Size        int64  `json:"size"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
	Fingerprint string `json:"fingerprint"`
}

type ChunkSessionStatus struct {
	ID             string `json:"id"`
	Filename       string `json:"filename"`
	DestDir        string `json:"dest_dir"`
	Size           int64  `json:"size"`
	ChunkSize      int64  `json:"chunk_size"`
	TotalChunks    int    `json:"total_chunks"`
	ReceivedChunks []int  `json:"received_chunks"`
	ReceivedCount  int    `json:"received_count"`
	Complete       bool   `json:"complete"`
	Progress       int    `json:"progress"` // 0-100
}

var chunkMu sync.Mutex

func (s *Service) uploadTempRoot() string {
	root := s.dataDir
	if root == "" {
		root = os.TempDir()
	}
	return filepath.Join(root, "tmp", "uploads")
}

func (s *Service) sessionDir(id string) string {
	return filepath.Join(s.uploadTempRoot(), filepath.Base(id))
}

func normalizeChunkSize(size int64) int64 {
	if size <= 0 {
		return defaultChunkSize
	}
	if size < minChunkSize {
		return minChunkSize
	}
	if size > maxChunkSize {
		return maxChunkSize
	}
	return size
}

func NewUploadID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func UploadFingerprint(destDir, filename string, size int64) string {
	h := sha1.Sum([]byte(fmt.Sprintf("%s\n%s\n%d", filepath.Clean(destDir), filepath.Base(filename), size)))
	return hex.EncodeToString(h[:])
}

func (s *Service) InitChunkUpload(destDir, filename string, size, chunkSize int64, resumeID string) (*ChunkSessionStatus, error) {
	chunkMu.Lock()
	defer chunkMu.Unlock()

	dir, err := s.resolvePath(destDir)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("upload target is not a directory")
	}
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || strings.Contains(filename, "..") {
		return nil, fmt.Errorf("invalid filename")
	}
	if size < 0 {
		return nil, fmt.Errorf("invalid size")
	}

	chunkSize = normalizeChunkSize(chunkSize)
	total := 1
	if size > 0 {
		total = int((size + chunkSize - 1) / chunkSize)
	}
	fp := UploadFingerprint(dir, filename, size)

	// Resume existing session by id or fingerprint.
	if resumeID != "" {
		if st, err := s.readSessionStatusLocked(resumeID); err == nil && st.Filename == filename && st.Size == size {
			return st, nil
		}
	}
	if existing, err := s.findSessionByFingerprintLocked(fp); err == nil && existing != nil {
		return existing, nil
	}

	id := NewUploadID()
	meta := ChunkSessionMeta{
		ID:          id,
		Filename:    filename,
		DestDir:     dir,
		Size:        size,
		ChunkSize:   chunkSize,
		TotalChunks: total,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
		Fingerprint: fp,
	}
	sessDir := s.sessionDir(id)
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		return nil, err
	}
	if err := writeSessionMeta(sessDir, &meta); err != nil {
		_ = os.RemoveAll(sessDir)
		return nil, err
	}
	return sessionStatusFromMeta(&meta, nil), nil
}

func (s *Service) GetChunkUploadStatus(id string) (*ChunkSessionStatus, error) {
	chunkMu.Lock()
	defer chunkMu.Unlock()
	return s.readSessionStatusLocked(id)
}

func (s *Service) PutChunk(id string, index int, r io.Reader, declaredSize int64) (*ChunkSessionStatus, error) {
	chunkMu.Lock()
	defer chunkMu.Unlock()

	meta, err := s.readSessionMetaLocked(id)
	if err != nil {
		return nil, err
	}
	if index < 0 || index >= meta.TotalChunks {
		return nil, fmt.Errorf("chunk index out of range")
	}

	sessDir := s.sessionDir(id)
	chunkPath := filepath.Join(sessDir, fmt.Sprintf("chunk_%06d", index))
	tmpPath := chunkPath + ".part"

	out, err := os.Create(tmpPath)
	if err != nil {
		return nil, err
	}
	n, copyErr := io.Copy(out, r)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return nil, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return nil, closeErr
	}
	if declaredSize > 0 && n != declaredSize {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("chunk size mismatch: got %d want %d", n, declaredSize)
	}

	// Validate expected size for non-last chunks.
	expected := meta.ChunkSize
	if index == meta.TotalChunks-1 && meta.Size > 0 {
		expected = meta.Size - int64(index)*meta.ChunkSize
	}
	if meta.Size > 0 && n != expected {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("unexpected chunk length: got %d want %d", n, expected)
	}

	if err := os.Rename(tmpPath, chunkPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}

	meta.UpdatedAt = time.Now().Unix()
	_ = writeSessionMeta(sessDir, meta)
	received := listReceivedChunks(sessDir, meta.TotalChunks)
	return sessionStatusFromMeta(meta, received), nil
}

func (s *Service) CompleteChunkUpload(id string) (string, error) {
	chunkMu.Lock()
	defer chunkMu.Unlock()

	meta, err := s.readSessionMetaLocked(id)
	if err != nil {
		return "", err
	}
	sessDir := s.sessionDir(id)
	received := listReceivedChunks(sessDir, meta.TotalChunks)
	if len(received) != meta.TotalChunks {
		return "", fmt.Errorf("upload incomplete: %d/%d chunks", len(received), meta.TotalChunks)
	}

	target := filepath.Join(meta.DestDir, meta.Filename)
	tmpTarget := target + ".uploading"
	out, err := os.Create(tmpTarget)
	if err != nil {
		return "", err
	}

	var written int64
	for i := 0; i < meta.TotalChunks; i++ {
		chunkPath := filepath.Join(sessDir, fmt.Sprintf("chunk_%06d", i))
		in, err := os.Open(chunkPath)
		if err != nil {
			out.Close()
			_ = os.Remove(tmpTarget)
			return "", fmt.Errorf("missing chunk %d: %w", i, err)
		}
		n, err := io.Copy(out, in)
		in.Close()
		if err != nil {
			out.Close()
			_ = os.Remove(tmpTarget)
			return "", err
		}
		written += n
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmpTarget)
		return "", err
	}
	if meta.Size > 0 && written != meta.Size {
		_ = os.Remove(tmpTarget)
		return "", fmt.Errorf("assembled size mismatch: got %d want %d", written, meta.Size)
	}
	if err := os.Rename(tmpTarget, target); err != nil {
		_ = os.Remove(tmpTarget)
		return "", err
	}
	_ = os.RemoveAll(sessDir)
	return target, nil
}

func (s *Service) CancelChunkUpload(id string) error {
	chunkMu.Lock()
	defer chunkMu.Unlock()
	id = filepath.Base(id)
	if id == "" || id == "." {
		return fmt.Errorf("invalid upload id")
	}
	return os.RemoveAll(s.sessionDir(id))
}

func (s *Service) readSessionMetaLocked(id string) (*ChunkSessionMeta, error) {
	id = filepath.Base(id)
	path := filepath.Join(s.sessionDir(id), "meta.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("upload session not found")
	}
	var meta ChunkSessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *Service) readSessionStatusLocked(id string) (*ChunkSessionStatus, error) {
	meta, err := s.readSessionMetaLocked(id)
	if err != nil {
		return nil, err
	}
	received := listReceivedChunks(s.sessionDir(id), meta.TotalChunks)
	return sessionStatusFromMeta(meta, received), nil
}

func (s *Service) findSessionByFingerprintLocked(fp string) (*ChunkSessionStatus, error) {
	root := s.uploadTempRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		meta, err := s.readSessionMetaLocked(e.Name())
		if err != nil {
			continue
		}
		if meta.Fingerprint == fp {
			received := listReceivedChunks(s.sessionDir(meta.ID), meta.TotalChunks)
			return sessionStatusFromMeta(meta, received), nil
		}
	}
	return nil, nil
}

func writeSessionMeta(sessDir string, meta *ChunkSessionMeta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sessDir, "meta.json"), data, 0o644)
}

func listReceivedChunks(sessDir string, total int) []int {
	out := make([]int, 0, total)
	for i := 0; i < total; i++ {
		path := filepath.Join(sessDir, fmt.Sprintf("chunk_%06d", i))
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			out = append(out, i)
		}
	}
	sort.Ints(out)
	return out
}

func sessionStatusFromMeta(meta *ChunkSessionMeta, received []int) *ChunkSessionStatus {
	if received == nil {
		received = []int{}
	}
	progress := 0
	if meta.TotalChunks > 0 {
		progress = len(received) * 100 / meta.TotalChunks
	}
	return &ChunkSessionStatus{
		ID:             meta.ID,
		Filename:       meta.Filename,
		DestDir:        meta.DestDir,
		Size:           meta.Size,
		ChunkSize:      meta.ChunkSize,
		TotalChunks:    meta.TotalChunks,
		ReceivedChunks: received,
		ReceivedCount:  len(received),
		Complete:       len(received) == meta.TotalChunks && meta.TotalChunks > 0,
		Progress:       progress,
	}
}

// ParseChunkIndex reads chunk index from form/query.
func ParseChunkIndex(raw string) (int, error) {
	i, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid chunk index")
	}
	return i, nil
}
