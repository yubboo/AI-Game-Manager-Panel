package control

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPendingApprovalWaitsUntilExplicitDecisionAndIsSingleUse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "approval-state.json")
	store := New(path, ModeAsk)
	hash, err := Fingerprint(KindTool, "files.write", map[string]any{"path": "demo.txt", "content": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	item, err := store.CreatePending(KindTool, "files.write", "modify", "请求修改文件", hash)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeApproval(item.ID, hash); !errors.Is(err, ErrApprovalPending) {
		t.Fatalf("pending approval should wait, got %v", err)
	}
	if _, err := store.Resolve(item.ID, true, "owner"); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeApproval(item.ID, hash); err != nil {
		t.Fatalf("approved request should execute once: %v", err)
	}
	if err := store.ConsumeApproval(item.ID, hash); !errors.Is(err, ErrApprovalConsumed) {
		t.Fatalf("approval must not be replayable, got %v", err)
	}
}

func TestApprovalCannotAuthorizeDifferentPayload(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "approval-state.json"), ModeRisk)
	hashA, _ := Fingerprint(KindCommand, "manual.shell", map[string]any{"command": "echo a"})
	hashB, _ := Fingerprint(KindCommand, "manual.shell", map[string]any{"command": "echo b"})
	item, err := store.CreatePending(KindCommand, "manual.shell", "system", "请求执行手动终端命令", hashA)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Resolve(item.ID, true, "owner"); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeApproval(item.ID, hashB); !errors.Is(err, ErrApprovalMismatch) {
		t.Fatalf("approval must be bound to exact payload, got %v", err)
	}
}

func TestModePersistsWithoutStoringPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "approval-state.json")
	store := New(path, ModeAsk)
	if _, err := store.SetMode(ModeFull, "owner"); err != nil {
		t.Fatal(err)
	}
	reloaded := New(path, ModeAsk)
	if got := reloaded.Mode(); got != ModeFull {
		t.Fatalf("expected persisted full mode, got %s", got)
	}
}

func TestApprovalFileDoesNotStoreRawPayload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "approval-state.json")
	store := New(path, ModeAsk)
	secretCommand := "echo SUPER_SECRET_VALUE"
	hash, err := Fingerprint(KindCommand, "manual.shell", map[string]string{"command": secretCommand})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePending(KindCommand, "manual.shell", "system", "请求执行受控命令", hash); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secretCommand) || strings.Contains(string(raw), "SUPER_SECRET_VALUE") {
		t.Fatal("approval state must not persist raw command payload")
	}
}

func TestRejectedApprovalNeverExecutes(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "approval-state.json"), ModeAsk)
	hash, _ := Fingerprint(KindTool, "files.delete", map[string]string{"path": "demo.txt"})
	item, err := store.CreatePending(KindTool, "files.delete", "destructive", "请求删除文件", hash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Resolve(item.ID, false, "owner"); err != nil {
		t.Fatal(err)
	}
	if err := store.ConsumeApproval(item.ID, hash); !errors.Is(err, ErrApprovalRejected) {
		t.Fatalf("rejected approval must stay blocked, got %v", err)
	}
}
