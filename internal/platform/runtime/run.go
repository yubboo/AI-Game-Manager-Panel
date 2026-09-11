package platformruntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
)

var ErrOutputTruncated = errors.New("runtime command output exceeded capture limit")

type captureBuffer struct {
	mu        sync.Mutex
	buf       bytes.Buffer
	max       int
	truncated bool
}

func (b *captureBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	original := len(p)
	if b.max <= 0 {
		b.max = DefaultMaxOutputBytes
	}
	remaining := b.max - b.buf.Len()
	if remaining <= 0 {
		b.truncated = true
		return original, nil
	}
	if len(p) > remaining {
		_, _ = b.buf.Write(p[:remaining])
		b.truncated = true
		return original, nil
	}
	_, _ = b.buf.Write(p)
	return original, nil
}

func (b *captureBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
func (b *captureBuffer) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}

// Run executes a bounded, one-shot process using the same OS isolation and
// environment rules as Session. It intentionally does not invoke a shell.
func Run(ctx context.Context, spec RunSpec) (RunResult, error) {
	cmd, err := newCommand(spec.Spec)
	if err != nil {
		return RunResult{ExitCode: -1}, err
	}
	limit := spec.MaxOutputBytes
	if limit <= 0 {
		limit = DefaultMaxOutputBytes
	}
	stdout := &captureBuffer{max: limit}
	stderr := &captureBuffer{max: limit}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if spec.Stdin != "" {
		cmd.Stdin = strings.NewReader(spec.Stdin)
	}
	if err := cmd.Start(); err != nil {
		return RunResult{ExitCode: -1}, err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var waitErr error
	timedOut := false
	select {
	case waitErr = <-done:
	case <-ctx.Done():
		timedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		if cmd.Process != nil {
			_ = killProcess(cmd.Process)
		}
		waitErr = <-done
		if waitErr == nil {
			waitErr = ctx.Err()
		} else {
			waitErr = fmt.Errorf("%w: %v", ctx.Err(), waitErr)
		}
	}

	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	result := RunResult{
		Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: exitCode,
		TimedOut: timedOut, Truncated: stdout.Truncated() || stderr.Truncated(),
	}
	if result.Truncated {
		return result, ErrOutputTruncated
	}
	return result, waitErr
}

// StartDetached launches a non-interactive child and reaps it asynchronously so
// callers do not leak process handles/zombies. It is for installer/open helpers,
// not for game consoles that need stdin/stdout ownership.
func StartDetached(spec Spec) (int, error) {
	cmd, err := newCommand(spec)
	if err != nil {
		return 0, err
	}
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	pid := cmd.Process.Pid
	go func() { _ = cmd.Wait() }()
	return pid, nil
}
