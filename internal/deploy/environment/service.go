package environment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	platformfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/files"
	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

const (
	steamCMDWindowsURL = "https://steamcdn-a.akamaihd.net/client/installer/steamcmd.zip"
	steamCMDLinuxURL   = "https://steamcdn-a.akamaihd.net/client/installer/steamcmd_linux.tar.gz"
	dstDedicatedAppID  = steam.AppID(343050)
)

type SteamFinder interface {
	FindApp(context.Context, steam.AppID) (steam.AppInstallation, bool, error)
}

type Options struct {
	Root        string
	DataDir     string
	InstanceDir string
	BackupDir   string
	TempDir     string
	ExportDir   string
	PluginDir   string
	CacheDir    string
	Steam       SteamFinder
	HTTPClient  *http.Client
}

type ToolStatus struct {
	Detected bool   `json:"detected"`
	Path     string `json:"path"`
	Root     string `json:"root"`
	Source   string `json:"source"`
}

type GameInstallStatus struct {
	AppID    uint32 `json:"appId"`
	Detected bool   `json:"detected"`
	Path     string `json:"path"`
}

// StoragePaths 是所有平台 Adapter 共用的路径真相源。业务代码不应再从当前工作目录拼接 games/tools。
type StoragePaths struct {
	ProgramDir         string `json:"programDir"`
	DataDir            string `json:"dataDir"`
	RuntimeRoot        string `json:"runtimeRoot"`
	SteamCMDRoot       string `json:"steamcmdRoot"`
	SteamCMDPath       string `json:"steamcmdPath"`
	GameLibraryRoot    string `json:"gameLibraryRoot"`
	InstanceConfigRoot string `json:"instanceConfigRoot"`
	GameSaveRoot       string `json:"gameSaveRoot"`
	CacheRoot          string `json:"cacheRoot"`
}

type Status struct {
	Platform           string            `json:"platform"`
	Architecture       string            `json:"architecture"`
	Initialized        bool              `json:"initialized"`
	InitializedAt      int64             `json:"initializedAt"`
	Skipped            bool              `json:"skipped"`
	SkippedAt          int64             `json:"skippedAt"`
	SteamCMD           ToolStatus        `json:"steamcmd"`
	DedicatedServer    GameInstallStatus `json:"dedicatedServer"`
	DefaultInstallRoot string            `json:"defaultInstallRoot"` // 兼容旧前端，等同 GameLibraryRoot。
	Paths              StoragePaths      `json:"paths"`
	RequiredDirs       []string          `json:"requiredDirs"`
	Warnings           []string          `json:"warnings"`
}

type InitializeRequest struct {
	SteamCMDPath       string `json:"steamcmdPath"`
	DefaultInstallRoot string `json:"defaultInstallRoot"`
	SteamCMDRoot       string `json:"steamcmdRoot,omitempty"`
	GameLibraryRoot    string `json:"gameLibraryRoot,omitempty"`
}

type StoragePathsRequest struct {
	SteamCMDRoot       string `json:"steamcmdRoot"`
	SteamCMDPath       string `json:"steamcmdPath"`
	GameLibraryRoot    string `json:"gameLibraryRoot"`
	InstanceConfigRoot string `json:"instanceConfigRoot"`
	GameSaveRoot       string `json:"gameSaveRoot"`
	CacheRoot          string `json:"cacheRoot"`
}

type SteamCMDInstallRequest struct {
	TargetRoot string `json:"targetRoot"`
}

type MigratePathRequest struct {
	TargetRoot   string `json:"targetRoot"`
	RemoveSource bool   `json:"removeSource"`
}

type MigrationResult struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	CopiedFiles int    `json:"copiedFiles"`
	CopiedBytes int64  `json:"copiedBytes"`
	RemovedOld  bool   `json:"removedOld"`
	Message     string `json:"message"`
}

