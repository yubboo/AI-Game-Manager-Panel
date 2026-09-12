package app

import (
	"context"
	"runtime"
	"sort"

	dstruntimecore "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"
	serverinstance "github.com/yubboo/AI-Game-Manager-Panel/internal/server/instance"
)

// GameInstances returns the shared instance model consumed by the visual UI and
// XiaoYu. 0.2.23 adopts existing DST clusters first; later deployers create the
// same Instance shape instead of inventing a second AI-only resource model.
func (a *Application) GameInstances() []serverinstance.Instance {
	if a == nil {
		return []serverinstance.Instance{}
	}
	env := a.DSTEnvironment()
	items := make([]serverinstance.Instance, 0, len(env.Clusters))
	ctx := a.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	for _, cluster := range env.Clusters {
		state := "stopped"
		managed := false
		if a.dstRuntime != nil {
			if snapshot, err := a.dstRuntime.ClusterStatus(ctx, dstruntimecore.ClusterRequest{ClusterPath: cluster.Path}); err == nil {
				state = snapshot.Overall
				managed = snapshot.Master.Found || snapshot.Caves.Found
			}
		}
		items = append(items, serverinstance.Instance{
			ID:           serverinstance.StableID("steam.dst", cluster.Path),
			Name:         cluster.Name,
			GameID:       "steam.dst",
			Origin:       serverinstance.OriginDiscovered,
			NodeOS:       runtime.GOOS,
			NodeArch:     runtime.GOARCH,
			InstallPath:  cluster.Path,
			RuntimeState: state,
			Capabilities: []string{"overview", "console", "players", "config", "mods", "network", "backups", "files", "shards"},
			Managed:      managed,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].GameID == items[j].GameID {
			return items[i].Name < items[j].Name
		}
		return items[i].GameID < items[j].GameID
	})
	return items
}
