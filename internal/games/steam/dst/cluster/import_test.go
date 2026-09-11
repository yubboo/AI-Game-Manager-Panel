package cluster

import (
	"os"
	"path/filepath"
	"testing"
)

func TestImportCopiesWorldButSkipsToken(t *testing.T) {
	source := filepath.Join(t.TempDir(), "Cluster_1")
	mustWrite(t, filepath.Join(source, "cluster.ini"), "[GAMEPLAY]\n")
	mustWrite(t, filepath.Join(source, "Master", "server.ini"), "[SHARD]\n")
	mustWrite(t, filepath.Join(source, "Master", "save", "session", "0001"), "world")
	mustWrite(t, filepath.Join(source, "cluster_token.txt"), "SECRET")
	root := filepath.Join(t.TempDir(), "DoNotStarveTogether")

	result, err := Import(root, ImportRequest{SourcePath: source, TargetName: "Imported_1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.TokenSkipped {
		t.Fatal("expected token to be skipped")
	}
	if _, err := os.Stat(filepath.Join(result.Path, "cluster_token.txt")); !os.IsNotExist(err) {
		t.Fatalf("token should not be copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Path, "Master", "save", "session", "0001")); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}
