package instance

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStorePersistsSharedGameInstance(t *testing.T) {
	path := filepath.Join(t.TempDir(), "instances.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Upsert(Instance{ID: "inst_mc", Name: "生存服", GameID: "minecraft.java", Origin: OriginAgent, RuntimeState: "stopped", GameVersion: "1.21.1", ServerType: "paper", Port: 25565, Managed: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.CreatedAt == 0 || got.UpdatedAt == 0 {
		t.Fatal("timestamps not populated")
	}
	reopened, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	items := reopened.List()
	if len(items) != 1 || items[0].GameVersion != "1.21.1" || items[0].Origin != OriginAgent {
		t.Fatalf("unexpected reload: %#v", items)
	}
}

func TestStoreRollsBackMemoryWhenPersistenceFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state", "instances.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Dir(path), []byte("not-a-directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Upsert(Instance{ID: "inst_fail", Name: "must rollback", GameID: "minecraft.java"}); err == nil {
		t.Fatal("expected persistence failure")
	}
	if _, err := store.Get("inst_fail"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("failed upsert leaked into in-memory state: %v", err)
	}
}

func TestStoreDeleteRestoresMemoryWhenPersistenceFails(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "state", "instances.json")
	store, err := NewStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Upsert(Instance{ID: "inst_keep", Name: "keep", GameID: "minecraft.java"}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Dir(path)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Dir(path), []byte("not-a-directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("inst_keep"); err == nil {
		t.Fatal("expected persistence failure")
	}
	if got, err := store.Get("inst_keep"); err != nil || got.ID != "inst_keep" {
		t.Fatalf("failed delete did not restore in-memory state: got=%#v err=%v", got, err)
	}
}
