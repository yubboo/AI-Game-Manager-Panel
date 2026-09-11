package runtime

import (
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
)

type Manager struct {
	mu         sync.RWMutex
	procs      map[string]*Process
	logFactory LogSessionFactory
}

func NewManager(factories ...LogSessionFactory) *Manager {
	manager := &Manager{procs: make(map[string]*Process)}
	if len(factories) > 0 {
		manager.logFactory = factories[0]
	}
	return manager
}

func (m *Manager) Start(request StartRequest) (*Process, error) {
	key := processKey(request.ClusterPath, request.ShardName)

	m.mu.Lock()
	if existing := m.procs[key]; existing != nil {
		snapshot := existing.Snapshot()
		if isRunningLike(snapshot.Status) {
			m.mu.Unlock()
			return existing, nil
		}
	}

	var sink LogSessionSink
	var persistenceErr error
	if m.logFactory != nil {
		sink, persistenceErr = m.logFactory.StartSession(request)
	}
	process := NewProcess(request, sink, persistenceErr)
	if err := process.Start(); err != nil {
		m.mu.Unlock()
		return nil, err
	}
	m.procs[key] = process
	m.mu.Unlock()
	return process, nil
}

func (m *Manager) Get(clusterPath, shardName string) *Process {
	key := processKey(clusterPath, shardName)
	m.mu.RLock()
	process := m.procs[key]
	m.mu.RUnlock()
	return process
}

func (m *Manager) Lookup(clusterPath, shardName string) LookupResult {
	process := m.Get(clusterPath, shardName)
	if process == nil {
		return LookupResult{Found: false}
	}
	return LookupResult{Found: true, Process: process.Snapshot()}
}

func (m *Manager) Running() []*Process {
	m.mu.RLock()
	result := make([]*Process, 0, len(m.procs))
	for _, process := range m.procs {
		if isRunningLike(process.Snapshot().Status) {
			result = append(result, process)
		}
	}
	m.mu.RUnlock()
	return result
}

func (m *Manager) AnyRunning() bool { return len(m.Running()) > 0 }

func (m *Manager) Snapshots() []ProcessSnapshot {
	m.mu.RLock()
	result := make([]ProcessSnapshot, 0, len(m.procs))
	for _, process := range m.procs {
		if process != nil {
			result = append(result, process.Snapshot())
		}
	}
	m.mu.RUnlock()
	return result
}

func (m *Manager) ManagedPIDs() map[int]ProcessSnapshot {
	result := map[int]ProcessSnapshot{}
	for _, snapshot := range m.Snapshots() {
		// A PID from an already stopped/crashed process may later be reused by
		// Windows for an unrelated application. Never treat historical PIDs as
		// currently AGMP-managed port owners.
		if snapshot.PID > 0 && isRunningLike(snapshot.Status) {
			result[snapshot.PID] = snapshot
		}
	}
	return result
}

func (m *Manager) StopBlocking(clusterPath, shardName string, options StopOptions) bool {
	process := m.Get(clusterPath, shardName)
	if process == nil {
		return false
	}
	snapshot := process.Snapshot()
	if snapshot.Status == dedicated.StatusStopped || snapshot.Status == dedicated.StatusCrashed {
		return false
	}
	process.StopBlocking(options)
	return true
}

func (m *Manager) Stop(clusterPath, shardName string, options StopOptions) bool {
	process := m.Get(clusterPath, shardName)
	if process == nil {
		return false
	}
	snapshot := process.Snapshot()
	if snapshot.Status == dedicated.StatusStopped || snapshot.Status == dedicated.StatusCrashed {
		return false
	}
	go process.StopBlocking(options)
	return true
}

func (m *Manager) StopAllBlocking(options StopOptions) {
	processes := m.Running()
	var wg sync.WaitGroup
	wg.Add(len(processes))
	for _, process := range processes {
		go func(p *Process) {
			defer wg.Done()
			p.StopBlocking(options)
		}(process)
	}
	wg.Wait()
}

func isRunningLike(status dedicated.ServerStatus) bool {
	return status == dedicated.StatusStarting || status == dedicated.StatusRunning || status == dedicated.StatusStopping
}

func processKey(clusterPath, shardName string) string {
	path := filepath.Clean(strings.TrimSpace(clusterPath))
	shard := strings.TrimSpace(shardName)
	if goruntime.GOOS == "windows" {
		path = strings.ToLower(strings.ReplaceAll(path, "/", `\`))
		shard = strings.ToLower(shard)
	}
	return path + "\x00" + shard
}
