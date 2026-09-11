package host

import (
	"context"
	"errors"
	"testing"
	"time"
)

type blockingBrain struct{ started chan struct{} }
type noopExecutor struct{}

func (noopExecutor) Execute(context.Context, ToolCall) (ToolOutcome, error) {
	return ToolOutcome{}, nil
}

func (b *blockingBrain) Next(ctx context.Context, _ Frame) (Decision, error) {
	select {
	case b.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return Decision{}, ctx.Err()
}

func TestRunManagerPauseReleaseRoundTrip(t *testing.T) {
	runs := NewRunManager()
	state, err := runs.Create("test control ownership")
	if err != nil {
		t.Fatal(err)
	}
	state, err = runs.Pause(state.ID, "owner", "manual pause")
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != RunPaused || state.ControlOwner != ControlShared {
		t.Fatalf("unexpected pause state: %+v", state)
	}
	state, err = runs.ReleaseToXiaoYu(state.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != RunCreated || state.ControlOwner != ControlXiaoYu {
		t.Fatalf("unexpected release state: %+v", state)
	}
}

func TestHumanTakeoverInterruptsActiveBrainAtSafePoint(t *testing.T) {
	brain := &blockingBrain{started: make(chan struct{}, 1)}
	loop := NewLoop(brain, noopExecutor{}, nil, nil, NewTrace(), LoopConfig{MaxSteps: 4, MaxToolCalls: 2, MaxFailures: 1})
	runs := NewRunManager()
	state, err := runs.Create("human must win")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = runs.AdvanceAsync(context.Background(), state.ID, loop); err != nil {
		t.Fatal(err)
	}
	select {
	case <-brain.started:
	case <-time.After(time.Second):
		t.Fatal("brain did not begin advancing")
	}
	state, err = runs.Takeover(state.ID, "owner", "human takeover")
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != RunPaused || state.ControlOwner != ControlHuman {
		t.Fatalf("takeover must immediately expose human ownership: %+v", state)
	}
	deadline := time.Now().Add(time.Second)
	for runs.IsActive(state.ID) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if runs.IsActive(state.ID) {
		t.Fatal("active brain turn did not stop after human takeover")
	}
	state, _ = runs.Get(state.ID)
	if state.Status != RunPaused || state.ControlOwner != ControlHuman {
		t.Fatalf("finish path overwrote human takeover: %+v", state)
	}
}

func TestRunManagerCreateForPreservesInitiatorIdentity(t *testing.T) {
	runs := NewRunManager()
	state, err := runs.CreateFor("user-owned run", "user-123", "Alice")
	if err != nil {
		t.Fatal(err)
	}
	if state.InitiatorID != "user-123" || state.InitiatorName != "Alice" {
		t.Fatalf("initiator identity was not preserved: %+v", state)
	}
	stored, ok := runs.Get(state.ID)
	if !ok || stored.InitiatorID != "user-123" || stored.InitiatorName != "Alice" {
		t.Fatalf("stored run lost initiator identity: %+v", stored)
	}
}

func TestRunManagerOwnsTaskIDAndIgnoresCallerValue(t *testing.T) {
	runs := NewRunManager()
	state, err := runs.CreateForContext("server-owned task scope", "user-1", "User", RunContext{SessionID: "session-1", TaskID: "forged-other-run", ServerID: "server-1"})
	if err != nil {
		t.Fatal(err)
	}
	if state.Context.TaskID != state.ID {
		t.Fatalf("TaskID must equal server-owned Run ID: run=%s task=%s", state.ID, state.Context.TaskID)
	}
	if state.Context.TaskID == "forged-other-run" {
		t.Fatal("caller-controlled TaskID was trusted")
	}
	if state.Context.SessionID != "session-1" || state.Context.ServerID != "server-1" {
		t.Fatalf("non-task context was lost: %#v", state.Context)
	}
}

func TestRunManagerSteerInterruptsActiveTurnAndPreservesInstruction(t *testing.T) {
	brain := &blockingBrain{started: make(chan struct{}, 1)}
	loop := NewLoop(brain, noopExecutor{}, nil, nil, NewTrace(), LoopConfig{MaxSteps: 4, MaxToolCalls: 2, MaxFailures: 1})
	runs := NewRunManager()
	state, err := runs.CreateForContext("do old thing", "user-1", "User", RunContext{SessionID: "session-steer"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = runs.AdvanceAsync(context.Background(), state.ID, loop); err != nil {
		t.Fatal(err)
	}
	select {
	case <-brain.started:
	case <-time.After(time.Second):
		t.Fatal("brain did not begin advancing")
	}
	state, err = runs.Steer(state.ID, "不要继续旧方案，改成新方案", "User")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for runs.IsActive(state.ID) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if runs.IsActive(state.ID) {
		t.Fatal("active turn did not stop after steering")
	}
	state, _ = runs.Get(state.ID)
	if state.Status != RunCreated || state.Phase != PhasePlanning {
		t.Fatalf("steered run must be replannable, got %+v", state)
	}
	if len(state.Observations) == 0 {
		t.Fatal("steering instruction was not preserved")
	}
	last := state.Observations[len(state.Observations)-1]
	if last.Summary != "用户实时纠正" {
		t.Fatalf("unexpected steering observation: %+v", last)
	}
}

func TestThreadContextKeepsRecentSessionReceipts(t *testing.T) {
	runs := NewRunManager()
	first, err := runs.CreateForContext("把主题改成浅色", "user-1", "User", RunContext{SessionID: "session-history"})
	if err != nil {
		t.Fatal(err)
	}
	runs.mu.Lock()
	state := runs.runs[first.ID]
	state.Status = RunCompleted
	state.Message = "主题已切换为浅色"
	state.Observations = append(state.Observations, Observation{
		Step: 1, Tool: "settings.theme.set", Summary: "AGMP 主题已切换为 light。",
		Data: map[string]any{"decision": "allow", "data": map[string]any{"theme": "light", "previousTheme": "dark"}}, Time: time.Now(),
	})
	runs.runs[first.ID] = state
	runs.mu.Unlock()
	second, err := runs.CreateForContext("改回去", "user-1", "User", RunContext{SessionID: "session-history"})
	if err != nil {
		t.Fatal(err)
	}
	thread := runs.ThreadContext("session-history", second.ID, 6)
	if len(thread.RecentTurns) != 1 || thread.RecentTurns[0].Goal != "把主题改成浅色" {
		t.Fatalf("unexpected thread context: %#v", thread)
	}
	if len(thread.RecentTurns[0].Actions) != 1 {
		t.Fatalf("missing recent tool receipt: %#v", thread.RecentTurns[0])
	}
}

func TestTerminalRunCannotBeAdvancedAgain(t *testing.T) {
	runs := NewRunManager()
	state, err := runs.CreateForContext("done", "user-1", "User", RunContext{SessionID: "session-terminal"})
	if err != nil {
		t.Fatal(err)
	}
	runs.mu.Lock()
	terminal := runs.runs[state.ID]
	terminal.Status = RunCompleted
	terminal.Phase = PhaseCompleted
	terminal.Message = "finished"
	runs.runs[state.ID] = terminal
	runs.mu.Unlock()

	loop := NewLoop(nil, nil, nil, nil, NewTrace(), LoopConfig{})
	got, err := runs.AdvanceAsync(context.Background(), state.ID, loop)
	if !errors.Is(err, ErrRunTerminal) {
		t.Fatalf("expected ErrRunTerminal, got state=%+v err=%v", got, err)
	}
	current, ok := runs.Get(state.ID)
	if !ok || current.Status != RunCompleted || runs.IsActive(state.ID) {
		t.Fatalf("terminal run was reactivated: %+v active=%v", current, runs.IsActive(state.ID))
	}
}
