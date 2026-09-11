package preferences

import (
	"path/filepath"
	"testing"
)

func TestStoreCanSaveRepeatedly(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "dst.json"))
	if err := store.Save(Value{DedicatedServerPath: "first", DedicatedServerExtraArgs: "-a 1"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(Value{DedicatedServerPath: "second", DedicatedServerExtraArgs: "-b 2"}); err != nil {
		t.Fatal(err)
	}
	got := store.Load()
	if got.DedicatedServerPath != "second" || got.DedicatedServerExtraArgs != "-b 2" {
		t.Fatalf("unexpected saved value: %+v", got)
	}
}
