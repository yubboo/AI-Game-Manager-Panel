package dedicated

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/preferences"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

type fakeSteam struct{ env steam.Environment }

func (f fakeSteam) Detect(context.Context) (steam.Environment, error) { return f.env, nil }

func TestSnapshotUsesRememberedDedicatedPath(t *testing.T) {
	root := t.TempDir()
	install := filepath.Join(root, InstallDirName)
	if err := os.MkdirAll(filepath.Join(install, BinDirAMD64), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, BinDirAMD64, ExecutableAMD64), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	store := preferences.New(filepath.Join(t.TempDir(), "dst.json"))
	if err := store.Save(preferences.Value{DedicatedServerPath: install, DedicatedServerExtraArgs: "-foo 1"}); err != nil {
		t.Fatal(err)
	}

	service := New(fakeSteam{}, store)
	snapshot, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Installation.Source != InstallSourceManual || !snapshot.Installation.Valid {
		t.Fatalf("expected remembered install: %+v", snapshot.Installation)
	}
	if snapshot.ExtraArgs != "-foo 1" {
		t.Fatalf("expected saved extra args, got %q", snapshot.ExtraArgs)
	}
}