type persistedState struct {
	Version            int    `json:"version"`
	Initialized        bool   `json:"initialized"`
	InitializedAt      int64  `json:"initializedAt"`
	Skipped            bool   `json:"skipped"`
	SkippedAt          int64  `json:"skippedAt"`
	SteamCMDPath       string `json:"steamcmdPath"`
	SteamCMDRoot       string `json:"steamcmdRoot"`
	GameLibraryRoot    string `json:"gameLibraryRoot"`
	DefaultInstallRoot string `json:"defaultInstallRoot,omitempty"` // 读取 0.1.62 旧状态。
	InstanceConfigRoot string `json:"instanceConfigRoot"`
	GameSaveRoot       string `json:"gameSaveRoot"`
	CacheRoot          string `json:"cacheRoot"`
}

type Service struct {
	options    Options
	path       string
	mu         sync.RWMutex
	state      persistedState
	registryMu sync.RWMutex
	runtimes   []RuntimeRecord
	installMu  sync.Mutex
}

func New(options Options) *Service {
	if options.HTTPClient == nil {
		options.HTTPClient = &http.Client{Timeout: 90 * time.Second}
	}
	service := &Service{options: options, path: filepath.Join(options.DataDir, "environment", "setup.json")}
	_ = service.load()
	_ = service.loadRegistry()
	return service
}

func (s *Service) Status(ctx context.Context) Status {
	s.mu.RLock()
	state := s.state
	s.mu.RUnlock()
	paths := s.resolvePaths(state)
	status := Status{
		Platform: runtime.GOOS, Architecture: runtime.GOARCH,
		Initialized: state.Initialized, InitializedAt: state.InitializedAt, Skipped: state.Skipped, SkippedAt: state.SkippedAt,
		DefaultInstallRoot: paths.GameLibraryRoot, Paths: paths, RequiredDirs: s.requiredDirs(paths),
		DedicatedServer: GameInstallStatus{AppID: uint32(dstDedicatedAppID)}, Warnings: []string{},
	}
	status.SteamCMD = s.detectSteamCMD(state.SteamCMDPath, paths.SteamCMDRoot)
	if status.SteamCMD.Detected {
		status.Paths.SteamCMDPath = status.SteamCMD.Path
		status.Paths.SteamCMDRoot = status.SteamCMD.Root
	}
	if s.options.Steam != nil {
		app, found, err := s.options.Steam.FindApp(ctx, dstDedicatedAppID)
		if err != nil {
			status.Warnings = append(status.Warnings, "检测 DST Dedicated Server 失败："+err.Error())
		} else if found && app.InstallPath != "" {
			status.DedicatedServer.Detected = app.InstallPathExists
			status.DedicatedServer.Path = app.InstallPath
		}
	}
	if state.Initialized && !status.SteamCMD.Detected {
		status.Warnings = append(status.Warnings, "已完成过初始化，但当前 SteamCMD 路径失效，请重新检测、选择或安装。")
	}
	return status
}

func (s *Service) Initialize(ctx context.Context, request InitializeRequest) (Status, error) {
	current := s.Status(ctx)
	steamCMDPath := strings.TrimSpace(request.SteamCMDPath)
	if steamCMDPath == "" {
		steamCMDPath = current.SteamCMD.Path
	}
	if steamCMDPath != "" {
		if info, err := os.Stat(steamCMDPath); err != nil || info.IsDir() {
			return current, errors.New("SteamCMD 路径无效，请选择 steamcmd 可执行文件")
		}
	}
	gameRoot := firstNonEmpty(request.GameLibraryRoot, request.DefaultInstallRoot, current.Paths.GameLibraryRoot)
	gameRoot = s.absolutePath(gameRoot)
	steamDirFromPath := ""
	if steamCMDPath != "" {
		steamDirFromPath = filepath.Dir(steamCMDPath)
	}
	steamRoot := firstNonEmpty(request.SteamCMDRoot, steamDirFromPath, current.Paths.SteamCMDRoot)
	steamRoot = s.absolutePath(steamRoot)
	paths := current.Paths
	paths.SteamCMDPath = filepath.Clean(steamCMDPath)
	paths.SteamCMDRoot = steamRoot
	paths.GameLibraryRoot = gameRoot
	for _, dir := range s.requiredDirs(paths) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return current, fmt.Errorf("创建运行目录失败 %s: %w", dir, err)
		}
	}
	next := s.stateFromPaths(paths)
	next.Initialized = true
	next.InitializedAt = time.Now().Unix()
	next.Skipped = false
	next.SkippedAt = 0
	if err := s.replaceState(next); err != nil {
		return current, err
	}
	return s.Status(ctx), nil
}

