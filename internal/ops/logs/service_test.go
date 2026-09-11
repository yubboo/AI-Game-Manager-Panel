package loghub

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCatalogAggregatesMultipleSourcesAndRealLines(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "agmp", "agmp.log"), "a\nb\nc\n")
	mustWrite(t, filepath.Join(root, "operations", "2026-09-01.log"), `{"ts":"2026-09-01T01:02:03Z","level":"info","category":"operation","action":"start"}`+"\n")
	dstLog := filepath.Join(root, "games", "steam.dst", "Cluster_1-abcd", "Master", "session.log")
	mustWrite(t, dstLog, "[00:00:00]: [Steam] ready\n[00:00:01]: World ready\n")
	mustWriteMeta(t, strings.TrimSuffix(dstLog, ".log")+".json", dstSessionMeta{ID: "abc", ClusterName: "Cluster_1", ClusterPath: `D:\\Klei\\Cluster_1`, ShardName: "Master", Status: "stopped", StartedAt: 1, EndedAt: 2})

	s := New(root, Options{ExportSubdir: "exports"})
	page, err := s.Catalog(CatalogRequest{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if page.Summary.Files != 3 || page.Summary.Lines != 6 {
		t.Fatalf("summary = %+v", page.Summary)
	}
	if page.Summary.ActiveFiles != 1 { // 当前 AI Game Manager Panel Core 日志受保护。
		t.Fatalf("active = %d", page.Summary.ActiveFiles)
	}

	gamePage, err := s.Catalog(CatalogRequest{GameID: "steam.dst", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if gamePage.Total != 1 || gamePage.Items[0].Shard != "Master" || gamePage.Items[0].InstanceName != "Cluster_1" {
		t.Fatalf("game page = %+v", gamePage)
	}
}

func TestReadSupportsHeadTailAndStructuredFilter(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "games", "steam.dst", "Cluster_1", "Master", "session.log")
	mustWrite(t, path, "[00:00:00]: Starting\n[00:00:01]: [Steam] SteamGameServer_Init success\n[00:00:02]: [Warning] port warning\n[00:00:03]: World ready\n")
	mustWriteMeta(t, strings.TrimSuffix(path, ".log")+".json", dstSessionMeta{ID: "s1", ClusterName: "Cluster_1", ShardName: "Master", Status: "stopped"})
	s := New(root, Options{})
	catalog, _ := s.Catalog(CatalogRequest{GameID: "steam.dst"})
	id := catalog.Items[0].ID

	head, err := s.Read(ReadRequest{ID: id, Direction: "head", Limit: 2})
	if err != nil || len(head.Lines) != 2 || head.Lines[0].LineNumber != 1 || head.EOF {
		t.Fatalf("head=%+v err=%v", head, err)
	}
	next, err := s.Read(ReadRequest{ID: id, Direction: "next", Cursor: head.NextCursor, StartLine: head.NextLine, Limit: 2})
	if err != nil || len(next.Lines) != 2 || next.Lines[0].LineNumber != 3 {
		t.Fatalf("next=%+v err=%v", next, err)
	}
	tail, err := s.Read(ReadRequest{ID: id, Direction: "tail", Limit: 2})
	if err != nil || len(tail.Lines) != 2 || tail.Lines[1].LineNumber != 4 {
		t.Fatalf("tail=%+v err=%v", tail, err)
	}
	filtered, err := s.Read(ReadRequest{ID: id, Direction: "tail", Level: "warning", Limit: 10})
	if err != nil || len(filtered.Lines) != 1 || filtered.Lines[0].Level != "warning" {
		t.Fatalf("filtered=%+v err=%v", filtered, err)
	}
}

func TestDeleteProtectsActiveAndManualDeletionIsReflected(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "agmp", "agmp.log"), "running\n")
	history := filepath.Join(root, "steam", "old.log")
	mustWrite(t, history, "old\n")
	s := New(root, Options{})
	page, _ := s.Catalog(CatalogRequest{Limit: 100})
	var coreID, oldID string
	for _, item := range page.Items {
		if item.Source == "agmp" {
			coreID = item.ID
		} else if item.RelativePath == "steam/old.log" {
			oldID = item.ID
		}
	}
	if _, err := s.DeleteOne(coreID); !errors.Is(err, ErrLogActive) {
		t.Fatalf("expected ErrLogActive, got %v", err)
	}
	result, err := s.DeleteOne(oldID)
	if err != nil || result.Deleted != 1 {
		t.Fatalf("delete=%+v err=%v", result, err)
	}
	manual := filepath.Join(root, "steam", "manual.log")
	mustWrite(t, manual, "manual\n")
	before, _ := s.Catalog(CatalogRequest{Limit: 100})
	if err := os.Remove(manual); err != nil {
		t.Fatal(err)
	}
	after, _ := s.Catalog(CatalogRequest{Limit: 100})
	if after.Total != before.Total-1 {
		t.Fatalf("manual deletion not reflected: before=%d after=%d", before.Total, after.Total)
	}
}

