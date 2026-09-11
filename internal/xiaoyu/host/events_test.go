package host

import (
	"testing"
	"time"
)

func TestTraceInitialListReturnsNewestWindow(t *testing.T) {
	trace := NewTrace(5)
	for i := 0; i < 8; i++ {
		trace.Append(Event{Type: "test"})
	}
	items := trace.List(0, 3)
	if len(items) != 3 || items[0].Sequence != 6 || items[2].Sequence != 8 {
		t.Fatalf("initial list should return newest retained window: %+v", items)
	}
}

func TestTraceSubscribeBacklogKeepsNewestWhenBufferBounded(t *testing.T) {
	trace := NewTrace(400)
	for i := 0; i < 300; i++ {
		trace.Append(Event{Type: "test"})
	}
	stream, cancel := trace.Subscribe(0, 64)
	defer cancel()
	first := <-stream
	if first.Sequence != 237 { // Subscribe enforces a minimum buffer of 64.
		t.Fatalf("expected newest 64-event backlog to start at 237, got %d", first.Sequence)
	}
	var last Event
	for i := 1; i < 64; i++ {
		last = <-stream
	}
	if last.Sequence != 300 {
		t.Fatalf("expected latest backlog event 300, got %d", last.Sequence)
	}
}

func TestTraceRedactsSecretLookingPayloadsAtBoundary(t *testing.T) {
	trace := NewTrace(10)
	stored := trace.Append(Event{Type: "test", Summary: "ok", Data: map[string]any{
		"apiKey": "sk-should-never-leak",
		"nested": map[string]any{"Authorization": "Bearer secret", "safe": "visible"},
	}})
	if stored.Data["apiKey"] != "[REDACTED]" {
		t.Fatalf("api key was not redacted: %+v", stored.Data)
	}
	nested, ok := stored.Data["nested"].(map[string]any)
	if !ok || nested["Authorization"] != "[REDACTED]" || nested["safe"] != "visible" {
		t.Fatalf("nested trace redaction failed: %+v", stored.Data)
	}
	items := trace.List(0, 10)
	if len(items) != 1 || items[0].Data["apiKey"] != "[REDACTED]" {
		t.Fatalf("retained trace contains an unredacted secret: %+v", items)
	}
}

func TestTraceFilterKeepsRunsIsolatedForBacklogAndLiveEvents(t *testing.T) {
	trace := NewTrace(20)
	trace.Append(Event{Type: "run", RunID: "A"})
	trace.Append(Event{Type: "run", RunID: "B"})
	trace.Append(Event{Type: "host"})
	filterA := func(event Event) bool { return event.RunID == "" || event.RunID == "A" }
	items := trace.ListFiltered(0, 10, filterA)
	if len(items) != 2 || items[0].RunID != "A" || items[1].RunID != "" {
		t.Fatalf("filtered history leaked another run: %+v", items)
	}
	stream, cancel := trace.SubscribeFiltered(items[len(items)-1].Sequence, 64, filterA)
	defer cancel()
	trace.Append(Event{Type: "run", RunID: "B"})
	trace.Append(Event{Type: "run", RunID: "A"})
	select {
	case event := <-stream:
		if event.RunID != "A" {
			t.Fatalf("live stream leaked another run: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("filtered live event not delivered")
	}
}
