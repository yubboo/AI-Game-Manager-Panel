package gameworkspace

import (
	"context"
	"errors"
	"time"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

var ErrUnknownGame = errors.New("unknown game workspace")

// AppState is the installation state of one Steam AppID required by a game workspace.
type AppState struct {
	AppID             uint32 `json:"appId"`
	Installed         bool   `json:"installed"`
	Name              string `json:"name"`
	InstallPath       string `json:"installPath"`
	InstallPathExists bool   `json:"installPathExists"`
	LibraryPath       string `json:"libraryPath"`
	BuildID           string `json:"buildId"`
	LastUpdated       int64  `json:"lastUpdated"`
	SizeOnDisk        uint64 `json:"sizeOnDisk"`
}

// GameState combines static provider/catalog metadata with local installation state.
type GameState struct {
	Catalog   game.CatalogEntry `json:"catalog"`
	GameApp   *AppState         `json:"gameApp,omitempty"`
	ServerApp *AppState         `json:"serverApp,omitempty"`
}

// Snapshot is a targeted snapshot for exactly one selected game workspace.
// It intentionally does not enumerate unrelated Steam applications.
type Snapshot struct {
	Game       GameState         `json:"game"`
	Steam      steam.Environment `json:"steam"`
	Warnings   []string          `json:"warnings"`
	ScannedAt  int64             `json:"scannedAt"`
	DurationMs int64             `json:"durationMs"`
}

type SteamFinder interface {
	FindApps(context.Context, []steam.AppID) (steam.TargetedSnapshot, error)
}

type Service struct {
	catalog map[game.ID]game.CatalogEntry
	steam   SteamFinder
}

func New(catalog []game.CatalogEntry, steamService SteamFinder) *Service {
	entries := make(map[game.ID]game.CatalogEntry, len(catalog))
	for _, entry := range catalog {
		entries[entry.ID] = entry
	}
	return &Service{catalog: entries, steam: steamService}
}

func (s *Service) Snapshot(ctx context.Context, id game.ID) (Snapshot, error) {
	started := time.Now()
	entry, ok := s.catalog[id]
	if !ok {
		return Snapshot{}, ErrUnknownGame
	}

	state := GameState{Catalog: entry}
	result := Snapshot{Game: state, ScannedAt: time.Now().Unix()}
	if entry.Family != game.FamilySteam || entry.Steam == nil {
		result.DurationMs = time.Since(started).Milliseconds()
		return result, nil
	}

	ids := make([]steam.AppID, 0, 2)
	if entry.Steam.GameAppID != 0 {
		ids = append(ids, steam.AppID(entry.Steam.GameAppID))
	}
	if entry.Steam.ServerAppID != 0 {
		ids = append(ids, steam.AppID(entry.Steam.ServerAppID))
	}

	steamSnapshot, err := s.steam.FindApps(ctx, ids)
	if err != nil {
		return Snapshot{}, err
	}
	byAppID := make(map[uint32]steam.AppInstallation, len(steamSnapshot.Apps))
	for _, app := range steamSnapshot.Apps {
		byAppID[uint32(app.AppID)] = app
	}

	state.GameApp = appState(entry.Steam.GameAppID, byAppID)
	state.ServerApp = appState(entry.Steam.ServerAppID, byAppID)
	result.Game = state
	result.Steam = steamSnapshot.Environment
	result.Warnings = append([]string(nil), steamSnapshot.Warnings...)
	result.ScannedAt = steamSnapshot.ScannedAt
	result.DurationMs = time.Since(started).Milliseconds()
	return result, nil
}

func appState(appID uint32, found map[uint32]steam.AppInstallation) *AppState {
	if appID == 0 {
		return nil
	}
	state := &AppState{AppID: appID}
	app, ok := found[appID]
	if !ok {
		return state
	}
	state.Installed = app.InstallPathExists
	state.Name = app.Name
	state.InstallPath = app.InstallPath
	state.InstallPathExists = app.InstallPathExists
	state.LibraryPath = app.LibraryPath
	state.BuildID = app.BuildID
	state.LastUpdated = app.LastUpdated
	state.SizeOnDisk = app.SizeOnDisk
	return state
}