func TestExportFilteredAndRecorder(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "steam", "steam.log"), "one\ntwo\n")
	s := New(root, Options{ExportSubdir: "exports", OperationDir: "operations", OperationAudit: true, FlushIntervalMS: 10})
	s.RecordOperation(OperationRecord{Source: "dst", Action: "start_cluster", Target: "Cluster_1", Result: "success", GameID: "steam.dst"})
	time.Sleep(30 * time.Millisecond)
	s.Close()

	catalog, err := s.Catalog(CatalogRequest{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Total < 2 {
		t.Fatalf("expected recorder log, catalog=%+v", catalog)
	}
	exported, err := s.ExportFiltered(CatalogRequest{Source: "steam"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(exported.Path, filepath.Join(root, "exports")) {
		t.Fatalf("export path escaped log root: %s", exported.Path)
	}
	archive, err := zip.OpenReader(exported.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) < 2 { // 日志 + manifest.json
		t.Fatalf("archive files=%d", len(archive.File))
	}

	// exports 目录不应再次出现在日志目录中，避免导出包污染真实日志数量。
	after, _ := s.Catalog(CatalogRequest{Limit: 100})
	if after.Total != catalog.Total {
		t.Fatalf("export polluted catalog: before=%d after=%d", catalog.Total, after.Total)
	}
}

func TestLineCountCacheHandlesGrowthExactly(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "steam", "growth.log")
	mustWrite(t, path, "one\ntwo\n")
	s := New(root, Options{})
	first, _ := s.Catalog(CatalogRequest{Source: "steam"})
	if first.Summary.Lines != 2 {
		t.Fatalf("first lines=%d", first.Summary.Lines)
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.WriteString("three\nfour\n")
	_ = file.Close()
	second, _ := s.Catalog(CatalogRequest{Source: "steam"})
	if second.Summary.Lines != 4 {
		t.Fatalf("second lines=%d", second.Summary.Lines)
	}

	// 未以换行结束的文件增长时会自动回退到完整重数，避免增量算法重复计数最后一行。
	mustWrite(t, path, "one\ntwo")
	third, _ := s.Catalog(CatalogRequest{Source: "steam"})
	if third.Summary.Lines != 2 {
		t.Fatalf("third lines=%d", third.Summary.Lines)
	}
	file, _ = os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = file.WriteString("\nthree\n")
	_ = file.Close()
	fourth, _ := s.Catalog(CatalogRequest{Source: "steam"})
	if fourth.Summary.Lines != 3 {
		t.Fatalf("fourth lines=%d", fourth.Summary.Lines)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustWriteMeta(t *testing.T, path string, value dstSessionMeta) {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, path, string(payload))
}

func TestConfiguredNestedDirectoriesAreClassified(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "system", "core", "agmp.log"), "system ready\n")
	mustWrite(t, filepath.Join(root, "audit-trail", "security.log"), "security denied\n")
	mustWrite(t, filepath.Join(root, "game-logs", "steam.dst", "Cluster_2", "Caves", "session.log"), "World ready\n")

	s := New(root, Options{
		CoreDir:  filepath.Join("system", "core"),
		AuditDir: "audit-trail",
		GamesDir: "game-logs",
	})
	page, err := s.Catalog(CatalogRequest{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 3 {
		t.Fatalf("total=%d items=%+v", page.Total, page.Items)
	}
	seen := map[string]LogFile{}
	for _, item := range page.Items {
		seen[item.Source] = item
	}
	if seen["agmp"].Kind != "system" {
		t.Fatalf("configured core not classified: %+v", seen["agmp"])
	}
	if seen["audit"].Kind != "audit" {
		t.Fatalf("configured audit not classified: %+v", seen["audit"])
	}
	if seen["dst"].GameID != "steam.dst" || seen["dst"].Shard != "Caves" {
		t.Fatalf("configured games not classified: %+v", seen["dst"])
	}
}

func TestClassifyLineSupportsAuditAndSecurityCategories(t *testing.T) {
	audit := classifyLine(1, `{"ts":"2026-09-09T00:00:00Z","level":"info","category":"audit"}`)
	if audit.Category != "audit" {
		t.Fatalf("audit category=%s", audit.Category)
	}
	security := classifyLine(2, "security request denied")
	if security.Category != "security" {
		t.Fatalf("security category=%s", security.Category)
	}
}
