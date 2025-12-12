package sessions

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

type Manager struct {
	baseDir     string
	createdDirs map[string]bool
	dirsMutex   sync.Mutex
}

func NewManager(baseDir string) *Manager {
	return &Manager{
		baseDir:     baseDir,
		createdDirs: make(map[string]bool),
	}
}

func (m *Manager) GetWorkspaceDir(sessionID string) (string, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	var sessionDir string
	if m.baseDir != "" {
		sessionDir = filepath.Join(m.baseDir, sessionID)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		sessionDir = filepath.Join(homeDir, ".kiro", "sub-agents", sessionID)
	}

	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", err
	}

	m.dirsMutex.Lock()
	m.createdDirs[sessionDir] = true
	m.dirsMutex.Unlock()

	return sessionDir, nil
}

func (m *Manager) GetSessionID(sessionDir string) string {
	return filepath.Base(sessionDir)
}

func (m *Manager) Cleanup() {
	m.dirsMutex.Lock()
	defer m.dirsMutex.Unlock()

	for dir := range m.createdDirs {
		os.RemoveAll(dir)
	}
	m.createdDirs = make(map[string]bool)
}
