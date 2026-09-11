package dstworkspace

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/preferences"
	steamdst "github.com/yubboo/AI-Game-Manager-Panel/internal/games/steam/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/server/workspace"
)

type countingSteam struct {
	findCalls   int
	detectCalls int
	env         steam.Environment
}

func (c *countingSteam) FindApps(context.Context, []steam.AppID) (steam.TargetedSnapshot, error) {
	c.findCalls++
	return steam.TargetedSnapshot{Environment: c.env, Apps: []steam.AppInstallation{}}, nil
}

func (c *countingSteam) Detect(context.Context) (steam.Environment, error) {
	c.detectCalls++
	return c.env, nil
}

func TestDSTWorkspaceRefreshReusesSteamEnvironment(t *testing.T) {
	backend := &countingSteam{env: steam.Environment{Detected: true, Libraries: []steam.Library{}}}
	games := gameworkspace.New([]game.CatalogEntry{steamdst.CatalogEntry()}, backend)
	dedicated := dedicated.New(backend, preferences.New(filepath.Join(t.TempDir(), "dst.json")))
	service := New(games, dedicated)

	if _, err := service.Snapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if backend.findCalls != 1 {
		t.Fatalf("expected one targeted Steam scan, got %d", backend.findCalls)
	}
	if backend.detectCalls != 0 {
		t.Fatalf("dedicated validation must reuse workspace Steam environment; extra Detect calls=%d", backend.detectCalls)
	}
}