func (s *Service) UpdateStoragePaths(ctx context.Context, request StoragePathsRequest) (Status, error) {
	current := s.Status(ctx)
	paths := current.Paths
	if value := strings.TrimSpace(request.SteamCMDRoot); value != "" {
		paths.SteamCMDRoot = s.absolutePath(value)
	}
	if value := strings.TrimSpace(request.SteamCMDPath); value != "" {
		paths.SteamCMDPath = s.absolutePath(value)
	}
	if value := strings.TrimSpace(request.GameLibraryRoot); value != "" {
		paths.GameLibraryRoot = s.absolutePath(value)
	}
	if value := strings.TrimSpace(request.InstanceConfigRoot); value != "" {
		paths.InstanceConfigRoot = s.absolutePath(value)
	}
	if value := strings.TrimSpace(request.GameSaveRoot); value != "" {
		paths.GameSaveRoot = s.absolutePath(value)
	}
	if value := strings.TrimSpace(request.CacheRoot); value != "" {
		paths.CacheRoot = s.absolutePath(value)
	}
	if paths.GameLibraryRoot == "" {
		return current, errors.New("游戏服务器库目录不能为空")
	}
	for _, dir := range []string{paths.GameLibraryRoot, paths.InstanceConfigRoot, paths.CacheRoot} {
		if strings.TrimSpace(dir) != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return current, fmt.Errorf("创建目录失败 %s: %w", dir, err)
			}
		}
	}
	next := s.stateFromPaths(paths)
	next.Initialized = s.state.Initialized
	next.InitializedAt = s.state.InitializedAt
	next.Skipped = s.state.Skipped
	next.SkippedAt = s.state.SkippedAt
	if err := s.replaceState(next); err != nil {
		return current, err
	}
	return s.Status(ctx), nil
}

func (s *Service) Skip(ctx context.Context) (Status, error) {
	current := s.Status(ctx)
	s.mu.Lock()
	previous := s.state
	s.state.Version = 3
	s.state.Initialized = false
	s.state.InitializedAt = 0
	s.state.Skipped = true
	s.state.SkippedAt = time.Now().Unix()
	if s.state.GameLibraryRoot == "" {
		s.state.GameLibraryRoot = current.Paths.GameLibraryRoot
	}
	err := s.saveLocked()
	if err != nil {
		s.state = previous
	}
	s.mu.Unlock()
	if err != nil {
		return current, err
	}
	return s.Status(ctx), nil
}

func (s *Service) InstallSteamCMD(ctx context.Context) (Status, error) {
	return s.InstallSteamCMDAt(ctx, SteamCMDInstallRequest{})
}

