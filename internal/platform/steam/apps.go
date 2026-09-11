package steam

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Apps scans appmanifest_*.acf files in every detected Steam library.
// Individual corrupt manifests are warnings rather than fatal scan errors.
func (s *Service) Apps(ctx context.Context) (AppInventory, error) {
	env, err := s.Detect(ctx)
	if err != nil {
		return AppInventory{}, err
	}
	return s.appsFromEnvironment(ctx, env)
}

// Snapshot performs one Steam environment discovery and scans AppManifest files
// from that exact environment. It is the preferred operation for UI refreshes.
func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	started := time.Now()
	env, err := s.Detect(ctx)
	if err != nil {
		return Snapshot{}, err
	}
	inventory, err := s.appsFromEnvironment(ctx, env)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Environment: env,
		Inventory:   inventory,
		ScannedAt:   time.Now().Unix(),
		DurationMs:  time.Since(started).Milliseconds(),
	}, nil
}

func (s *Service) appsFromEnvironment(ctx context.Context, env Environment) (AppInventory, error) {
	inventory := AppInventory{
		Detected: env.Detected,
		Apps:     []AppInstallation{},
		Warnings: append([]string{}, env.Warnings...),
	}
	if !env.Detected {
		return inventory, nil
	}

	byID := map[AppID]AppInstallation{}
	for _, library := range env.Libraries {
		if err := ctx.Err(); err != nil {
			return AppInventory{}, err
		}
		entries, readErr := os.ReadDir(library.SteamAppsPath)
		if readErr != nil {
			inventory.Warnings = append(inventory.Warnings, "无法读取 Steam 库："+library.SteamAppsPath+"："+readErr.Error())
			continue
		}

		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return AppInventory{}, err
			}
			if entry.IsDir() || !appManifestFilenamePattern.MatchString(entry.Name()) {
				continue
			}
			manifestPath := filepath.Join(library.SteamAppsPath, entry.Name())
			app, warnings, manifestErr := readAppManifest(manifestPath, library)
			inventory.Warnings = append(inventory.Warnings, warnings...)
			if manifestErr != nil {
				inventory.Warnings = append(inventory.Warnings, manifestErr.Error())
				continue
			}

			if previous, exists := byID[app.AppID]; exists {
				inventory.Warnings = append(inventory.Warnings, fmt.Sprintf("发现重复 Steam AppID %d，AI Game Manager Panel 将优先使用有效安装目录", app.AppID))
				byID[app.AppID] = preferredInstallation(previous, app)
				continue
			}
			byID[app.AppID] = app
		}
	}

	inventory.Apps = make([]AppInstallation, 0, len(byID))
	for _, app := range byID {
		inventory.Apps = append(inventory.Apps, app)
	}
	sort.Slice(inventory.Apps, func(i, j int) bool {
		left := strings.ToLower(inventory.Apps[i].Name)
		right := strings.ToLower(inventory.Apps[j].Name)
		if left == right {
			return inventory.Apps[i].AppID < inventory.Apps[j].AppID
		}
		if left == "" {
			return false
		}
		if right == "" {
			return true
		}
		return left < right
	})
	return inventory, nil
}

func preferredInstallation(left, right AppInstallation) AppInstallation {
	if left.InstallPathExists != right.InstallPathExists {
		if right.InstallPathExists {
			return right
		}
		return left
	}
	if left.Name == "" && right.Name != "" {
		return right
	}
	return left
}
