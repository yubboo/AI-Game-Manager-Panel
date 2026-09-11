package logcenter

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	dstruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"
)

const persistentQueueCapacity = 16384

type sessionWriter struct {
	store         *Store
	file          *os.File
	buffer        *bufio.Writer
	metaMu        sync.RWMutex
	meta          Session
	lines         chan dstruntime.LogLine
	flushRequests chan chan struct{}
	closeOnce     sync.Once
	done          chan struct{}
	finalMu       sync.Mutex
	final         *dstruntime.ProcessSnapshot
	dropped       atomic.Uint64
}

func newSessionWriter(store *Store, file *os.File, meta Session) *sessionWriter {
	return &sessionWriter{
		store:         store,
		file:          file,
		buffer:        bufio.NewWriterSize(file, 256*1024),
		meta:          meta,
		lines:         make(chan dstruntime.LogLine, persistentQueueCapacity),
		flushRequests: make(chan chan struct{}),
		done:          make(chan struct{}),
	}
}

func (w *sessionWriter) ID() string { w.metaMu.RLock(); defer w.metaMu.RUnlock(); return w.meta.ID }

func (w *sessionWriter) Append(line dstruntime.LogLine) {
	select {
	case w.lines <- line:
	default:
		w.dropped.Add(1)
	}
}

func (w *sessionWriter) Close(snapshot dstruntime.ProcessSnapshot) {
	w.finalMu.Lock()
	copied := snapshot
	w.final = &copied
	w.finalMu.Unlock()
	w.closeOnce.Do(func() { close(w.lines) })
	// Persistence must never hold the runtime lifecycle forever. The writer keeps
	// draining in the background if a slow or unhealthy disk exceeds this bound.
	select {
	case <-w.done:
	case <-time.After(5 * time.Second):
	}
}

func (w *sessionWriter) abort() {
	w.closeOnce.Do(func() { close(w.lines) })
	_ = w.file.Close()
	select {
	case <-w.done:
	default:
		close(w.done)
	}
}

func (w *sessionWriter) recordPersistenceError(err error) {
	if err == nil {
		return
	}
	w.metaMu.Lock()
	if w.meta.PersistenceError == "" {
		w.meta.PersistenceError = err.Error()
	}
	w.metaMu.Unlock()
}

func (w *sessionWriter) flushBuffer(syncFile bool) {
	if err := w.buffer.Flush(); err != nil {
		w.recordPersistenceError(err)
	}
	if syncFile {
		if err := w.file.Sync(); err != nil {
			w.recordPersistenceError(err)
		}
	}
}

func (w *sessionWriter) snapshot() Session {
	w.metaMu.RLock()
	value := w.meta
	w.metaMu.RUnlock()
	value.DroppedLines = w.dropped.Load()
	if stat, err := os.Stat(value.LogPath); err == nil {
		value.ByteSize = stat.Size()
	}
	return value
}

func (w *sessionWriter) flushSync(timeout time.Duration) {
	ack := make(chan struct{})
	select {
	case w.flushRequests <- ack:
	case <-w.done:
		return
	case <-time.After(timeout):
		return
	}
	select {
	case <-ack:
	case <-w.done:
	case <-time.After(timeout):
	}
}

func (w *sessionWriter) run() {
	defer close(w.done)
	defer w.store.removeWriter(w.ID())
	flushTicker := time.NewTicker(time.Second)
	metaTicker := time.NewTicker(5 * time.Second)
	defer flushTicker.Stop()
	defer metaTicker.Stop()

	for {
		select {
		case line, ok := <-w.lines:
			if !ok {
				w.finish()
				return
			}
			w.writeLine(line)
		case ack := <-w.flushRequests:
			// A flush request is a read barrier. Because the log queue and flush
			// requests are separate channels, select may observe the flush before
			// older queued lines. Drain the backlog that existed when the barrier
			// was accepted, then flush it. New lines may continue queueing without
			// making a UI read wait forever.
			pending := len(w.lines)
			for i := 0; i < pending; i++ {
				line, ok := <-w.lines
				if !ok {
					break
				}
				w.writeLine(line)
			}
			w.flushBuffer(true)
			close(ack)
		case <-flushTicker.C:
			w.flushBuffer(false)
		case <-metaTicker.C:
			w.flushBuffer(false)
			w.recordPersistenceError(writeSessionMeta(w.snapshot()))
		}
	}
}

func (w *sessionWriter) writeLine(line dstruntime.LogLine) {
	text := strings.ReplaceAll(line.Text, "\x00", "")
	written, writeErr := w.buffer.WriteString(text + "\n")
	w.recordPersistenceError(writeErr)
	w.metaMu.Lock()
	w.meta.LineCount++
	w.meta.ByteSize += int64(written)
	w.meta.DroppedLines = w.dropped.Load()
	w.metaMu.Unlock()
}

func (w *sessionWriter) finish() {
	w.flushBuffer(true)
	if err := w.file.Close(); err != nil {
		w.recordPersistenceError(err)
	}
	w.finalMu.Lock()
	final := w.final
	w.finalMu.Unlock()
	w.metaMu.Lock()
	if final != nil {
		w.meta.Status = string(final.Status)
		w.meta.PID = final.PID
		w.meta.WorldReady = final.WorldReady
		w.meta.ExitCode = final.ExitCode
		w.meta.Error = final.Error
		w.meta.EndedAt = final.UpdatedAt
	}
	w.meta.DroppedLines = w.dropped.Load()
	value := w.meta
	w.metaMu.Unlock()
	if stat, err := os.Stat(value.LogPath); err == nil {
		value.ByteSize = stat.Size()
	}
	_ = writeSessionMeta(value)
}
