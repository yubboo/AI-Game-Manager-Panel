package platformruntime

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRuntimeHelperProcess(t *testing.T) {
	mode := os.Getenv("AGMP_PLATFORM_RUNTIME_HELPER_V2")
	if mode == "" {
		return
	}
	switch mode {
	case "stream":
		fmt.Println("ready")
		fmt.Fprintln(os.Stderr, "stderr-ready")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			fmt.Println("echo:", scanner.Text())
		}
	case "env":
		fmt.Println(os.Getenv("AGMP_RUNTIME_OVERLAY"))
		fmt.Println(os.Getenv("PATH") != "")
	case "sleep":
		time.Sleep(2 * time.Second)
		fmt.Println("late")
	case "large":
		fmt.Print(strings.Repeat("x", 2048))
	}
}

func helperSpec(mode string) Spec {
	return Spec{
		Executable:  os.Args[0],
		Arguments:   []string{"-test.run=^TestRuntimeHelperProcess$"},
		Environment: []string{"AGMP_PLATFORM_RUNTIME_HELPER_V2=" + mode},
	}
}

func TestSessionSeparatesStdoutAndStderrAndSerializesCommands(t *testing.T) {
	var mu sync.Mutex
	var stdout, stderr []string
	s := NewSession(helperSpec("stream"), Hooks{OnOutput: func(line OutputLine) {
		mu.Lock()
		defer mu.Unlock()
		switch line.Source {
		case OutputStdout:
			stdout = append(stdout, line.Text)
		case OutputStderr:
			stderr = append(stderr, line.Text)
		}
	}})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ready := strings.Contains(strings.Join(stdout, "\n"), "ready")
		mu.Unlock()
		if ready {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := s.SendLine("hello"); err != nil {
		t.Fatal(err)
	}
	if !s.Wait(2 * time.Second) {
		t.Fatal("session did not exit")
	}
	mu.Lock()
	outText := strings.Join(stdout, "\n")
	errText := strings.Join(stderr, "\n")
	mu.Unlock()
	if !strings.Contains(outText, "echo: hello") {
		t.Fatalf("missing stdout echo: %q", outText)
	}
	if !strings.Contains(errText, "stderr-ready") {
		t.Fatalf("missing stderr source: %q", errText)
	}
	snapshot := s.Snapshot()
	if snapshot.State != StateExited || snapshot.PID <= 0 || snapshot.ExitCode == nil || *snapshot.ExitCode != 0 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}

func TestEnvironmentDefaultsToParentOverlay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	result, err := Run(ctx, RunSpec{Spec: Spec{
		Executable: os.Args[0],
		Arguments:  []string{"-test.run=^TestRuntimeHelperProcess$"},
		Environment: []string{
			"AGMP_PLATFORM_RUNTIME_HELPER_V2=env",
			"AGMP_RUNTIME_OVERLAY=ok",
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Fields(result.Stdout)
	if len(lines) < 2 || lines[0] != "ok" || lines[1] != "true" {
		t.Fatalf("environment overlay lost parent variables: %q", result.Stdout)
	}
}

func TestRunHonorsContextAndBoundsOutput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	result, err := Run(ctx, RunSpec{Spec: helperSpec("sleep")})
	if err == nil || !result.TimedOut {
		t.Fatalf("expected timeout, result=%+v err=%v", result, err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel2()
	result, err = Run(ctx2, RunSpec{Spec: helperSpec("large"), MaxOutputBytes: 128})
	if err == nil || !result.Truncated || len(result.Stdout) != 128 {
		t.Fatalf("expected bounded truncation, len=%d result=%+v err=%v", len(result.Stdout), result, err)
	}
}

func TestManagerOwnsNamedSessionLifecycle(t *testing.T) {
	manager := NewManager()
	session, err := manager.Start("test", helperSpec("stream"), Hooks{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start("test", helperSpec("stream"), Hooks{}); !errors.Is(err, ErrSessionExists) {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
	if err := session.SendLine("done"); err != nil {
		t.Fatal(err)
	}
	if !session.Wait(2 * time.Second) {
		t.Fatal("session did not exit")
	}
	if len(manager.Snapshots()) != 1 {
		t.Fatal("manager should retain completed session until explicit delete")
	}
	if err := manager.Delete("test"); err != nil {
		t.Fatal(err)
	}
	if _, ok := manager.Get("test"); ok {
		t.Fatal("session should be deleted")
	}
}

func TestManagerStopTerminatesLongRunningSession(t *testing.T) {
	manager := NewManager()
	session, err := manager.Start("sleep", helperSpec("sleep"), Hooks{})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Stop("sleep", 100*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !session.Wait(500 * time.Millisecond) {
		t.Fatal("stopped session was not reaped")
	}
	if state := session.Snapshot().State; state != StateExited {
		t.Fatalf("expected exited state after manager stop, got %s", state)
	}
}

func TestSessionHistoryAndSubscription(t *testing.T) {
	historyOnly := NewSession(Spec{HistoryLines: 2}, Hooks{})
	historyOnly.publishOutput(OutputLine{Source: OutputStdout, Text: "one"})
	historyOnly.publishOutput(OutputLine{Source: OutputStderr, Text: "two"})
	historyOnly.publishOutput(OutputLine{Source: OutputStdout, Text: "three"})
	history := historyOnly.History(0)
	if len(history) != 2 || history[0].Text != "two" || history[1].Text != "three" {
		t.Fatalf("expected newest two lines, got %#v", history)
	}
	if history[0].Sequence == 0 || history[1].Sequence <= history[0].Sequence {
		t.Fatalf("history sequence is not monotonic: %#v", history)
	}

	s := NewSession(helperSpec("stream"), Hooks{})
	stream, cancel := s.Subscribe(1)
	defer cancel()
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.SendLine("history"); err != nil {
		t.Fatal(err)
	}
	if !s.Wait(2 * time.Second) {
		t.Fatal("session did not exit")
	}
	// Subscribers are closed automatically after process exit; a slow consumer
	// must not keep the stdout/stderr readers or Wait() blocked.
	for range stream {
	}
}

func TestManagerConvenienceAPIsAndStopAll(t *testing.T) {
	manager := NewManager()
	streamSession, err := manager.Start("stream", helperSpec("stream"), Hooks{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := manager.Subscribe("missing", 1); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("missing subscription should fail, got %v", err)
	}
	if err := manager.SendLine("stream", "manager"); err != nil {
		t.Fatal(err)
	}
	if !streamSession.Wait(2 * time.Second) {
		t.Fatal("stream session did not exit")
	}
	if history, err := manager.History("stream", 10); err != nil || len(history) == 0 {
		t.Fatalf("manager history unavailable: len=%d err=%v", len(history), err)
	}

	if _, err := manager.Start("sleep-a", helperSpec("sleep"), Hooks{}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Start("sleep-b", helperSpec("sleep"), Hooks{}); err != nil {
		t.Fatal(err)
	}
	failures := manager.StopAll(100 * time.Millisecond)
	if len(failures) != 0 {
		t.Fatalf("StopAll failures: %#v", failures)
	}
	for _, snapshot := range manager.Snapshots() {
		if strings.HasPrefix(snapshot.ID, "sleep-") && snapshot.State != StateExited {
			t.Fatalf("session %s still active after StopAll: %+v", snapshot.ID, snapshot)
		}
	}
}
