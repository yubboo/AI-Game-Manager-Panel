package dedicated

import (
	"context"
	"errors"
	"strings"

	dstdomain "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/preferences"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

type SteamDetector interface {
	Detect(context.Context) (steam.Environment, error)
}

type Service struct {
	steam SteamDetector
	prefs *preferences.Store
}

func New(steamDetector SteamDetector, prefs *preferences.Store) *Service {
	return &Service{steam: steamDetector, prefs: prefs}
}

func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	steamEnv, err := s.steam.Detect(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	return s.SnapshotWithSteamEnvironment(steamEnv), nil
}

// SnapshotWithSteamEnvironment reuses a Steam discovery result produced by an
// outer workspace refresh. This prevents the DST page from scanning the Steam
// root and libraryfolders.vdf twice during one user refresh.
func (s *Service) SnapshotWithSteamEnvironment(steamEnv steam.Environment) Snapshot {
	return s.SnapshotWithEnvironments(steamEnv, dstdomain.DiscoverEnvironment())
}

// SnapshotWithEnvironments reuses both Steam and DST/Klei discovery when an
// outer workspace refresh already collected them.
func (s *Service) SnapshotWithEnvironments(steamEnv steam.Environment, dstEnv dstdomain.Environment) Snapshot {
	pref := s.prefs.Load()
	install, warnings := DiscoverInstall(pref.DedicatedServerPath, steamEnv.Libraries)
	conf := DescribeConfDir(dstEnv.DocumentsDir, dstEnv.KleiRoot)
	if conf.Error != "" {
		warnings = append(warnings, conf.Error)
	}
	return Snapshot{
		Installation: install,
		ConfDir:      conf,
		ManualPath:   pref.DedicatedServerPath,
		ExtraArgs:    pref.DedicatedServerExtraArgs,
		Warnings:     warnings,
	}
}

func (s *Service) Preferences() preferences.Value { return s.prefs.Load() }

func (s *Service) SavePreferences(value preferences.Value) error {
	return s.prefs.Save(value)
}

func (s *Service) BuildLaunchSpec(ctx context.Context, request LaunchRequest) (LaunchSpec, error) {
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return LaunchSpec{}, err
	}
	if !snapshot.Installation.Valid {
		return LaunchSpec{}, errors.New("未检测到有效的 DST Dedicated Server 安装")
	}
	if strings.TrimSpace(request.KleiRoot) == "" {
		request.KleiRoot = snapshot.ConfDir.KleiRoot
	}
	if strings.TrimSpace(request.ExtraArgs) == "" {
		request.ExtraArgs = snapshot.ExtraArgs
	}
	return BuildLaunchSpec(snapshot.Installation, snapshot.ConfDir.DocumentsDir, request)
}
