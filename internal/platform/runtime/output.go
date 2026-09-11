package platformruntime

import (
	"sync"
	"time"
)

// publishOutput records a line and broadcasts it to live subscribers without
// allowing a slow UI/WebSocket consumer to block the process stdout/stderr
// reader. History uses a fixed-capacity ring so high-volume game consoles do
// not copy the full scrollback buffer for every line.
func (s *Session) publishOutput(line OutputLine) OutputLine {
	s.outputMu.Lock()
	s.nextSequence++
	line.Sequence = s.nextSequence
	if line.Timestamp.IsZero() {
		line.Timestamp = time.Now()
	}
	if s.historyLimit > 0 {
		if s.historyCount < s.historyLimit {
			index := (s.historyStart + s.historyCount) % s.historyLimit
			s.history[index] = line
			s.historyCount++
		} else {
			s.history[s.historyStart] = line
			s.historyStart = (s.historyStart + 1) % s.historyLimit
		}
	}
	for _, ch := range s.subscribers {
		select {
		case ch <- line:
		default:
			// Prefer current output over stale buffered output. Subscriptions are
			// intentionally lossy under backpressure; Sequence exposes gaps and
			// durable in-process scrollback remains available through History().
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- line:
			default:
			}
		}
	}
	s.outputMu.Unlock()
	return line
}

// History returns the newest retained console lines in chronological order.
// A non-positive limit returns all retained lines.
func (s *Session) History(limit int) []OutputLine {
	s.outputMu.RLock()
	defer s.outputMu.RUnlock()
	count := s.historyCount
	if limit > 0 && count > limit {
		count = limit
	}
	result := make([]OutputLine, count)
	if count == 0 {
		return result
	}
	offset := s.historyCount - count
	for i := 0; i < count; i++ {
		index := (s.historyStart + offset + i) % s.historyLimit
		result[i] = s.history[index]
	}
	return result
}

// Subscribe returns a non-blocking live console stream. The returned cancel
// function is idempotent and must be called when a UI/WebSocket consumer goes
// away. Consumers that need initial scrollback should call History() first.
func (s *Session) Subscribe(buffer int) (<-chan OutputLine, func()) {
	if buffer <= 0 {
		buffer = DefaultSubscriberBuffer
	}
	ch := make(chan OutputLine, buffer)
	select {
	case <-s.done:
		close(ch)
		return ch, func() {}
	default:
	}

	s.outputMu.Lock()
	select {
	case <-s.done:
		s.outputMu.Unlock()
		close(ch)
		return ch, func() {}
	default:
	}
	s.nextSubscriberID++
	id := s.nextSubscriberID
	s.subscribers[id] = ch
	s.outputMu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			s.outputMu.Lock()
			if current, ok := s.subscribers[id]; ok {
				delete(s.subscribers, id)
				close(current)
			}
			s.outputMu.Unlock()
		})
	}
	return ch, cancel
}

func (s *Session) closeSubscribers() {
	s.outputMu.Lock()
	for id, ch := range s.subscribers {
		delete(s.subscribers, id)
		close(ch)
	}
	s.outputMu.Unlock()
}
