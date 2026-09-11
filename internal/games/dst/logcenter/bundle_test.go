package logcenter

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBundleCollectsLogsAndSafeConfigsWithoutSensitiveFiles(t *testing.T) {
	root := t.TempDir()
	cluster := filepath.Join(root, "Cluster_1")
	master := filepath.Join(cluster, "Master")
	backup := filepath.Join(master, "backup", "server_log")
	if err := os.MkdirAll(backup, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite := func(path, value string) {
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite(filepath.Join(master, "server_log.txt"), "Loading mod: workshop-123456 (Example Mod)\n")
	mustWrite(filepath.Join(backup, "server_log_1.txt"), "LUA ERROR stack traceback\n")
	mustWrite(filepath.Join(master, "server.ini"), "[NETWORK]\nserver_port=11001\n")
	mustWrite(filepath.Join(master, "modoverrides.lua"), "return {}\n")
	mustWrite(filepath.Join(master, "leveldataoverride.lua"), "return {}\n")
	mustWrite(filepath.Join(cluster, "cluster_token.txt"), "SUPER-SECRET\n")
	mustWrite(filepath.Join(cluster, "adminlist.txt"), "KU_SECRET\n")

	exportRoot := filepath.Join(root, "project", "log")
	store := New(filepath.Join(root, "store"), exportRoot)
	result, err := store.Bundle(BundleRequest{ClusterPath: cluster, ShardNames: []string{"Master"}})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(filepath.Dir(result.Path)) != filepath.Clean(exportRoot) {
		t.Fatalf("bundle should be exported to project log directory: %s", result.Path)
	}
	reader, err := zip.OpenReader(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	joined := strings.Join(names, "\n")
	for _, expected := range []string{"Master/server_log.txt", "Master/server.ini", "Master/modoverrides.lua", "Master/leveldataoverride.lua", "manifest.txt", "mod_list.txt"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %s in %s", expected, joined)
		}
	}
	if strings.Contains(joined, "cluster_token") || strings.Contains(joined, "adminlist") {
		t.Fatalf("sensitive files leaked into bundle: %s", joined)
	}
}

func TestStreamBundleDoesNotRequireIntermediateDownloadFile(t *testing.T) {
	root := t.TempDir()
	cluster := filepath.Join(root, "Cluster_Stream")
	master := filepath.Join(cluster, "Master")
	if err := os.MkdirAll(master, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(master, "server_log.txt"), []byte("[00:00:01]: Sim paused\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cluster, "cluster_token.txt"), []byte("NEVER-STREAM-ME"), 0o644); err != nil {
		t.Fatal(err)
	}

	store := New(filepath.Join(root, "store"))
	var output bytes.Buffer
	name, err := store.StreamBundle(BundleRequest{ClusterPath: cluster, ShardNames: []string{"Master"}}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.ToLower(name), ".zip") || output.Len() == 0 {
		t.Fatalf("unexpected stream bundle result: name=%q bytes=%d", name, output.Len())
	}
	reader, err := zip.NewReader(bytes.NewReader(output.Bytes()), int64(output.Len()))
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	joined := strings.Join(names, "\n")
	if !strings.Contains(joined, "Master/server_log.txt") {
		t.Fatalf("stream bundle missing server log: %s", joined)
	}
	if strings.Contains(joined, "cluster_token") {
		t.Fatalf("stream bundle leaked cluster token: %s", joined)
	}
}