func (s *Service) InstallSteamCMDAt(ctx context.Context, request SteamCMDInstallRequest) (Status, error) {
	s.installMu.Lock()
	defer s.installMu.Unlock()
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return s.Status(ctx), fmt.Errorf("SteamCMD 自动安装当前仅支持 Windows/Linux，当前平台：%s", runtime.GOOS)
	}
	current := s.Status(ctx)
	targetDir := firstNonEmpty(request.TargetRoot, current.Paths.SteamCMDRoot, s.defaultSteamCMDRoot())
	targetDir = s.absolutePath(targetDir)
	stage := targetDir + ".stage-" + fmt.Sprint(time.Now().UnixNano())
	_ = os.RemoveAll(stage)
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return current, err
	}
	defer os.RemoveAll(stage)
	url := steamCMDWindowsURL
	ext := ".zip"
	if runtime.GOOS == "linux" {
		url = steamCMDLinuxURL
		ext = ".tar.gz"
	}
	archivePath := filepath.Join(s.options.TempDir, fmt.Sprintf("steamcmd-%d%s", time.Now().UnixNano(), ext))
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return current, err
	}
	defer os.Remove(archivePath)
	checksum, err := downloadWithSHA256(ctx, s.options.HTTPClient, url, archivePath, 256<<20)
	if err != nil {
		return current, fmt.Errorf("下载 SteamCMD 失败: %w", err)
	}
	if runtime.GOOS == "windows" {
		err = extractZIP(archivePath, stage)
	} else {
		err = extractTarGZ(archivePath, stage)
	}
	if err != nil {
		return current, fmt.Errorf("SteamCMD 压缩包无效: %w", err)
	}
	executable := filepath.Join(stage, managedSteamCMDExecutableName())
	if !regularFileExists(executable) {
		return current, fmt.Errorf("SteamCMD 下载完成，但没有找到 %s", managedSteamCMDExecutableName())
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(executable, 0o755)
	}
	if err := replaceManagedTree(stage, targetDir, RuntimeRecord{Kind: RuntimeSteamCMD}); err != nil {
		return current, err
	}
	finalExecutable := filepath.Join(targetDir, managedSteamCMDExecutableName())
	record := RuntimeRecord{ID: "steamcmd-managed", Kind: RuntimeSteamCMD, Name: "SteamCMD", Version: "official", OS: runtime.GOOS, Arch: runtime.GOARCH, Root: targetDir, Executable: finalExecutable, Source: "steam-official", Managed: true, Default: true, Checksum: checksum, InstalledAt: time.Now().Unix(), LastVerified: time.Now().Unix()}
	if _, err := s.upsertRuntime(record, true); err != nil {
		return current, err
	}
	next := s.state
	next.Version = 4
	next.SteamCMDRoot = targetDir
	next.SteamCMDPath = finalExecutable
	if next.GameLibraryRoot == "" {
		next.GameLibraryRoot = current.Paths.GameLibraryRoot
	}
	if err := s.replaceState(next); err != nil {
		return current, err
	}
	return s.Status(ctx), nil
}

func (s *Service) MigrateSteamCMD(ctx context.Context, request MigratePathRequest) (MigrationResult, error) {
	current := s.Status(ctx)
	if !current.SteamCMD.Detected {
		return MigrationResult{}, errors.New("当前没有可迁移的 SteamCMD")
	}
	sourceRoot := filepath.Clean(current.SteamCMD.Root)
	targetRoot := s.absolutePath(request.TargetRoot)
	if targetRoot == "" || targetRoot == "." {
		return MigrationResult{}, errors.New("请选择新的 SteamCMD 目录")
	}
	result, err := copyTree(ctx, sourceRoot, targetRoot)
	if err != nil {
		return result, err
	}
	exe := filepath.Join(targetRoot, managedSteamCMDExecutableName())
	if !regularFileExists(exe) {
		exe = filepath.Join(targetRoot, executableName())
	}
	if !regularFileExists(exe) {
		return result, errors.New("迁移校验失败：目标目录没有 steamcmd 可执行文件")
	}
	next := s.state
	next.Version = 3
	next.SteamCMDRoot = targetRoot
	next.SteamCMDPath = exe
	if err := s.replaceState(next); err != nil {
		return result, err
	}
	if request.RemoveSource && !samePath(sourceRoot, targetRoot) {
		if err := os.RemoveAll(sourceRoot); err != nil {
			return result, fmt.Errorf("新目录已启用，但删除旧 SteamCMD 失败: %w", err)
		}
		result.RemovedOld = true
	}
	result.Message = "SteamCMD 已迁移并更新 AI Game Manager Panel 配置"
	return result, nil
}

