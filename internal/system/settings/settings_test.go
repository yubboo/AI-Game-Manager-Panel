package settings

import (
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := NewStore(path)

	initial, err := store.Load()
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if initial.Theme != "dark" {
		t.Fatalf("unexpected default theme: %s", initial.Theme)
	}

	want := Settings{Theme: "light", Language: "zh-CN", Debug: true}
	if err := store.Save(want); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if got != want {
		t.Fatalf("round trip mismatch: got %#v want %#v", got, want)
	}
}
