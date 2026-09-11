package host

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type listener struct {
	id      uint64
	owner   string
	kind    string
	handler EventHandler
}

type EventBus struct {
	mu        sync.RWMutex
	nextID    atomic.Uint64
	listeners map[uint64]listener
}

func NewEventBus() *EventBus { return &EventBus{listeners: make(map[uint64]listener)} }

func (b *EventBus) On(owner, kind string, handler EventHandler) func() {
	if handler == nil {
		return func() {}
	}
	id := b.nextID.Add(1)
	item := listener{id: id, owner: strings.TrimSpace(owner), kind: strings.TrimSpace(kind), handler: handler}
	b.mu.Lock()
	b.listeners[id] = item
	b.mu.Unlock()
	return func() {
		b.mu.Lock()
		delete(b.listeners, id)
		b.mu.Unlock()
	}
}

func (b *EventBus) Emit(event Event) {
	b.mu.RLock()
	items := make([]listener, 0, len(b.listeners))
	for _, item := range b.listeners {
		if item.kind == "" || item.kind == "*" || item.kind == event.Type {
			items = append(items, item)
		}
	}
	b.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].id < items[j].id })
	for _, item := range items {
		item.handler(event)
	}
}

func (b *EventBus) RemoveOwner(owner string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, item := range b.listeners {
		if item.owner == owner {
			delete(b.listeners, id)
		}
	}
}

type EventFilter func(Event) bool

type traceSubscriber struct {
	id     uint64
	ch     chan Event
	filter EventFilter
}

// Trace is XiaoYu Host's server-owned append-only event history. It is bounded
// so a long-running Web/headless deployment cannot grow memory forever. Live
// subscribers are fed from the same sequence source, which lets browsers
// reconnect with ?after=<sequence> without owning or stopping the Run itself.
type Trace struct {
	mu          sync.RWMutex
	next        uint64
	events      []Event
	maxHistory  int
	nextSubID   atomic.Uint64
	subscribers map[uint64]traceSubscriber
}

func NewTrace(limit ...int) *Trace {
	maxHistory := 8192
	if len(limit) > 0 && limit[0] > 0 {
		maxHistory = limit[0]
	}
	return &Trace{maxHistory: maxHistory, subscribers: make(map[uint64]traceSubscriber)}
}

// Append is append-only inside the retained history window. Payloads should
// contain summaries/IDs rather than secrets or raw credentials.
func (t *Trace) Append(event Event) Event {
	// Trace is observable by Web/Desktop clients, so sanitize at the final
	// append boundary instead of trusting every producer to remember redaction.
	event = sanitizeTraceEvent(event)
	t.mu.Lock()
	t.next++
	event.Sequence = t.next
	if event.Time.IsZero() {
		event.Time = time.Now()
	}
	t.events = append(t.events, event)
	if t.maxHistory > 0 && len(t.events) > t.maxHistory {
		drop := len(t.events) - t.maxHistory
		copy(t.events, t.events[drop:])
		t.events = t.events[:t.maxHistory]
	}
	for _, sub := range t.subscribers {
		if sub.filter != nil && !sub.filter(event) {
			continue
		}
		select {
		case sub.ch <- event:
		default:
			// Live UI is advisory. Never let a slow WebSocket/SSE/browser block
			// XiaoYu or a game server; discard one stale event and keep newest.
			select {
			case <-sub.ch:
			default:
			}
			select {
			case sub.ch <- event:
			default:
			}
		}
	}
	t.mu.Unlock()
	return event
}

func (t *Trace) List(after uint64, limit int) []Event {
	return t.ListFiltered(after, limit, nil)
}

// ListFiltered applies access filtering while holding the same retained event
// window. This avoids leaking another user's Run trace in multi-user Web mode.
func (t *Trace) ListFiltered(after uint64, limit int, filter EventFilter) []Event {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	matching := make([]Event, 0, min(limit, len(t.events)))
	for _, event := range t.events {
		if event.Sequence <= after || (filter != nil && !filter(event)) {
			continue
		}
		matching = append(matching, event)
	}
	if after == 0 && len(matching) > limit {
		matching = matching[len(matching)-limit:]
	} else if len(matching) > limit {
		matching = matching[:limit]
	}
	return append([]Event(nil), matching...)
}

// Subscribe returns an ordered server-side event stream beginning after the
// supplied sequence. Browser disconnect only cancels this subscription; it
// never cancels the underlying XiaoYu Run.
func (t *Trace) Subscribe(after uint64, buffer int) (<-chan Event, func()) {
	return t.SubscribeFiltered(after, buffer, nil)
}

// SubscribeFiltered is the live equivalent of ListFiltered. Filtering happens
// before backlog admission and before live delivery, so one noisy Run cannot
// evict another user's visible events from a bounded subscriber buffer.
func (t *Trace) SubscribeFiltered(after uint64, buffer int, filter EventFilter) (<-chan Event, func()) {
	if buffer < 64 {
		buffer = 256
	}
	if buffer > 4096 {
		buffer = 4096
	}
	ch := make(chan Event, buffer)
	id := t.nextSubID.Add(1)

	t.mu.Lock()
	backlog := make([]Event, 0, min(buffer, len(t.events)))
	for _, event := range t.events {
		if event.Sequence <= after || (filter != nil && !filter(event)) {
			continue
		}
		backlog = append(backlog, event)
	}
	if len(backlog) > buffer {
		backlog = backlog[len(backlog)-buffer:]
	}
	for _, event := range backlog {
		ch <- event
	}
	t.subscribers[id] = traceSubscriber{id: id, ch: ch, filter: filter}
	t.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			t.mu.Lock()
			if _, ok := t.subscribers[id]; ok {
				delete(t.subscribers, id)
				close(ch)
			}
			t.mu.Unlock()
		})
	}
	return ch, cancel
}

const traceStringLimit = 16 * 1024

func sanitizeTraceEvent(event Event) Event {
	event.Summary = truncateTraceString(event.Summary)
	if event.Data != nil {
		clean := make(map[string]any, len(event.Data))
		for key, value := range event.Data {
			if traceSecretKey(key) {
				clean[key] = "[REDACTED]"
				continue
			}
			clean[key] = sanitizeTraceValue(value, 0)
		}
		event.Data = clean
	}
	return event
}

func sanitizeTraceValue(value any, depth int) any {
	if depth >= 8 {
		return "[TRUNCATED]"
	}
	switch typed := value.(type) {
	case string:
		return truncateTraceString(typed)
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, child := range typed {
			if traceSecretKey(key) {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = sanitizeTraceValue(child, depth+1)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, child := range typed {
			out[i] = sanitizeTraceValue(child, depth+1)
		}
		return out
	default:
		return value
	}
}

func traceSecretKey(key string) bool {
	normalized := strings.ToLower(strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.TrimSpace(key)))
	for _, marker := range []string{"apikey", "authorization", "accesstoken", "refreshtoken", "password", "secret", "bearertoken", "credential", "privatekey"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func truncateTraceString(value string) string {
	if len(value) <= traceStringLimit {
		return value
	}
	return value[:traceStringLimit] + "…[truncated]"
}