func (s *Service) MigrateGameLibrary(ctx context.Context, request MigratePathRequest) (MigrationResult, error) {
	current := s.Status(ctx)
	sourceRoot := filepath.Clean(current.Paths.GameLibraryRoot)
	targetRoot := s.absolutePath(request.TargetRoot)
	if targetRoot == "" || targetRoot == "." {
		return MigrationResult{}, errors.New("请选择新的游戏服务器库目录")
	}
	if samePath(sourceRoot, targetRoot) {
		return MigrationResult{Source: sourceRoot, Target: targetRoot, Message: "目标目录与当前目录相同"}, nil
	}
	result := MigrationResult{Source: sourceRoot, Target: targetRoot}
	if _, err := os.Stat(sourceRoot); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(targetRoot, 0o755); err != nil {
			return result, err
		}
	} else if err != nil {
		return result, err
	} else {
		var err error
		result, err = copyTree(ctx, sourceRoot, targetRoot)
		if err != nil {
			return result, err
		}
	}
	next := s.state
	next.Version = 3
	next.GameLibraryRoot = targetRoot
	next.DefaultInstallRoot = ""
	if err := s.replaceState(next); err != nil {
		return result, err
	}
	if request.RemoveSource {
		if err := os.RemoveAll(sourceRoot); err != nil {
			return result, fmt.Errorf("新游戏库已启用，但删除旧目录失败: %w", err)
		}
		result.RemovedOld = true
	}
	result.Message = "游戏服务器库已迁移；后续安装、更新和启动必须读取新的 GameLibraryRoot"
	return result, nil
}

func (s *Service) resolvePaths(state persistedState) StoragePaths {
	gameRoot := firstNonEmpty(state.GameLibraryRoot, state.DefaultInstallRoot, s.defaultGameLibraryRoot())
	steamRoot := firstNonEmpty(state.SteamCMDRoot, s.defaultSteamCMDRoot())
	instanceRoot := firstNonEmpty(state.InstanceConfigRoot, s.options.InstanceDir)
	cacheRoot := firstNonEmpty(state.CacheRoot, s.options.CacheDir)
	return StoragePaths{
		ProgramDir: filepath.Clean(s.options.Root), DataDir: filepath.Clean(s.options.DataDir), RuntimeRoot: s.runtimeRoot(),
		SteamCMDRoot: s.absolutePath(steamRoot), SteamCMDPath: strings.TrimSpace(state.SteamCMDPath),
		GameLibraryRoot: s.absolutePath(gameRoot), InstanceConfigRoot: s.absolutePath(instanceRoot),
		GameSaveRoot: strings.TrimSpace(state.GameSaveRoot), CacheRoot: s.absolutePath(cacheRoot),
	}
}

func (s *Service) stateFromPaths(paths StoragePaths) persistedState {
	steamCMDPath := strings.TrimSpace(paths.SteamCMDPath)
	if steamCMDPath != "" {
		steamCMDPath = filepath.Clean(steamCMDPath)
	}
	return persistedState{Version: 4, SteamCMDPath: steamCMDPath, SteamCMDRoot: filepath.Clean(paths.SteamCMDRoot), GameLibraryRoot: filepath.Clean(paths.GameLibraryRoot), InstanceConfigRoot: filepath.Clean(paths.InstanceConfigRoot), GameSaveRoot: strings.TrimSpace(paths.GameSaveRoot), CacheRoot: filepath.Clean(paths.CacheRoot)}
}

func (s *Service) defaultWindowsVolume() string {
	candidates := []string{s.options.Root, s.options.DataDir}
	if systemDrive := strings.TrimSpace(os.Getenv("SystemDrive")); systemDrive != "" {
		candidates = append(candidates, systemDrive+string(os.PathSeparator))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		candidates = append(candidates, home)
	}
	if temp := os.TempDir(); temp != "" {
		candidates = append(candidates, temp)
	}
	for _, candidate := range candidates {
		if volume := filepath.VolumeName(filepath.Clean(candidate)); volume != "" {
			return volume
		}
	}
	return ""
}

func (s *Service) defaultSteamCMDRoot() string {
	return filepath.Join(s.runtimeRoot(), "steamcmd")
}

func (s *Service) defaultGameLibraryRoot() string {
	if runtime.GOOS == "windows" {
		if volume := s.defaultWindowsVolume(); volume != "" {
			return volume + string(os.PathSeparator) + "GameServers"
		}
		return filepath.Join(s.options.DataDir, "games")
	}
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		return filepath.Join(filepath.Dir(configDir), "AI Game Manager Panel", "games")
	}
	return filepath.Join(s.options.DataDir, "games")
}

