package runtime

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
)

func TestRuntimeHelperProcess(t *testing.T) {
	if os.Getenv("AGMP_DST_RUNTIME_HELPER") != "1" {
		return
	}
	fmt.Println("About to start a shard with these settings:")
	fmt.Println("Reset() returning")
	fmt.Println("Sim paused")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println("COMMAND:", line)
		if shutdownCommandRE.MatchString(line) {
			return
		}
	}
}

func TestProcessLifecycleReadyCommandAndGracefulStop(t *testing.T) {
	t.Setenv("AGMP_DST_RUNTIME_HELPER", "1")
	process := NewProcess(StartRequest{
		ClusterName: "Cluster_1",
		ClusterPath: filepath.Join(t.TempDir(), "Cluster_1"),
		ShardName:   "Master",
		Spec: dedicated.LaunchSpec{
			ClusterName:      "Cluster_1",
			ShardName:        "Master",
			Role:             dedicated.ShardMaster,
			Executable:       os.Args[0],
			WorkingDirectory: t.TempDir(),
			Arguments:        []string{"-test.run=^TestRuntimeHelperProcess$"},
		},
	}, nil, nil)
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if process.Snapshot().WorldReady {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	snapshot := process.Snapshot()
	if !snapshot.WorldReady || snapshot.Status != dedicated.StatusRunning || snapshot.PID == 0 {
		t.Fatalf("unexpected running snapshot: %+v", snapshot)
	}

	if err := process.SendCommand(`c_announce("c_shutdown()")`); err != nil {
		t.Fatal(err)
	}
	if process.Snapshot().IntentionalShutdown {
		t.Fatal("shutdown text inside another command must not mark expected exit")
	}

	process.StopBlocking(StopOptions{GracefulTimeout: 2 * time.Second, TermTimeout: 500 * time.Millisecond})
	snapshot = process.Snapshot()
	if snapshot.Status != dedicated.StatusStopped || !snapshot.IntentionalShutdown {
		t.Fatalf("expected graceful stopped state: %+v", snapshot)
	}
	if snapshot.ExitCode == nil || *snapshot.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %+v", snapshot.ExitCode)
	}

	logs := process.ReadLogs(0, 100)
	joined := make([]string, 0, len(logs.Lines))
	for _, line := range logs.Lines {
		joined = append(joined, line.Text)
	}
	text := strings.Join(joined, "\n")
	if !strings.Contains(text, "Sim paused") || !strings.Contains(text, "COMMAND: c_shutdown()") {
		t.Fatalf("expected ready/shutdown output, got:\n%s", text)
	}
}

func TestReadLogsCursorAndDropSignal(t *testing.T) {
	process := NewProcess(StartRequest{Spec: dedicated.LaunchSpec{Role: dedicated.ShardMaster}}, nil, nil)
	for i := 0; i < StreamLogLimit+5; i++ {
		process.consumeLine(fmt.Sprintf("line-%d", i))
	}
	batch := process.ReadLogs(0, 3)
	if !batch.Dropped {
		t.Fatal("expected dropped signal when caller cursor predates ring buffer")
	}
	if len(batch.Lines) != 3 {
		t.Fatalf("expected bounded batch, got %d", len(batch.Lines))
	}
	next := process.ReadLogs(batch.NextCursor, 10)
	if next.Dropped {
		t.Fatal("cursor continuation should not be marked dropped")
	}
	if len(next.Lines) == 0 {
		t.Fatal("expected continuation lines")
	}
}

func TestRecentLogLinesStayBounded(t *testing.T) {
	process := NewProcess(StartRequest{Spec: dedicated.LaunchSpec{Role: dedicated.ShardMaster}}, nil, nil)
	for i := 0; i < RecentLogLimit+20; i++ {
		process.consumeLine(fmt.Sprintf("line-%d", i))
	}
	lines := process.RecentLogLines()
	if len(lines) != RecentLogLimit {
		t.Fatalf("recent log count=%d want=%d", len(lines), RecentLogLimit)
	}
	if lines[0] != "line-20" {
		t.Fatalf("unexpected first retained line: %q", lines[0])
	}
}

func TestShutdownCommandPatternMatchesPythonBaseline(t *testing.T) {
	accepted := []string{"c_shutdown()", " c_shutdown(1); ", "c_shutdown(false)", "c_shutdown(true);"}
	for _, value := range accepted {
		if !shutdownCommandRE.MatchString(value) {
			t.Fatalf("expected shutdown command: %q", value)
		}
	}
	rejected := []string{`c_announce("c_shutdown()")`, "print('c_shutdown()')", "c_shutdown(2)"}
	for _, value := range rejected {
		if shutdownCommandRE.MatchString(value) {
			t.Fatalf("must not classify as shutdown: %q", value)
		}
	}
}

func TestProcessKeyUsesClusterPathAndShard(t *testing.T) {
	left := processKey(filepath.Join("root", "Cluster_1"), "Master")
	right := processKey(filepath.Join("other", "Cluster_1"), "Master")
	if left == right {
		t.Fatal("same cluster name in different directories must not collide")
	}
}

func TestStopBlockingNeverReportsStoppedWithoutObservedProcessExit(t *testing.T) {
	process := NewProcess(StartRequest{Spec: dedicated.LaunchSpec{Role: dedicated.ShardMaster}}, nil, nil)
	process.mu.Lock()
	process.status = dedicated.StatusRunning
	process.mu.Unlock()

	process.StopBlocking(StopOptions{GracefulTimeout: time.Millisecond, TermTimeout: time.Millisecond})
	snapshot := process.Snapshot()
	if snapshot.Status == dedicated.StatusStopped {
		t.Fatalf("must not report stopped without cmd.Wait completion: %+v", snapshot)
	}
	if snapshot.Status != dedicated.StatusStopping {
		t.Fatalf("expected unresolved shutdown to remain stopping, got %+v", snapshot)
	}
	if !strings.Contains(snapshot.Error, "端口可能仍被占用") {
		t.Fatalf("expected stale-port warning, got %q", snapshot.Error)
	}
}

func TestManagedPIDsExcludeHistoricalStoppedProcess(t *testing.T) {
	manager := NewManager()
	process := NewProcess(StartRequest{ClusterPath: "Cluster_1", ShardName: "Master", Spec: dedicated.LaunchSpec{Role: dedicated.ShardMaster}}, nil, nil)
	process.mu.Lock()
	process.status = dedicated.StatusStopped
	// Historical exec.Cmd/PID reuse is hard to synthesize portably here; the
	// status gate itself is the safety invariant. A stopped process must never
	// enter the active managed set even if its snapshot still carries metadata.
	process.mu.Unlock()
	manager.procs[processKey("Cluster_1", "Master")] = process
	if got := manager.ManagedPIDs(); len(got) != 0 {
		t.Fatalf("historical stopped process must not be considered managed: %#v", got)
	}
}
