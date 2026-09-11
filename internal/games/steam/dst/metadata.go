package dst

import "github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"

const (
	ID          game.ID = "steam.dst"
	GameAppID   uint32  = 322330
	ServerAppID uint32  = 343050
)

// CatalogEntry is the first real game definition in AGMP's catalog.
// Runtime/config/server control are deliberately not implemented in 0.1.16.
func CatalogEntry() game.CatalogEntry {
	return game.CatalogEntry{
		ID:          ID,
		Family:      game.FamilySteam,
		NameZH:      "饥荒联机版",
		NameEN:      "Don't Starve Together",
		Description: "Klei 的多人合作生存游戏。AGMP 将通过 SteamCMD 管理专用服务器生命周期。",
		Aliases: []string{
			"DST",
			"饥荒",
			"Don't Starve Together Dedicated Server",
			"饥荒联机版专用服务器",
		},
		Steam: &game.SteamMetadata{
			GameAppID:   GameAppID,
			ServerAppID: ServerAppID,
		},
	}
}
