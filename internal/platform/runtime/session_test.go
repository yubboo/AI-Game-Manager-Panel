package platformruntime

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSessionHelperProcess(t *testing.T) {
	if os.Getenv("AGMP_PLATFORM_RUNTIME_HELPER") != "1" {
		return
	}
	fmt.Println("ready")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		fmt.Println("echo:", scanner.Text())
	}
}

func TestSessionStreamsCommandAndWaitsForRealExit(t *testing.T) {
	var mu sync.Mutex
	lines := []string{}
	exited := make(chan ExitResult, 1)
	s := NewSession(Spec{
		Executable:  os.Args[0],
		Arguments:   []string{"-test.run=^TestSessionHelperProcess$"},
		Environment: append(os.Environ(), "AGMP_PLATFORM_RUNTIME_HELPER=1"),
	}, Hooks{
		OnLine: func(line string) { mu.Lock(); lines = append(lines, line); mu.Unlock() },
		OnExit: func(result ExitResult) { exited <- result },
	})
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		ready := strings.Contains(strings.Join(lines, "\n"), "ready")
		mu.Unlock()
		if ready {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if s.PID() <= 0 {
		t.Fatal("expected process PID")
	}
	if err := s.SendLine("hello"); err != nil {
		t.Fatal(err)
	}
	if !s.Wait(2 * time.Second) {
		t.Fatal("session did not observe process exit")
	}
	result := <-exited
	if result.ExitCode != 0 {
		t.Fatalf("exit=%d err=%v", result.ExitCode, result.Err)
	}
	mu.Lock()
	text := strings.Join(lines, "\n")
	mu.Unlock()
	if !strings.Contains(text, "echo: hello") {
		t.Fatalf("missing echoed command: %s", text)
	}
}
