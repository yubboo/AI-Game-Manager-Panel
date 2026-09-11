package dst

import (
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"
)

func TestCatalogMetadata(t *testing.T) {
	entry := CatalogEntry()
	if entry.ID != ID || entry.Family != game.FamilySteam {
		t.Fatalf("unexpected catalog identity: %#v", entry)
	}
	if entry.NameZH != "饥荒联机版" || entry.NameEN != "Don't Starve Together" {
		t.Fatalf("unexpected localized names: %#v", entry)
	}
	if entry.Steam == nil || entry.Steam.GameAppID != 322330 || entry.Steam.ServerAppID != 343050 {
		t.Fatalf("unexpected Steam IDs: %#v", entry.Steam)
	}
}
