package dedicated

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

var ErrExecutableNotFound = errors.New("DST Dedicated Server executable not found")

// ValidateInstallDir mirrors DSTCamp dedicated_server.is_valid_install_dir().
// The basename check is intentional: the normal DST client also contains a
// nullrenderer executable and must not be mistaken for the free Dedicated
// Server package.
func ValidateInstallDir(path string) Installation {
	cleaned := filepath.Clean(strings.TrimSpace(path))
	result := Installation{
		AppID:   DedicatedServerAppID,
		RootDir: cleaned,
		Source:  InstallSourceNone,
		Issues:  []string{},
	}
	if cleaned == "" || cleaned == "." {
		result.Issues = append(result.Issues, "未提供专用服务器安装目录")
		return result
	}
	info, err := os.Stat(cleaned)
	if err != nil || !info.IsDir() {
		result.Issues = append(result.Issues, "专用服务器安装目录不存在")
		return result
	}
	result.Detected = true
	if !strings.Contains(strings.ToLower(filepath.Base(cleaned)), "dedicated server") {
		result.Issues = append(result.Issues, "目录名称不像 Don't Starve Together Dedicated Server；为避免误把游戏本体当成专用服务器，已拒绝该目录")
		return result
	}

	bitness, binDir, executable, err := PickBitness(cleaned)
	if err != nil {
		result.Issues = append(result.Issues, "未找到 DST Dedicated Server 专用服务器可执行文件")
		return result
	}
	result.Valid = true
	result.Bitness = bitness
	result.BinDir = binDir
	result.Executable = executable
	if bitness == 64 {
		result.Architecture = ArchitectureAMD64
	} else {
		result.Architecture = Architecture386
	}
	return result
}

// PickBitness preserves the Python priority: 64-bit first, 32-bit fallback.
func PickBitness(installDir string) (int, string, string, error) {
	candidates := []struct {
		bitness int
		binDir  string
		exe     string
	}{
		{64, BinDirAMD64, ExecutableAMD64},
		{32, BinDir386, Executable386},
	}
	for _, candidate := range candidates {
		binDir := filepath.Join(installDir, candidate.binDir)
		executable := filepath.Join(binDir, candidate.exe)
		if info, err := os.Stat(executable); err == nil && !info.IsDir() {
			return candidate.bitness, binDir, executable, nil
		}
	}
	return 0, "", "", ErrExecutableNotFound
}

// FindBinDir returns the selected bin64/bin directory, or an empty string when
// the expected executable is absent. This is the Go equivalent of Python's
// find_bin64_dir() despite the legacy function name.
func FindBinDir(installDir string) string {
	_, binDir, _, err := PickBitness(installDir)
	if err != nil {
		return ""
	}
	return binDir
}

// DiscoverInstall selects the dedicated-server installation using the Python
// baseline's priority: a remembered manual path if it is valid, then each Steam
// library's steamapps/common/Don't Starve Together Dedicated Server directory.
func DiscoverInstall(manualPath string, libraries []steam.Library) (Installation, []string) {
	warnings := []string{}
	if strings.TrimSpace(manualPath) != "" {
		candidate := ValidateInstallDir(manualPath)
		candidate.Source = InstallSourceManual
		if candidate.Valid {
			return candidate, warnings
		}
		warnings = append(warnings, "已保存的专用服务器路径无效，AGMP 已继续尝试 Steam Library 自动发现")
	}

	for _, library := range libraries {
		candidatePath := filepath.Join(library.Path, "steamapps", "common", InstallDirName)
		candidate := ValidateInstallDir(candidatePath)
		candidate.Source = InstallSourceSteam
		candidate.LibraryPath = library.Path
		if candidate.Valid {
			return candidate, warnings
		}
	}

	return Installation{
		Detected: false,
		Valid:    false,
		AppID:    DedicatedServerAppID,
		Source:   InstallSourceNone,
		Issues:   []string{"未检测到有效的 Don't Starve Together Dedicated Server 安装"},
	}, warnings
}
