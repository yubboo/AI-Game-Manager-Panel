package minecraft

import (
	"errors"
	"strings"
	"sync"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
)

const logLimit = 2000

type runtimeProcess struct {
	mu        sync.RWMutex
	id        string
	session   *platformruntime.Session
	state     string
	ready     bool
	startedAt int64
	updatedAt int64
	exitCode  *int
	err       string
	seq       uint64
	logs      []LogLine
}

func newRuntimeProcess(id string, spec platformruntime.Spec) *runtimeProcess {
	p := &runtimeProcess{id: id, state: "starting", startedAt: time.Now().Unix(), updatedAt: time.Now().Unix(), logs: make([]LogLine, 0, 256)}
	p.session = platformruntime.NewSession(spec, platformruntime.Hooks{OnLine: p.consume, OnReadError: p.readErr, OnExit: p.exit})
	return p
}
func (p *runtimeProcess) start() error {
	if err := p.session.Start(); err != nil {
		p.mu.Lock()
		p.state = "failed"
		p.err = err.Error()
		p.updatedAt = time.Now().Unix()
		p.mu.Unlock()
		return err
	}
	p.mu.Lock()
	p.state = "running"
	p.updatedAt = time.Now().Unix()
	p.mu.Unlock()
	return nil
}
func (p *runtimeProcess) consume(line string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.seq++
	p.logs = append(p.logs, LogLine{Sequence: p.seq, Timestamp: time.Now().Unix(), Text: line})
	if len(p.logs) > logLimit {
		p.logs = append([]LogLine(nil), p.logs[len(p.logs)-logLimit:]...)
	}
	if strings.Contains(line, "Done (") && (strings.Contains(line, "For help") || strings.Contains(line, "For help, type")) {
		p.ready = true
	}
	p.updatedAt = time.Now().Unix()
}
func (p *runtimeProcess) readErr(err error) {
	if err == nil {
		return
	}
	p.mu.Lock()
	if p.err == "" {
		p.err = err.Error()
	}
	p.updatedAt = time.Now().Unix()
	p.mu.Unlock()
}
func (p *runtimeProcess) exit(result platformruntime.ExitResult) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v := result.ExitCode
	p.exitCode = &v
	if p.state == "stopping" {
		p.state = "stopped"
	} else {
		p.state = "failed"
		if result.Err != nil {
			p.err = result.Err.Error()
		}
	}
	p.updatedAt = time.Now().Unix()
}
func (p *runtimeProcess) snapshot() RuntimeSnapshot {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pid := 0
	if p.session != nil {
		pid = p.session.PID()
	}
	return RuntimeSnapshot{InstanceID: p.id, State: p.state, PID: pid, Ready: p.ready, StartedAt: p.startedAt, UpdatedAt: p.updatedAt, ExitCode: p.exitCode, Error: p.err, LogCursor: p.seq}
}
func (p *runtimeProcess) logsAfter(after uint64, limit int) LogBatch {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	b := LogBatch{Lines: []LogLine{}, NextCursor: after}
	if len(p.logs) == 0 {
		return b
	}
	oldest := p.logs[0].Sequence
	if after+1 < oldest {
		b.Dropped = true
		after = oldest - 1
	}
	for _, line := range p.logs {
		if line.Sequence <= after {
			continue
		}
		b.Lines = append(b.Lines, line)
		b.NextCursor = line.Sequence
		if len(b.Lines) >= limit {
			break
		}
	}
	return b
}
func (p *runtimeProcess) stop() error {
	p.mu.Lock()
	if p.state != "running" && p.state != "starting" {
		p.mu.Unlock()
		return errors.New("Minecraft 实例未运行")
	}
	p.state = "stopping"
	p.updatedAt = time.Now().Unix()
	s := p.session
	p.mu.Unlock()
	_ = s.SendLine("stop")
	if s.Wait(30 * time.Second) {
		return nil
	}
	return s.Terminate()
}

type runtimeManager struct {
	mu    sync.RWMutex
	procs map[string]*runtimeProcess
}

func newRuntimeManager() *runtimeManager { return &runtimeManager{procs: map[string]*runtimeProcess{}} }
func (m *runtimeManager) get(id string) *runtimeProcess {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.procs[id]
}
func (m *runtimeManager) start(id string, spec platformruntime.Spec) (*runtimeProcess, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p := m.procs[id]; p != nil {
		snap := p.snapshot()
		if snap.State == "running" || snap.State == "starting" {
			return p, nil
		}
	}
	p := newRuntimeProcess(id, spec)
	if err := p.start(); err != nil {
		return nil, err
	}
	m.procs[id] = p
	return p, nil
}
