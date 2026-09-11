package platformruntime

import "time"

type State string

const (
	StateCreated  State = "created"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateStopping State = "stopping"
	StateExited   State = "exited"
	StateFailed   State = "failed"
)

type OutputSource string

const (
	OutputStdout OutputSource = "stdout"
	OutputStderr OutputSource = "stderr"
)

// Spec describes a process-backed terminal session. Domain packages provide the
// executable and arguments; this package owns the OS process/stdio mechanics.
//
// Environment is treated as an overlay on the parent process environment by
// default. Set ReplaceEnvironment only when a caller intentionally wants a
// completely isolated environment.
type Spec struct {
	Executable         string
	Arguments          []string
	WorkingDirectory   string
	Environment        []string
	ReplaceEnvironment bool
	ShowWindow         bool
	HistoryLines       int
}

// OutputLine preserves whether a line came from stdout or stderr while the
// legacy OnLine hook remains available for game parsers that do not care.
type OutputLine struct {
	Sequence  uint64
	Source    OutputSource
	Text      string
	Timestamp time.Time
}

// ExitResult is emitted only after the process has actually been reaped.
type ExitResult struct {
	ExitCode int
	Err      error
	ExitedAt time.Time
}

// Snapshot is a concurrency-safe view of the generic process session.
type Snapshot struct {
	State     State
	PID       int
	StartedAt time.Time
	ExitedAt  time.Time
	ExitCode  *int
	Error     string
}

// Hooks let a domain interpret console output and process exit without moving
// game-specific parsing into the platform layer.
type Hooks struct {
	OnLine      func(string)
	OnOutput    func(OutputLine)
	OnReadError func(error)
	OnExit      func(ExitResult)
}

// RunSpec describes a bounded one-shot command. It is used for internal
// component calls such as XiaoYu Runtime RPC-style CLI invocations; long-lived
// game consoles should use Session instead.
type RunSpec struct {
	Spec
	Stdin          string
	MaxOutputBytes int
}

type RunResult struct {
	Stdout    string
	Stderr    string
	ExitCode  int
	TimedOut  bool
	Truncated bool
}

var DefaultTerminateTimeout = 5 * time.Second

const DefaultMaxOutputBytes = 8 * 1024 * 1024
const DefaultHistoryLines = 2000
const DefaultSubscriberBuffer = 256
