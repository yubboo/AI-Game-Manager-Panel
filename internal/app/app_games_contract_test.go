package app

import (
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/config"
	game "github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"
)

func TestGamePacksUseSingleConfiguredCatalog(t *testing.T) {
	a := &Application{platformConfig: config.PlatformConfig{Games: config.GamesConfig{Templates: []config.GameTemplate{
		{ID: "minecraft.java", Family: "minecraft", NameZh: "Minecraft Java", State: "planned", Capabilities: []string{"mods"}, UIPanels: []string{"mods"}},
		{ID: "steam.dst", Family: "steam", NameZh: "饥荒联机版", State: "supported", SupportedOS: []string{"windows"}, Capabilities: []string{"console"}, UIPanels: []string{"console"}},
	}}}}
	packs := a.GamePacks()
	if len(packs) != 2 || packs[0].ID != game.ID("minecraft.java") || packs[1].ID != game.ID("steam.dst") {
		t.Fatalf("unexpected packs: %+v", packs)
	}
	if packs[0].Executable() || !packs[1].Executable() {
		t.Fatalf("pack support state drifted: %+v", packs)
	}
}
