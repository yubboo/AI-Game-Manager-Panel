package xiaoyuruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
)

const workerStderrLimit = 64 * 1024
const workerOutputBuffer = 1024

// rpcWorker owns one long-lived `xiaoyu rpc` child process. OS process and
// stdio mechanics stay inside platform/runtime; this package owns only XiaoYu's
// JSON-RPC lifecycle. The Rust Runtime keeps Session/Job state in memory, so
// AGMP must not spawn a fresh process for every request once stateful native
// capabilities are enabled.
type rpcWorker struct {
	session   *platformruntime.Session
	output    <-chan platformruntime.OutputLine
	cancelSub func()
	stderr    *tailWriter
}

func startRPCWorker(path, root string) (*rpcWorker, error) {
	session := platformruntime.NewSession(platformruntime.Spec{
		Executable:       path,
		Arguments:        []string{"--root", root, "rpc"},
		WorkingDirectory: root,
		HistoryLines:     256,
	}, platformruntime.Hooks{})
	output, cancelSub := session.Subscribe(workerOutputBuffer)
	if err := session.Start(); err != nil {
		cancelSub()
		return nil, fmt.Errorf("启动 XiaoYu Persistent Runtime Worker 失败：%w", err)
	}
	return &rpcWorker{
		session:   session,
		output:    output,
		cancelSub: cancelSub,
		stderr:    newTailWriter(workerStderrLimit),
	}, nil
}

func (w *rpcWorker) call(ctx context.Context, request any) ([]byte, error) {
	if w == nil || w.session == nil {
		return nil, errors.New("XiaoYu Persistent Runtime Worker 未启动")
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if err := w.session.SendLine(string(raw)); err != nil {
		return nil, fmt.Errorf("写入 XiaoYu RPC Worker 失败：%w%s", err, w.stderrSuffix())
	}

	for {
		select {
		case <-ctx.Done():
			_ = w.kill()
			return nil, fmt.Errorf("XiaoYu Persistent Runtime Worker RPC 超时：%w%s", ctx.Err(), w.stderrSuffix())
		case line, ok := <-w.output:
			if !ok {
				if result, exists := w.session.Exit(); exists && result.Err != nil {
					return nil, fmt.Errorf("XiaoYu RPC Worker 已退出（code=%d）：%w%s", result.ExitCode, result.Err, w.stderrSuffix())
				}
				return nil, errors.New("XiaoYu RPC Worker 已退出" + w.stderrSuffix())
			}
			if line.Source == platformruntime.OutputStderr {
				_, _ = w.stderr.Write([]byte(line.Text + "\n"))
				continue
			}
			text := strings.TrimSpace(line.Text)
			if text == "" {
				continue
			}
			return []byte(text), nil
		}
	}
}

func (w *rpcWorker) kill() error {
	if w == nil || w.session == nil {
		return nil
	}
	err := w.session.Kill()
	if errors.Is(err, platformruntime.ErrSessionNotRunning) {
		return nil
	}
	return err
}

func (w *rpcWorker) close() {
	if w == nil {
		return
	}
	if w.cancelSub != nil {
		defer w.cancelSub()
	}
	if w.session == nil {
		return
	}

	// EOF is the normal shutdown path for the line-oriented Rust RPC loop.
	_ = w.session.CloseInput()
	if w.session.Wait(750 * time.Millisecond) {
		return
	}
	if err := w.session.Terminate(); err != nil && !errors.Is(err, platformruntime.ErrSessionNotRunning) {
		_ = w.session.Kill()
	}
	if w.session.Wait(750 * time.Millisecond) {
		return
	}
	_ = w.kill()
	_ = w.session.Wait(750 * time.Millisecond)
}

func (w *rpcWorker) stderrSuffix() string {
	if w == nil || w.stderr == nil {
		return ""
	}
	text := strings.TrimSpace(w.stderr.String())
	if text == "" {
		return ""
	}
	return "；stderr=" + text
}

type tailWriter struct {
	mu    sync.Mutex
	limit int
	data  []byte
}

func newTailWriter(limit int) *tailWriter {
	if limit <= 0 {
		limit = workerStderrLimit
	}
	return &tailWriter{limit: limit}
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	original := len(p)
	if original >= w.limit {
		w.data = append(w.data[:0], p[original-w.limit:]...)
		return original, nil
	}
	if overflow := len(w.data) + original - w.limit; overflow > 0 {
		copy(w.data, w.data[overflow:])
		w.data = w.data[:len(w.data)-overflow]
	}
	w.data = append(w.data, p...)
	return original, nil
}

func (w *tailWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(append([]byte(nil), w.data...))
}
