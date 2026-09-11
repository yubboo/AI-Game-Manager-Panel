package dstworkspace

import (
	"context"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/common"
	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/dedicated"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/server/workspace"
)

const DSTGameID = "steam.dst"

type Snapshot struct {
	Workspace   gameworkspace.Snapshot `json:"workspace"`
	Dedicated   dedicated.Snapshot     `json:"dedicated"`
	Environment dstdomain.Environment  `json:"environment"`
}

type Service struct {
	games     *gameworkspace.Service
	dedicated *dedicated.Service
}

func New(games *gameworkspace.Service, dedicatedService *dedicated.Service) *Service {
	return &Service{games: games, dedicated: dedicatedService}
}

// Snapshot performs one targeted Steam workspace scan and reuses its Steam
// Environment for Dedicated Server validation. Desktop and Web call this exact
// method, avoiding duplicate Steam discovery and mismatched UI refreshes.
func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	workspace, err := s.games.Snapshot(ctx, game.ID(DSTGameID))
	if err != nil {
		return Snapshot{}, err
	}
	dstEnv := dstdomain.DiscoverEnvironment()
	return Snapshot{
		Workspace:   workspace,
		Dedicated:   s.dedicated.SnapshotWithEnvironments(workspace.Steam, dstEnv),
		Environment: dstEnv,
	}, nil
}
