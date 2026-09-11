package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceReadAndListStayInsideRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "b.txt"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Entries) != 2 || list.Entries[0].Name != "a.txt" || list.Entries[1].Name != "b.txt" {
		t.Fatalf("unexpected stable listing: %#v", list.Entries)
	}
	read, err := svc.Read("a.txt", 1024)
	if err != nil || read.Content != "hello" {
		t.Fatalf("read=%+v err=%v", read, err)
	}
	if _, err := svc.Read(filepath.Join(root, "..", "outside.txt"), 1024); err == nil || !errors.Is(err, ErrOutsideWorkspace) {
		// EvalSymlinks may fail first when outside.txt does not exist, so create a
		// real outside target and retry for the containment assertion.
		outside := filepath.Join(filepath.Dir(root), "outside.txt")
		if writeErr := os.WriteFile(outside, []byte("no"), 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		defer os.Remove(outside)
		if _, err = svc.Read(outside, 1024); !errors.Is(err, ErrOutsideWorkspace) {
			t.Fatalf("outside path should be rejected, got %v", err)
		}
	}
}

func TestSymlinkEscapeIsRejected(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	svc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Read(filepath.Join("escape", "secret.txt"), 1024); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("symlink escape should be rejected, got %v", err)
	}
}

func TestWorkspaceMutationToolsStayInsideRoot(t *testing.T) {
	root := t.TempDir()
	svc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Mkdir("configs", true); err != nil {
		t.Fatal(err)
	}
	write, err := svc.Write("configs/server.ini", "port=1\n", false)
	if err != nil {
		t.Fatal(err)
	}
	if !write.Created {
		t.Fatal("expected file creation")
	}
	replaced, err := svc.Replace("configs/server.ini", "port=1", "port=2", false)
	if err != nil || replaced.Replacements != 1 {
		t.Fatalf("replace=%+v err=%v", replaced, err)
	}
	read, err := svc.Read("configs/server.ini", 1024)
	if err != nil || read.Content != "port=2\n" {
		t.Fatalf("read=%+v err=%v", read, err)
	}
	stat, err := svc.Stat("configs/server.ini")
	if err != nil || !stat.Exists || stat.Directory {
		t.Fatalf("stat=%+v err=%v", stat, err)
	}
	if _, err := svc.Remove("configs/server.ini", false); err != nil {
		t.Fatal(err)
	}
	stat, err = svc.Stat("configs/server.ini")
	if err != nil || stat.Exists {
		t.Fatalf("expected removed file: stat=%+v err=%v", stat, err)
	}
}

func TestWorkspaceMutationRejectsMissingTargetThroughSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	svc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Write(filepath.Join("escape", "new.txt"), "no", true); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("missing target through symlink should be rejected, got %v", err)
	}
}

func TestWorkspaceCannotRemoveRoot(t *testing.T) {
	root := t.TempDir()
	svc, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Remove(".", true); !errors.Is(err, ErrWorkspaceRoot) {
		t.Fatalf("workspace root should be protected, got %v", err)
	}
}
