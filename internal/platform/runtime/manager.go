package platformruntime

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrSessionIDRequired = errors.New("terminal session id is required")
	ErrSessionExists     = errors.New("terminal session id already exists")
	ErrSessionNotFound   = errors.New("terminal session not found")
	ErrSessionActive     = errors.New("terminal session is still active")
)

type ManagedSnapshot struct {
	ID string
	Snapshot
}

// Manager owns named long-lived sessions for server consoles, SteamCMD or a
// persistent shell UI. It is deliberately protocol-agnostic and is the common
// ownership boundary used by higher-level server/domain managers.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

func NewManager() *Manager { return &Manager{sessions: make(map[string]*Session)} }

func (m *Manager) Start(id string, spec Spec, hooks Hooks) (*Session, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrSessionIDRequired
	}
	m.mu.Lock()
	if _, exists := m.sessions[id]; exists {
		m.mu.Unlock()
		return nil, ErrSessionExists
	}
	session := NewSession(spec, hooks)
	m.sessions[id] = session
	m.mu.Unlock()
	if err := session.Start(); err != nil {
		return session, err
	}
	return session, nil
}

func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}
	state := session.Snapshot().State
	if state == StateRunning || state == StateStarting || state == StateStopping {
		return ErrSessionActive
	}
	delete(m.sessions, id)
	return nil
}

func (m *Manager) Send(id, text string) error {
	session, ok := m.Get(id)
	if !ok {
		return ErrSessionNotFound
	}
	return session.Send(text)
}

func (m *Manager) SendLine(id, text string) error {
	session, ok := m.Get(id)
	if !ok {
		return ErrSessionNotFound
	}
	return session.SendLine(text)
}

func (m *Manager) History(id string, limit int) ([]OutputLine, error) {
	session, ok := m.Get(id)
	if !ok {
		return nil, ErrSessionNotFound
	}
	return session.History(limit), nil
}

func (m *Manager) Subscribe(id string, buffer int) (<-chan OutputLine, func(), error) {
	session, ok := m.Get(id)
	if !ok {
		return nil, nil, ErrSessionNotFound
	}
	ch, cancel := session.Subscribe(buffer)
	return ch, cancel, nil
}

func (m *Manager) Stop(id string, timeout time.Duration) error {
	session, ok := m.Get(id)
	if !ok {
		return ErrSessionNotFound
	}
	return stopSession(session, timeout)
}

// StopAll drains all currently active sessions concurrently. It is intended for
// AGMP application shutdown so no supervised child is left running accidentally.
// Domain services should issue their own graceful console command before this
// platform-level fallback when the game/protocol supports one.
func (m *Manager) StopAll(timeout time.Duration) map[string]error {
	m.mu.RLock()
	targets := make(map[string]*Session, len(m.sessions))
	for id, session := range m.sessions {
		state := session.Snapshot().State
		if state == StateRunning || state == StateStarting || state == StateStopping {
			targets[id] = session
		}
	}
	m.mu.RUnlock()

	var wg sync.WaitGroup
	var resultMu sync.Mutex
	result := make(map[string]error)
	for id, session := range targets {
		wg.Add(1)
		go func(id string, session *Session) {
			defer wg.Done()
			if err := stopSession(session, timeout); err != nil {
				resultMu.Lock()
				result[id] = err
				resultMu.Unlock()
			}
		}(id, session)
	}
	wg.Wait()
	return result
}

func stopSession(session *Session, timeout time.Duration) error {
	if err := session.Terminate(); err != nil && !errors.Is(err, ErrSessionNotRunning) {
		return err
	}
	if timeout <= 0 {
		timeout = DefaultTerminateTimeout
	}
	if session.Wait(timeout) {
		return nil
	}
	if err := session.Kill(); err != nil && !errors.Is(err, ErrSessionNotRunning) {
		return err
	}
	if !session.Wait(timeout) {
		return errors.New("terminal session did not exit after kill")
	}
	return nil
}

func (m *Manager) Snapshots() []ManagedSnapshot {
	m.mu.RLock()
	result := make([]ManagedSnapshot, 0, len(m.sessions))
	for id, session := range m.sessions {
		result = append(result, ManagedSnapshot{ID: id, Snapshot: session.Snapshot()})
	}
	m.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
