package dedicated

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureConsoleEnabledAddsMisc(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cluster.ini")
	if err := os.WriteFile(path, []byte("[GAMEPLAY]\ngame_mode = survival\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureConsoleEnabled(path); err != nil {
		t.Fatal(err)
	}
	if !ConsoleEnabled(path) {
		t.Fatal("console should be enabled")
	}
	body, _ := os.ReadFile(path)
	if !strings.Contains(string(body), "[MISC]\nconsole_enabled = true") {
		t.Fatalf("unexpected file: %s", body)
	}
}

func TestEnsureConsoleEnabledReplacesFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cluster.ini")
	if err := os.WriteFile(path, []byte("[MISC]\r\nconsole_enabled = false\r\n[NETWORK]\r\ncluster_name = Test\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureConsoleEnabled(path); err != nil {
		t.Fatal(err)
	}
	if !ConsoleEnabled(path) {
		t.Fatal("console should be enabled")
	}
	body, _ := os.ReadFile(path)
	if strings.Contains(string(body), "console_enabled = false") {
		t.Fatal("old false value remained")
	}
}
