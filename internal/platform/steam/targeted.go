package steam

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// FindApps performs a targeted AppManifest lookup for the requested AppIDs.
// It intentionally avoids enumerating every appmanifest file in the user's
// Steam libraries, which keeps the normal AI Game Manager Panel game library lightweight.
func (s *Service) FindApps(ctx context.Context, appIDs []AppID) (TargetedSnapshot, error) {
	started := time.Now()
	env, err := s.Detect(ctx)
	if err != nil {
		return TargetedSnapshot{}, err
	}

	result := TargetedSnapshot{
		Environment: env,
		Apps:        []AppInstallation{},
		Warnings:    append([]string{}, env.Warnings...),
	}
	if !env.Detected || len(appIDs) == 0 {
		result.ScannedAt = time.Now().Unix()
		result.DurationMs = time.Since(started).Milliseconds()
		return result, nil
	}

	requested := make(map[AppID]struct{}, len(appIDs))
	for _, appID := range appIDs {
		if appID != 0 {
			requested[appID] = struct{}{}
		}
	}

	found := make(map[AppID]AppInstallation, len(requested))
	for appID := range requested {
		if err := ctx.Err(); err != nil {
			return TargetedSnapshot{}, err
		}
		filename := fmt.Sprintf("appmanifest_%d.acf", appID)
		for _, library := range env.Libraries {
			manifestPath := filepath.Join(library.SteamAppsPath, filename)
			if _, statErr := os.Stat(manifestPath); statErr != nil {
				if os.IsNotExist(statErr) {
					continue
				}
				result.Warnings = append(result.Warnings, "无法检查 Steam 清单："+manifestPath+"："+statErr.Error())
				continue
			}

			app, warnings, readErr := readAppManifest(manifestPath, library)
			result.Warnings = append(result.Warnings, warnings...)
			if readErr != nil {
				result.Warnings = append(result.Warnings, readErr.Error())
				continue
			}
			if previous, exists := found[appID]; exists {
				found[appID] = preferredInstallation(previous, app)
			} else {
				found[appID] = app
			}
		}
	}

	for _, app := range found {
		result.Apps = append(result.Apps, app)
	}
	sort.Slice(result.Apps, func(i, j int) bool { return result.Apps[i].AppID < result.Apps[j].AppID })
	result.ScannedAt = time.Now().Unix()
	result.DurationMs = time.Since(started).Milliseconds()
	return result, nil
}