func (s *Service) requiredDirs(paths StoragePaths) []string {
	values := []string{s.options.DataDir, paths.InstanceConfigRoot, s.options.BackupDir, s.options.TempDir, s.options.ExportDir, s.options.PluginDir, paths.CacheRoot, paths.GameLibraryRoot}
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = filepath.Clean(strings.TrimSpace(value))
		if value == "." || value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	return result
}

func (s *Service) detectSteamCMD(preferred, preferredRoot string) ToolStatus {
	type candidate struct{ path, source string }
	candidates := []candidate{}
	if strings.TrimSpace(preferred) != "" {
		candidates = append(candidates, candidate{preferred, "saved"})
	}
	if strings.TrimSpace(preferredRoot) != "" {
		candidates = append(candidates, candidate{filepath.Join(preferredRoot, executableName()), "configured-root"}, candidate{filepath.Join(preferredRoot, managedSteamCMDExecutableName()), "configured-root"})
	}
	candidates = append(candidates, candidate{filepath.Join(s.defaultSteamCMDRoot(), executableName()), "same-volume-default"}, candidate{filepath.Join(s.defaultSteamCMDRoot(), managedSteamCMDExecutableName()), "managed-default"})
	if runtime.GOOS == "windows" {
		for drive := 'C'; drive <= 'Z'; drive++ {
			candidates = append(candidates, candidate{fmt.Sprintf(`%c:\SteamCMD\steamcmd.exe`, drive), "common-path"})
		}
	} else {
		for _, path := range []string{"/usr/games/steamcmd", "/usr/local/bin/steamcmd", "/opt/steamcmd/steamcmd"} {
			candidates = append(candidates, candidate{path, "common-path"})
		}
	}
	for _, name := range []string{executableName(), "steamcmd", "steamcmd.sh"} {
		if path := findExecutableInPATH(name); path != "" {
			candidates = append(candidates, candidate{path, "PATH"})
		}
	}
	seen := map[string]struct{}{}
	for _, c := range candidates {
		path := filepath.Clean(strings.TrimSpace(c.path))
		if path == "." || path == "" {
			continue
		}
		key := strings.ToLower(path)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			abs, _ := filepath.Abs(path)
			if abs != "" {
				path = abs
			}
			return ToolStatus{Detected: true, Path: path, Root: filepath.Dir(path), Source: c.source}
		}
	}
	return ToolStatus{Detected: false, Root: s.absolutePath(firstNonEmpty(preferredRoot, s.defaultSteamCMDRoot()))}
}

func findExecutableInPATH(name string) string {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func executableName() string {
	if runtime.GOOS == "windows" {
		return "steamcmd.exe"
	}
	return "steamcmd"
}

func managedSteamCMDExecutableName() string {
	if runtime.GOOS == "windows" {
		return "steamcmd.exe"
	}
	return "steamcmd.sh"
}

func (s *Service) absolutePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if filepath.IsAbs(value) {
		return filepath.Clean(value)
	}
	return filepath.Clean(filepath.Join(s.options.Root, value))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
func samePath(a, b string) bool { return strings.EqualFold(filepath.Clean(a), filepath.Clean(b)) }

func copyTree(ctx context.Context, sourceRoot, targetRoot string) (MigrationResult, error) {
	result := MigrationResult{Source: sourceRoot, Target: targetRoot}
	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return result, err
	}
	err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rel, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		destination := filepath.Join(targetRoot, rel)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("迁移目录包含符号链接，已停止：%s", path)
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			in.Close()
			return err
		}
		written, copyErr := io.Copy(out, in)
		closeIn := in.Close()
		closeOut := out.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeIn != nil {
			return closeIn
		}
		if closeOut != nil {
			return closeOut
		}
		result.CopiedFiles++
		result.CopiedBytes += written
		return nil
	})
	if err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) replaceState(next persistedState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := s.state
	s.state = next
	if err := s.saveLocked(); err != nil {
		s.state = previous
		return err
	}
	return nil
}

func (s *Service) load() error {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.state)
}

func (s *Service) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := platformfiles.AtomicReplace(temp, s.path); err != nil {
		_ = os.Remove(temp)
		return err
	}
	return nil
}
