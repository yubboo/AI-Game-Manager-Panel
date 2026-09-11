package runtime

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
)

var shutdownCommandRE = regexp.MustCompile(`(?i)^\s*c_shutdown\s*\(\s*(?:(?:true|false|[01])\s*)?\)\s*;?\s*$`)

var (
	ErrProcessNotRunning = errors.New("DST server process is not running")
	ErrProcessStopping   = errors.New("DST server process is stopping")
)

// Process keeps only DST-specific runtime state. OS process and stdio mechanics
// live in platform/runtime so future games can reuse the same terminal session.
type Process struct {
	mu sync.RWMutex

	clusterName string
	clusterPath string
	shardName   string
	role        dedicated.ShardRole
	spec        dedicated.LaunchSpec
	session     *platformruntime.Session

	logSink             LogSessionSink
	logPersistenceError string

	status              dedicated.ServerStatus
	worldReady          bool
	realStartSeen       bool
	intentionalShutdown bool
	exitCode            *int
	startedAt           int64
	updatedAt           int64
	lastError           string

	nextSequence uint64
	streamLogs   []LogLine
	recentLogs   []string
}

func NewProcess(request StartRequest, sink LogSessionSink, persistenceErr error) *Process {
	now := time.Now().Unix()
	process := &Process{
		clusterName: request.ClusterName,
		clusterPath: request.ClusterPath,
		shardName:   request.ShardName,
		role:        request.Spec.Role,
		spec:        request.Spec,
		logSink:     sink,
		status:      dedicated.StatusStarting,
		startedAt:   now,
		updatedAt:   now,
		streamLogs:  make([]LogLine, 0, 256),
		recentLogs:  make([]string, 0, RecentLogLimit),
	}
	if persistenceErr != nil {
		process.logPersistenceError = persistenceErr.Error()
	}
	return process
}

func (p *Process) Start() error {
	p.mu.Lock()
	if p.session != nil {
		p.mu.Unlock()
		return errors.New("DST server process has already been started")
	}

	session := platformruntime.NewSession(platformruntime.Spec{
		Executable:       p.spec.Executable,
		Arguments:        append([]string(nil), p.spec.Arguments...),
		WorkingDirectory: p.spec.WorkingDirectory,
	}, platformruntime.Hooks{
		OnLine:      p.consumeLine,
		OnReadError: p.handleReadError,
		OnExit:      p.handleExit,
	})
	p.session = session
	p.status = dedicated.StatusRunning
	p.updatedAt = time.Now().Unix()
	p.mu.Unlock()

	if err := session.Start(); err != nil {
		p.mu.Lock()
		p.status = dedicated.StatusCrashed
		p.lastError = err.Error()
		p.updatedAt = time.Now().Unix()
		p.mu.Unlock()
		if p.logSink != nil {
			p.logSink.Close(p.Snapshot())
		}
		return err
	}
	return nil
}

func (p *Process) handleReadError(err error) {
	p.mu.Lock()
	if p.lastError == "" && !p.intentionalShutdown {
		p.lastError = fmt.Sprintf("读取服务器控制台失败: %v", err)
	}
	p.updatedAt = time.Now().Unix()
	p.mu.Unlock()
}

func (p *Process) handleExit(result platformruntime.ExitResult) {
	p.mu.Lock()
	p.exitCode = intPtr(result.ExitCode)
	p.updatedAt = time.Now().Unix()
	expected := p.intentionalShutdown || p.status == dedicated.StatusStopping
	if expected {
		p.status = dedicated.StatusStopped
	} else {
		p.status = dedicated.StatusCrashed
		if result.Err != nil {
			p.lastError = result.Err.Error()
		} else if p.lastError == "" {
			p.lastError = fmt.Sprintf("服务器进程意外退出，退出代码 %d", result.ExitCode)
		}
	}
	p.mu.Unlock()
	if p.logSink != nil {
		p.logSink.Close(p.Snapshot())
	}
}

func (p *Process) consumeLine(line string) {
	now := time.Now().Unix()

	p.mu.Lock()
	p.nextSequence++
	entry := LogLine{Sequence: p.nextSequence, Timestamp: now, Text: line}
	p.streamLogs = append(p.streamLogs, entry)
	if len(p.streamLogs) > StreamLogLimit {
		drop := len(p.streamLogs) - StreamLogLimit
		copy(p.streamLogs, p.streamLogs[drop:])
		p.streamLogs = p.streamLogs[:StreamLogLimit]
	}

	p.recentLogs = append(p.recentLogs, line)
	if len(p.recentLogs) > RecentLogLimit {
		drop := len(p.recentLogs) - RecentLogLimit
		copy(p.recentLogs, p.recentLogs[drop:])
		p.recentLogs = p.recentLogs[:RecentLogLimit]
	}

	if !p.worldReady {
		isMaster := p.role == dedicated.ShardMaster
		startSeen, readyNow := dedicated.AdvanceWorldReadyMarker(line, isMaster, p.realStartSeen)
		p.realStartSeen = startSeen
		if readyNow {
			p.worldReady = true
		}
	}
	p.updatedAt = now
	sink := p.logSink
	p.mu.Unlock()
	if sink != nil {
		sink.Append(entry)
	}
}

