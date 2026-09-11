package platformruntime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	ErrSessionStarted    = errors.New("terminal session has already been started")
	ErrSessionNotRunning = errors.New("terminal session is not running")
)

// Session is AGMP's common process-backed terminal primitive. It deliberately
// knows nothing about DST, Minecraft, SteamCMD or any other domain protocol.
type Session struct {
	mu       sync.RWMutex
	writeMu  sync.Mutex
	outputMu sync.RWMutex

	spec  Spec
	hooks Hooks

	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	readWG    sync.WaitGroup
	done      chan struct{}
	doneOnce  sync.Once
	started   bool
	state     State
	startedAt time.Time
	exit      *ExitResult

	history          []OutputLine
	historyLimit     int
	historyStart     int
	historyCount     int
	subscribers      map[uint64]chan OutputLine
	nextSubscriberID uint64
	nextSequence     uint64
}

func NewSession(spec Spec, hooks Hooks) *Session {
	historyLimit := spec.HistoryLines
	if historyLimit == 0 {
		historyLimit = DefaultHistoryLines
	}
	if historyLimit < 0 {
		historyLimit = 0
	}
	history := []OutputLine(nil)
	if historyLimit > 0 {
		history = make([]OutputLine, historyLimit)
	}
	return &Session{
		spec: spec, hooks: hooks, done: make(chan struct{}), state: StateCreated,
		history: history, historyLimit: historyLimit, subscribers: make(map[uint64]chan OutputLine),
	}
}

func (s *Session) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return ErrSessionStarted
	}
	s.started = true
	s.state = StateStarting
	s.mu.Unlock()

	cmd, err := newCommand(s.spec)
	if err != nil {
		s.finishStartFailure(err)
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		s.finishStartFailure(err)
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		s.finishStartFailure(err)
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		s.finishStartFailure(err)
		return err
	}

	s.mu.Lock()
	s.cmd = cmd
	s.stdin = stdin
	s.stdout = stdout
	s.stderr = stderr
	s.mu.Unlock()

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		s.finishStartFailure(err)
		return err
	}

	s.mu.Lock()
	s.state = StateRunning
	s.startedAt = time.Now()
	s.mu.Unlock()

	s.readWG.Add(2)
	go s.readLoop(OutputStdout, stdout)
	go s.readLoop(OutputStderr, stderr)
	go s.waitLoop(cmd)
	return nil
}

func (s *Session) readLoop(source OutputSource, reader io.ReadCloser) {
	defer s.readWG.Done()
	defer reader.Close()

	s.mu.RLock()
	hooks := s.hooks
	s.mu.RUnlock()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		text := strings.TrimSuffix(scanner.Text(), "\r")
		line := s.publishOutput(OutputLine{Source: source, Text: text, Timestamp: time.Now()})
		if hooks.OnOutput != nil {
			hooks.OnOutput(line)
		}
		if hooks.OnLine != nil {
			hooks.OnLine(text)
		}
	}
	if err := scanner.Err(); err != nil && hooks.OnReadError != nil {
		hooks.OnReadError(fmt.Errorf("read terminal %s: %w", source, err))
	}
}

func (s *Session) waitLoop(cmd *exec.Cmd) {
	err := cmd.Wait()
	s.readWG.Wait()

	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	} else if err == nil {
		exitCode = 0
	}
	s.finish(ExitResult{ExitCode: exitCode, Err: err, ExitedAt: time.Now()})
}

func (s *Session) finishStartFailure(err error) {
	result := ExitResult{ExitCode: -1, Err: err, ExitedAt: time.Now()}
	s.doneOnce.Do(func() {
		s.mu.Lock()
		s.state = StateFailed
		copyResult := result
		s.exit = &copyResult
		hooks := s.hooks
		s.mu.Unlock()
		s.closeSubscribers()
		close(s.done)
		if hooks.OnExit != nil {
			hooks.OnExit(result)
		}
	})
}

func (s *Session) finish(result ExitResult) {
	s.doneOnce.Do(func() {
		s.mu.Lock()
		s.state = StateExited
		copyResult := result
		s.exit = &copyResult
		hooks := s.hooks
		s.mu.Unlock()
		s.closeSubscribers()
		close(s.done)
		if hooks.OnExit != nil {
			hooks.OnExit(result)
		}
	})
}

// Send writes raw text to stdin without appending a line terminator.
func (s *Session) Send(text string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	s.mu.RLock()
	stdin := s.stdin
	state := s.state
	done := s.done
	s.mu.RUnlock()
	if stdin == nil || (state != StateRunning && state != StateStopping) {
		return ErrSessionNotRunning
	}
	select {
	case <-done:
		return ErrSessionNotRunning
	default:
	}
	_, err := io.WriteString(stdin, text)
	return err
}

func (s *Session) SendLine(text string) error {
	return s.Send(text + "\n")
}

func (s *Session) CloseInput() error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	s.mu.Lock()
	stdin := s.stdin
	s.stdin = nil
	s.mu.Unlock()
	if stdin == nil {
		return nil
	}
	return stdin.Close()
}

func (s *Session) PID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cmd == nil {
		return 0
	}
	if s.cmd.Process == nil {
		return 0
	}
	return s.cmd.Process.Pid
}

func (s *Session) Done() <-chan struct{} { return s.done }

func (s *Session) Wait(timeout time.Duration) bool {
	if timeout <= 0 {
		select {
		case <-s.done:
			return true
		default:
			return false
		}
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-s.done:
		return true
	case <-timer.C:
		return false
	}
}

func (s *Session) WaitContext(ctx context.Context) error {
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Session) Exit() (ExitResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.exit == nil {
		return ExitResult{}, false
	}
	return *s.exit, true
}

func (s *Session) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := Snapshot{State: s.state, StartedAt: s.startedAt}
	if s.cmd != nil {
		if s.cmd.Process != nil {
			snapshot.PID = s.cmd.Process.Pid
		}
	}
	if s.exit != nil {
		exitCode := s.exit.ExitCode
		snapshot.ExitCode = &exitCode
		snapshot.ExitedAt = s.exit.ExitedAt
		if s.exit.Err != nil {
			snapshot.Error = s.exit.Err.Error()
		}
	}
	return snapshot
}

func (s *Session) Terminate() error {
	s.mu.RLock()
	cmd := s.cmd
	state := s.state
	s.mu.RUnlock()
	if cmd == nil || (state != StateRunning && state != StateStopping) {
		return ErrSessionNotRunning
	}
	if cmd.Process == nil {
		return ErrSessionNotRunning
	}
	if err := terminateProcess(cmd.Process); err != nil {
		return err
	}
	s.mu.Lock()
	if s.state == StateRunning {
		s.state = StateStopping
	}
	s.mu.Unlock()
	return nil
}

func (s *Session) Kill() error {
	s.mu.RLock()
	cmd := s.cmd
	state := s.state
	s.mu.RUnlock()
	if cmd == nil || (state != StateRunning && state != StateStopping) {
		return ErrSessionNotRunning
	}
	if cmd.Process == nil {
		return ErrSessionNotRunning
	}
	if err := killProcess(cmd.Process); err != nil {
		return err
	}
	s.mu.Lock()
	if s.state == StateRunning {
		s.state = StateStopping
	}
	s.mu.Unlock()
	return nil
}
