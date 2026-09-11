package steam

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	appManifestFilenamePattern = regexp.MustCompile(`(?i)^appmanifest_([0-9]+)\.acf$`)
	windowsAbsolutePathPattern = regexp.MustCompile(`(?i)^[a-z]:[\\/]`)
)

func manifestAppIDFromPath(path string) (AppID, error) {
	match := appManifestFilenamePattern.FindStringSubmatch(filepath.Base(path))
	if len(match) != 2 {
		return 0, fmt.Errorf("invalid Steam app manifest filename: %s", filepath.Base(path))
	}
	value, err := strconv.ParseUint(match[1], 10, 32)
	if err != nil || value == 0 {
		return 0, fmt.Errorf("invalid Steam app id in manifest filename: %s", filepath.Base(path))
	}
	return AppID(value), nil
}

func readAppManifest(path string, library Library) (AppInstallation, []string, error) {
	fileAppID, err := manifestAppIDFromPath(path)
	if err != nil {
		return AppInstallation{}, nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return AppInstallation{}, nil, fmt.Errorf("read Steam app manifest %s: %w", path, err)
	}

	values := parseKeyValueDocument(string(data))
	warnings := []string{}
	app := AppInstallation{
		AppID:        fileAppID,
		Name:         strings.TrimSpace(values["name"]),
		InstallDir:   strings.TrimSpace(values["installdir"]),
		LibraryPath:  library.Path,
		ManifestPath: path,
		BuildID:      strings.TrimSpace(values["buildid"]),
	}

	if manifestAppID := strings.TrimSpace(values["appid"]); manifestAppID != "" {
		parsed, parseErr := strconv.ParseUint(manifestAppID, 10, 32)
		if parseErr != nil || parsed == 0 {
			warnings = append(warnings, fmt.Sprintf("Steam manifest %s 的 appid 无效：%s", filepath.Base(path), manifestAppID))
		} else if AppID(parsed) != fileAppID {
			warnings = append(warnings, fmt.Sprintf("Steam manifest %s 的文件名 AppID 与内容不一致，已使用文件名中的 %d", filepath.Base(path), fileAppID))
		}
	}

	app.LastUpdated = parseInt64Field(values, "lastupdated", path, &warnings)
	app.SizeOnDisk = parseUint64Field(values, "sizeondisk", path, &warnings)
	app.StateFlags = parseUint64Field(values, "stateflags", path, &warnings)
	app.BytesDownloaded = parseUint64Field(values, "bytesdownloaded", path, &warnings)
	app.BytesToDownload = parseUint64Field(values, "bytestodownload", path, &warnings)
	app.BytesStaged = parseUint64Field(values, "bytesstaged", path, &warnings)
	app.BytesToStage = parseUint64Field(values, "bytestostage", path, &warnings)
	app.TargetBuildID = strings.TrimSpace(values["targetbuildid"])
	app.Branch = strings.TrimSpace(values["betakey"])

	if app.InstallDir == "" {
		warnings = append(warnings, fmt.Sprintf("Steam manifest %s 缺少 installdir", filepath.Base(path)))
		return app, warnings, nil
	}
	if !safeInstallDir(app.InstallDir) {
		warnings = append(warnings, fmt.Sprintf("Steam manifest %s 的 installdir 不安全，已忽略安装路径", filepath.Base(path)))
		return app, warnings, nil
	}

	app.InstallPath = filepath.Join(library.SteamAppsPath, "common", installDirToPath(app.InstallDir))
	if info, statErr := os.Stat(app.InstallPath); statErr == nil && info.IsDir() {
		app.InstallPathExists = true
	}
	return app, warnings, nil
}

func safeInstallDir(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || windowsAbsolutePathPattern.MatchString(value) {
		return false
	}
	canonical := strings.ReplaceAll(value, `\`, "/")
	if strings.HasPrefix(canonical, "//") {
		return false
	}
	cleaned := path.Clean(canonical)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
		return false
	}
	return true
}

func installDirToPath(value string) string {
	return filepath.FromSlash(strings.ReplaceAll(value, `\`, "/"))
}

func parseInt64Field(values map[string]string, key, path string, warnings *[]string) int64 {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("Steam manifest %s 的 %s 无效：%s", filepath.Base(path), key, value))
		return 0
	}
	return parsed
}

func parseUint64Field(values map[string]string, key, path string, warnings *[]string) uint64 {
	value := strings.TrimSpace(values[key])
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		*warnings = append(*warnings, fmt.Sprintf("Steam manifest %s 的 %s 无效：%s", filepath.Base(path), key, value))
		return 0
	}
	return parsed
}
