package steam

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Service is AI Game Manager Panel's shared Steam platform implementation.
// Concrete games consume this service rather than reimplementing Steam discovery.
type Service struct {
	candidateRoots func(context.Context) []string
}

func New() *Service {
	return &Service{candidateRoots: platformSteamRoots}
}

func newWithRoots(roots ...string) *Service {
	return &Service{candidateRoots: func(context.Context) []string { return roots }}
}

func (s *Service) Detect(ctx context.Context) (Environment, error) {
	env := Environment{Libraries: []Library{}, Warnings: []string{}}

	root := firstSteamRoot(s.candidateRoots(ctx))
	if root == "" {
		return env, nil
	}

	env.Detected = true
	env.InstallPath = root
	env.ExecutablePath = steamExecutable(root)

	libraries, warnings := discoverLibraries(root)
	env.Libraries = libraries
	env.Warnings = warnings
	return env, nil
}

func (s *Service) Libraries(ctx context.Context) ([]Library, error) {
	env, err := s.Detect(ctx)
	if err != nil {
		return nil, err
	}
	return env.Libraries, nil
}

func (s *Service) FindApp(ctx context.Context, appID AppID) (AppInstallation, bool, error) {
	snapshot, err := s.FindApps(ctx, []AppID{appID})
	if err != nil {
		return AppInstallation{}, false, err
	}
	for _, app := range snapshot.Apps {
		if app.AppID == appID {
			return app, true, nil
		}
	}
	return AppInstallation{}, false, nil
}

func firstSteamRoot(candidates []string) string {
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = normalizePath(candidate)
		if candidate == "" {
			continue
		}
		key := comparablePath(candidate)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		if steamRootLooksValid(candidate) {
			return candidate
		}
	}
	return ""
}

func steamRootLooksValid(root string) bool {
	if info, err := os.Stat(filepath.Join(root, "steamapps")); err == nil && info.IsDir() {
		return true
	}
	if info, err := os.Stat(steamExecutable(root)); err == nil && !info.IsDir() {
		return true
	}
	return false
}

func steamExecutable(root string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(root, "steam.exe")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(root, "Steam.AppBundle", "Steam", "Contents", "MacOS", "steam_osx")
	}
	return filepath.Join(root, "steam.sh")
}

func discoverLibraries(root string) ([]Library, []string) {
	libraries := []Library{}
	warnings := []string{}
	primarySteamApps := filepath.Join(root, "steamapps")
	if info, err := os.Stat(primarySteamApps); err == nil && info.IsDir() {
		libraries = append(libraries, Library{
			Path:          root,
			SteamAppsPath: primarySteamApps,
			Primary:       true,
		})
	} else {
		warnings = append(warnings, "已发现 Steam，但主 steamapps 目录尚不可用："+primarySteamApps)
	}

	vdfPath := filepath.Join(primarySteamApps, "libraryfolders.vdf")
	data, err := os.ReadFile(vdfPath)
	if os.IsNotExist(err) {
		return libraries, warnings
	}
	if err != nil {
		return libraries, append(warnings, "无法读取 Steam libraryfolders.vdf："+err.Error())
	}

	paths := parseLibraryFolderPaths(string(data))
	seen := map[string]struct{}{comparablePath(root): {}}
	for _, path := range paths {
		path = normalizePath(path)
		if path == "" {
			continue
		}
		key := comparablePath(path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		steamApps := filepath.Join(path, "steamapps")
		if info, statErr := os.Stat(steamApps); statErr != nil || !info.IsDir() {
			warnings = append(warnings, "Steam 库目录不存在或不可访问："+path)
			continue
		}

		libraries = append(libraries, Library{
			Path:          path,
			SteamAppsPath: steamApps,
			Primary:       false,
		})
	}

	return libraries, warnings
}

func normalizePath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"")
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, `\\`, `\`)
	return filepath.Clean(filepath.FromSlash(value))
}

func comparablePath(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}
