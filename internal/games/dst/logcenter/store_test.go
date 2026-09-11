package logcenter

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	dstruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"
)

func TestPersistentLogSessionReadSearchDiagnostics(t *testing.T) {
	store := New(t.TempDir())
	clusterPath := filepath.Join(t.TempDir(), "Cluster_1")
	sink, err := store.StartSession(dstruntime.StartRequest{
		ClusterName: "Cluster_1",
		ClusterPath: clusterPath,
		ShardName:   "Master",
		Spec:        dedicated.LaunchSpec{Role: dedicated.ShardMaster},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 1500; i++ {
		text := fmt.Sprintf("[%02d:00:00]: normal line %d", i%60, i)
		if i == 100 {
			text = "[00:00:02]: ERROR: Failed to run code from modoverrides.lua"
		}
		if i == 200 {
			text = "[00:00:07]: Token retrieved from: APP:Klei//DoNotStarveTogether/Cluster_1/cluster_token.txt"
		}
		if i == 300 {
			text = "[00:00:00]: [Steam] SteamGameServer_Init success"
		}
		if i == 400 {
			text = "[00:00:07]: [Warning] Could not confirm port 11001 is open in the firewall."
		}
		if i == 500 {
			text = "[00:00:31]: Sim paused"
		}
		if i == 600 {
			text = "[00:00:33]: Server registered via geo DNS in eu-central-1"
		}
		if i == 700 {
			text = "[00:00:00]: [WARNING] -console has been deprecated: Use the [MISC] / console_enabled setting instead."
		}
		sink.Append(dstruntime.LogLine{Sequence: uint64(i), Timestamp: time.Now().Unix(), Text: text})
	}
	exit := 0
	sink.Close(dstruntime.ProcessSnapshot{ClusterName: "Cluster_1", ClusterPath: clusterPath, ShardName: "Master", Status: dedicated.StatusStopped, WorldReady: true, ExitCode: &exit, UpdatedAt: time.Now().Unix()})

	sessions, err := store.List(ListRequest{ClusterPath: clusterPath, ShardName: "Master", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].LineCount != 1500 {
		t.Fatalf("expected 1500 lines, got %d", sessions[0].LineCount)
	}
	if sessions[0].DroppedLines != 0 {
		t.Fatalf("unexpected dropped lines: %d", sessions[0].DroppedLines)
	}

	page, err := store.Read(ReadRequest{SessionID: sessions[0].ID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Lines) != 100 || page.Lines[0].LineNumber != 1 || page.NextLine != 101 {
		t.Fatalf("unexpected first page: %+v", page)
	}
	page2, err := store.Read(ReadRequest{SessionID: sessions[0].ID, Cursor: page.NextCursor, StartLine: page.NextLine, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Lines) != 100 || page2.Lines[0].LineNumber != 101 {
		t.Fatalf("unexpected second page")
	}

	tail, err := store.Tail(TailRequest{SessionID: sessions[0].ID, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(tail.Lines) != 50 || tail.Lines[0].LineNumber != 1451 {
		t.Fatalf("unexpected tail start: %+v", tail.Lines[0])
	}

	search, err := store.Search(SearchRequest{SessionID: sessions[0].ID, Query: "modoverrides", Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(search.Matches) != 1 || search.Matches[0].Level != "error" || search.Matches[0].Category != "mod" {
		t.Fatalf("unexpected search result: %+v", search)
	}

	diag, err := store.Diagnostics(sessions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if !diag.SteamReady || !diag.TokenLoaded || !diag.WorldReady || !diag.Registered {
		t.Fatalf("expected readiness markers: %+v", diag)
	}
	if diag.ErrorCount < 1 || diag.WarningCount < 1 {
		t.Fatalf("expected error and warning counts: %+v", diag)
	}
	if len(diag.Issues) < 2 {
		t.Fatalf("expected diagnostic issues, got %+v", diag.Issues)
	}
}

func TestAppendDoesNotBlockWhenPersistenceQueueIsFull(t *testing.T) {
	w := &sessionWriter{lines: make(chan dstruntime.LogLine, 1)}
	w.lines <- dstruntime.LogLine{Sequence: 1, Text: "occupied"}
	started := time.Now()
	w.Append(dstruntime.LogLine{Sequence: 2, Text: "must not block"})
	if elapsed := time.Since(started); elapsed > 50*time.Millisecond {
		t.Fatalf("Append blocked for %s", elapsed)
	}
	if w.dropped.Load() != 1 {
		t.Fatalf("expected dropped counter to increment")
	}
}

func TestActiveSessionReadFlushesBufferedWriter(t *testing.T) {
	store := New(t.TempDir())
	clusterPath := filepath.Join(t.TempDir(), "Cluster_Active")
	sink, err := store.StartSession(dstruntime.StartRequest{
		ClusterName: "Cluster_Active",
		ClusterPath: clusterPath,
		ShardName:   "Master",
		Spec:        dedicated.LaunchSpec{Role: dedicated.ShardMaster},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 12; i++ {
		sink.Append(dstruntime.LogLine{Sequence: uint64(i), Timestamp: time.Now().Unix(), Text: fmt.Sprintf("active line %d", i)})
	}

	page, err := store.Read(ReadRequest{SessionID: sink.ID(), Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Lines) != 12 {
		t.Fatalf("active read should flush buffered writer; got %d lines", len(page.Lines))
	}
	if page.Lines[11].Text != "active line 12" {
		t.Fatalf("unexpected last active line: %+v", page.Lines[11])
	}

	exit := 0
	sink.Close(dstruntime.ProcessSnapshot{
		ClusterName: "Cluster_Active",
		ClusterPath: clusterPath,
		ShardName:   "Master",
		Status:      dedicated.StatusStopped,
		ExitCode:    &exit,
		UpdatedAt:   time.Now().Unix(),
	})
}

func TestDiagnosticsResolveFirewallHintAfterMasterBecomesReady(t *testing.T) {
	store := New(t.TempDir())
	clusterPath := filepath.Join(t.TempDir(), "Cluster_Master_Healthy")
	sink, err := store.StartSession(dstruntime.StartRequest{
		ClusterName: "Cluster_Master_Healthy",
		ClusterPath: clusterPath,
		ShardName:   "Master",
		Spec:        dedicated.LaunchSpec{Role: dedicated.ShardMaster},
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := []string{
		"[00:00:00]: [Steam] SteamGameServer_Init success",
		"[00:00:00]: Token retrieved from: APP:Klei//DoNotStarveTogether/Cluster_Master_Healthy/cluster_token.txt",
		"[00:00:07]: [Warning] Could not confirm port 10999 is open in the firewall.",
		"[00:00:07]: Online Server Started on port: 10999",
		"[00:00:27]: [Shard] Shard server started on port: 10888",
		"[00:00:28]: Sim paused",
		"[00:00:30]: Server registered via geo DNS in eu-central-1",
		"[00:00:33]: [Shard] Secondary Caves(1838182312) ready!",
	}
	for i, text := range lines {
		sink.Append(dstruntime.LogLine{Sequence: uint64(i + 1), Timestamp: time.Now().Unix(), Text: text})
	}

	diag, err := store.Diagnostics(sink.ID())
	if err != nil {
		t.Fatal(err)
	}
	if !diag.Healthy || !diag.NetworkReady || !diag.Registered || !diag.WorldReady {
		t.Fatalf("expected healthy master diagnostics, got %+v", diag)
	}
	if diag.WarningCount != 0 {
		t.Fatalf("resolved firewall hint must not remain a warning: %+v", diag)
	}
	for _, issue := range diag.Issues {
		if issue.Code == "firewall" {
			t.Fatalf("resolved firewall hint must not remain an issue: %+v", issue)
		}
	}

	exit := 0
	sink.Close(dstruntime.ProcessSnapshot{ClusterName: "Cluster_Master_Healthy", ClusterPath: clusterPath, ShardName: "Master", Status: dedicated.StatusStopped, WorldReady: true, ExitCode: &exit, UpdatedAt: time.Now().Unix()})
}

func TestDiagnosticsTreatCavesShardLinkAsHealthyWithoutGeoRegistrationMarker(t *testing.T) {
	store := New(t.TempDir())
	clusterPath := filepath.Join(t.TempDir(), "Cluster_Caves_Healthy")
	sink, err := store.StartSession(dstruntime.StartRequest{
		ClusterName: "Cluster_Caves_Healthy",
		ClusterPath: clusterPath,
		ShardName:   "Caves",
		Spec:        dedicated.LaunchSpec{Role: dedicated.ShardSecondary},
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := []string{
		"[00:00:00]: [Steam] SteamGameServer_Init success",
		"[00:00:00]: Token retrieved from: APP:Klei//DoNotStarveTogether/Cluster_Caves_Healthy/cluster_token.txt",
		"[00:00:07]: [Warning] Could not confirm port 11000 is open in the firewall.",
		"[00:00:07]: Online Server Started on port: 11000",
		"[00:00:28]: [Shard] Connecting to master...",
		"[00:00:33]: [Shard] secondary shard is now ready!",
		"[00:00:33]: World 1(Master) is now connected",
		"[00:00:34]: [Shard] secondary shard LUA is now ready!",
		"[00:00:34]: Sim paused",
	}
	for i, text := range lines {
		sink.Append(dstruntime.LogLine{Sequence: uint64(i + 1), Timestamp: time.Now().Unix(), Text: text})
	}

	diag, err := store.Diagnostics(sink.ID())
	if err != nil {
		t.Fatal(err)
	}
	if !diag.Healthy || !diag.NetworkReady || !diag.WorldReady || !diag.TokenLoaded || !diag.SteamReady {
		t.Fatalf("expected healthy caves diagnostics, got %+v", diag)
	}
	if diag.Registered {
		t.Fatalf("caves does not need a geo registration marker to be healthy: %+v", diag)
	}
	if diag.WarningCount != 0 {
		t.Fatalf("resolved caves firewall hint must not remain a warning: %+v", diag)
	}
	for _, issue := range diag.Issues {
		if issue.Code == "firewall" {
			t.Fatalf("resolved caves firewall hint must not remain an issue: %+v", issue)
		}
	}

	exit := 0
	sink.Close(dstruntime.ProcessSnapshot{ClusterName: "Cluster_Caves_Healthy", ClusterPath: clusterPath, ShardName: "Caves", Status: dedicated.StatusStopped, WorldReady: true, ExitCode: &exit, UpdatedAt: time.Now().Unix()})
}

func TestExportUsesConfiguredProjectLogDirectory(t *testing.T) {
	root := t.TempDir()
	exportRoot := filepath.Join(root, "project", "log")
	store := New(filepath.Join(root, "store"), exportRoot)
	clusterPath := filepath.Join(root, "Cluster_Export")
	sink, err := store.StartSession(dstruntime.StartRequest{
		ClusterName: "Cluster_Export",
		ClusterPath: clusterPath,
		ShardName:   "Master",
		Spec:        dedicated.LaunchSpec{Role: dedicated.ShardMaster},
	})
	if err != nil {
		t.Fatal(err)
	}
	sink.Append(dstruntime.LogLine{Sequence: 1, Timestamp: time.Now().Unix(), Text: "[00:00:01]: Sim paused"})
	exit := 0
	sink.Close(dstruntime.ProcessSnapshot{ClusterName: "Cluster_Export", ClusterPath: clusterPath, ShardName: "Master", Status: dedicated.StatusStopped, WorldReady: true, ExitCode: &exit, UpdatedAt: time.Now().Unix()})

	result, err := store.Export(sink.ID())
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(filepath.Dir(result.Path)) != filepath.Clean(exportRoot) {
		t.Fatalf("expected export in project log directory %q, got %q", exportRoot, result.Path)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("exported log missing: %v", err)
	}
}