func (p *Process) Snapshot() ProcessSnapshot {
	p.mu.RLock()
	session := p.session
	snapshot := ProcessSnapshot{
		ClusterName:         p.clusterName,
		ClusterPath:         p.clusterPath,
		ShardName:           p.shardName,
		Role:                p.role,
		Status:              p.status,
		WorldReady:          p.worldReady,
		IntentionalShutdown: p.intentionalShutdown,
		ExitCode:            copyIntPtr(p.exitCode),
		StartedAt:           p.startedAt,
		UpdatedAt:           p.updatedAt,
		Error:               p.lastError,
		LogCursor:           p.nextSequence,
		LogSessionID:        logSessionID(p.logSink),
		LogPersistenceError: p.logPersistenceError,
	}
	p.mu.RUnlock()
	if session != nil {
		snapshot.PID = session.PID()
	}
	return snapshot
}

func (p *Process) RecentLogLines() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]string, len(p.recentLogs))
	copy(result, p.recentLogs)
	return result
}

func (p *Process) ReadLogs(after uint64, limit int) LogBatch {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	batch := LogBatch{Lines: []LogLine{}, NextCursor: after}
	if len(p.streamLogs) == 0 {
		if p.nextSequence > after {
			batch.NextCursor = p.nextSequence
		}
		return batch
	}

	oldest := p.streamLogs[0].Sequence
	if after+1 < oldest {
		batch.Dropped = true
		after = oldest - 1
	}

	for _, line := range p.streamLogs {
		if line.Sequence <= after {
			continue
		}
		batch.Lines = append(batch.Lines, line)
		batch.NextCursor = line.Sequence
		if len(batch.Lines) >= limit {
			break
		}
	}
	return batch
}

func (p *Process) SendCommand(text string) error {
	p.mu.Lock()
	session := p.session
	if session == nil || p.status == dedicated.StatusStopped || p.status == dedicated.StatusCrashed {
		p.mu.Unlock()
		return ErrProcessNotRunning
	}
	if p.status == dedicated.StatusStopping && !shutdownCommandRE.MatchString(text) {
		p.mu.Unlock()
		return ErrProcessStopping
	}
	isShutdown := shutdownCommandRE.MatchString(text)
	p.mu.Unlock()

	if err := session.SendLine(text); err != nil {
		return err
	}

	p.mu.Lock()
	if isShutdown {
		p.intentionalShutdown = true
		p.status = dedicated.StatusStopping
	}
	p.updatedAt = time.Now().Unix()
	p.mu.Unlock()
	return nil
}

func (p *Process) RequestShutdown() error {
	return p.SendCommand("c_shutdown()")
}

func (p *Process) StopBlocking(options StopOptions) {
	if options.GracefulTimeout <= 0 {
		options.GracefulTimeout = 30 * time.Second
	}
	if options.TermTimeout <= 0 {
		options.TermTimeout = platformruntime.DefaultTerminateTimeout
	}

	p.mu.Lock()
	if p.status == dedicated.StatusStopped || p.status == dedicated.StatusCrashed {
		p.mu.Unlock()
		return
	}
	p.intentionalShutdown = true
	p.status = dedicated.StatusStopping
	p.updatedAt = time.Now().Unix()
	session := p.session
	p.mu.Unlock()

	if err := p.RequestShutdown(); err == nil && session != nil {
		if session.Wait(options.GracefulTimeout) {
			return
		}
	}

	if session != nil {
		_ = session.Terminate()
		if session.Wait(options.TermTimeout) {
			return
		}
		_ = session.Kill()
		if session.Wait(options.TermTimeout) {
			return
		}
	}

	// Never report a process as stopped until the shared runtime observed the
	// real process exit. Otherwise a new shard could start while stale ports are
	// still owned by the old process.
	p.mu.Lock()
	if p.status != dedicated.StatusStopped && p.status != dedicated.StatusCrashed {
		p.status = dedicated.StatusStopping
		if p.lastError == "" {
			p.lastError = "强制结束后仍未在等待时间内确认进程退出；服务器端口可能仍被占用"
		}
		p.updatedAt = time.Now().Unix()
	}
	p.mu.Unlock()
}

func logSessionID(sink LogSessionSink) string {
	if sink == nil {
		return ""
	}
	return sink.ID()
}

func intPtr(value int) *int { return &value }

func copyIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
