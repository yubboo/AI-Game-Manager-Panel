package gameworkspace

import (
	"context"
	"errors"
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

type fakeSteamFinder struct {
	requested []steam.AppID
}

func (f *fakeSteamFinder) FindApps(_ context.Context, ids []steam.AppID) (steam.TargetedSnapshot, error) {
	f.requested = append([]steam.AppID(nil), ids...)
	return steam.TargetedSnapshot{
		Environment: steam.Environment{Detected: true, InstallPath: `D:\Steam`},
		Apps: []steam.AppInstallation{
			{AppID: 322330, Name: "Don't Starve Together", InstallPath: `D:\Steam\steamapps\common\Don't Starve Together`, InstallPathExists: true},
		},
		ScannedAt: 123,
	}, nil
}

func TestSnapshotOnlyRequestsSelectedGameAppIDs(t *testing.T) {
	finder := &fakeSteamFinder{}
	service := New([]game.CatalogEntry{{
		ID:     "steam.dst",
		Family: game.FamilySteam,
		NameZH: "饥荒联机版",
		Steam:  &game.SteamMetadata{GameAppID: 322330, ServerAppID: 343050},
	}}, finder)

	snapshot, err := service.Snapshot(context.Background(), "steam.dst")
	if err != nil {
		t.Fatal(err)
	}
	if len(finder.requested) != 2 || finder.requested[0] != 322330 || finder.requested[1] != 343050 {
		t.Fatalf("unexpected targeted ids: %#v", finder.requested)
	}
	if snapshot.Game.GameApp == nil || !snapshot.Game.GameApp.Installed {
		t.Fatalf("game installation was not mapped: %#v", snapshot.Game.GameApp)
	}
	if snapshot.Game.ServerApp == nil || snapshot.Game.ServerApp.Installed {
		t.Fatalf("missing server app should stay uninstalled: %#v", snapshot.Game.ServerApp)
	}
	if !snapshot.Steam.Detected {
		t.Fatal("steam environment should be carried into workspace snapshot")
	}
}

func TestSnapshotRejectsUnknownWorkspace(t *testing.T) {
	service := New(nil, &fakeSteamFinder{})
	_, err := service.Snapshot(context.Background(), "steam.unknown")
	if !errors.Is(err, ErrUnknownGame) {
		t.Fatalf("expected ErrUnknownGame, got %v", err)
	}
}
