package environment

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInitializationWithExplicitSteamCMDPersists(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	steamCMD := filepath.Join(root, "fixtures", platformSteamCMDExecutableName())
	if err := os.MkdirAll(filepath.Dir(steamCMD), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(steamCMD, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	options := testOptions(root, dataDir)
	service := New(options)
	gameRoot := filepath.Join(root, "GameServers")
	status, err := service.Initialize(context.Background(), InitializeRequest{SteamCMDPath: steamCMD, GameLibraryRoot: gameRoot})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Initialized || !status.SteamCMD.Detected {
		t.Fatalf("unexpected initialized status: %+v", status)
	}
	if filepath.Clean(status.Paths.GameLibraryRoot) != filepath.Clean(gameRoot) {
		t.Fatalf("unexpected game library: %+v", status.Paths)
	}
	reloaded := New(options).Status(context.Background())
	if !reloaded.Initialized || filepath.Clean(reloaded.Paths.SteamCMDPath) != filepath.Clean(steamCMD) {
		t.Fatalf("unexpected reloaded status: %+v", reloaded)
	}
}

func TestDefaultGameLibraryDoesNotUseInstancesGames(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	status := service.Status(context.Background())
	if strings.Contains(strings.ToLower(filepath.ToSlash(status.Paths.GameLibraryRoot)), "instances/games") {
		t.Fatalf("game library must not default to instances/games: %s", status.Paths.GameLibraryRoot)
	}
	if status.DefaultInstallRoot != status.Paths.GameLibraryRoot {
		t.Fatalf("legacy alias must follow GameLibraryRoot: %+v", status)
	}
}

func TestUpdateStoragePathsPersists(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	service := New(testOptions(root, dataDir))
	gameRoot := filepath.Join(root, "custom games")
	instanceRoot := filepath.Join(root, "instance-config")
	status, err := service.UpdateStoragePaths(context.Background(), StoragePathsRequest{GameLibraryRoot: gameRoot, InstanceConfigRoot: instanceRoot})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(status.Paths.GameLibraryRoot) != filepath.Clean(gameRoot) {
		t.Fatalf("unexpected paths: %+v", status.Paths)
	}
	reloaded := New(testOptions(root, dataDir)).Status(context.Background())
	if filepath.Clean(reloaded.Paths.GameLibraryRoot) != filepath.Clean(gameRoot) || filepath.Clean(reloaded.Paths.InstanceConfigRoot) != filepath.Clean(instanceRoot) {
		t.Fatalf("paths did not persist: %+v", reloaded.Paths)
	}
}

func TestMigrateGameLibraryCopiesAndSwitchesRoot(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	source := filepath.Join(root, "old-games")
	target := filepath.Join(root, "new-games")
	if err := os.MkdirAll(filepath.Join(source, "DST"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "DST", "server.bin"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateStoragePaths(context.Background(), StoragePathsRequest{GameLibraryRoot: source}); err != nil {
		t.Fatal(err)
	}
	result, err := service.MigrateGameLibrary(context.Background(), MigratePathRequest{TargetRoot: target})
	if err != nil {
		t.Fatal(err)
	}
	if result.CopiedFiles != 1 {
		t.Fatalf("unexpected migration result: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(target, "DST", "server.bin")); err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(service.Status(context.Background()).Paths.GameLibraryRoot) != filepath.Clean(target) {
		t.Fatal("new game root was not activated")
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal("source should be retained unless removeSource=true")
	}
}

func TestMigrateGameLibraryFailureKeepsCurrentRoot(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	source := filepath.Join(root, "old-games")
	target := filepath.Join(root, "new-games")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "server.bin"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := service.UpdateStoragePaths(context.Background(), StoragePathsRequest{GameLibraryRoot: source}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.MigrateGameLibrary(ctx, MigratePathRequest{TargetRoot: target}); err == nil {
		t.Fatal("cancelled migration must fail")
	}
	if got := filepath.Clean(service.Status(context.Background()).Paths.GameLibraryRoot); got != filepath.Clean(source) {
		t.Fatalf("failed migration must keep old root active: got %s want %s", got, source)
	}
	if _, err := os.Stat(filepath.Join(source, "server.bin")); err != nil {
		t.Fatalf("failed migration must keep source usable: %v", err)
	}
}

func TestInitializationDoesNotRequireSteamCMD(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	status, err := service.Initialize(context.Background(), InitializeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Initialized {
		t.Fatal("base Runtime Manager initialization should not require SteamCMD")
	}
	if status.SteamCMD.Detected {
		t.Skip("host environment already provides SteamCMD; initialization independence is still satisfied")
	}
}

func TestGameProfilesDeclareOnlyRelevantDependencies(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	mc := service.GameRuntimeProfile(context.Background(), GameRuntimeProfileRequest{GameID: "minecraft", JavaMajor: 21})
	if len(mc.Requirements) != 1 || mc.Requirements[0].Kind != RuntimeJava || mc.Requirements[0].Major != 21 {
		t.Fatalf("minecraft profile must require only Java 21: %+v", mc)
	}
	dst := service.GameRuntimeProfile(context.Background(), GameRuntimeProfileRequest{GameID: "dst"})
	if len(dst.Requirements) != 1 || dst.Requirements[0].Kind != RuntimeSteamCMD {
		t.Fatalf("DST profile must require SteamCMD: %+v", dst)
	}
}

func TestRuntimeRootIsOwnedByAGMPRoot(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	catalog := service.RuntimeCatalog()
	want := filepath.Join(root, "runtime", "environments")
	if filepath.Clean(catalog.RuntimeRoot) != filepath.Clean(want) {
		t.Fatalf("runtime root=%s want=%s", catalog.RuntimeRoot, want)
	}
}

func TestSkipEnvironmentPersistsAndCanInitializeLater(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	options := testOptions(root, dataDir)
	service := New(options)
	status, err := service.Skip(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !status.Skipped || status.Initialized || status.SkippedAt == 0 {
		t.Fatalf("unexpected skipped status: %+v", status)
	}
	reloaded := New(options)
	status = reloaded.Status(context.Background())
	if !status.Skipped || status.Initialized {
		t.Fatalf("skip state should persist: %+v", status)
	}
	steamCMD := filepath.Join(root, "fixtures", platformSteamCMDExecutableName())
	if err := os.MkdirAll(filepath.Dir(steamCMD), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(steamCMD, []byte("test"), 0o755); err != nil {
		t.Fatal(err)
	}
	status, err = reloaded.Initialize(context.Background(), InitializeRequest{SteamCMDPath: steamCMD})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Initialized || status.Skipped || status.SkippedAt != 0 {
		t.Fatalf("initialization should clear skipped state: %+v", status)
	}
}

func platformSteamCMDExecutableName() string {
	if runtime.GOOS == "windows" {
		return "steamcmd.exe"
	}
	return "steamcmd"
}
func testOptions(root, dataDir string) Options {
	return Options{Root: root, DataDir: dataDir, InstanceDir: filepath.Join(root, "instances"), BackupDir: filepath.Join(root, "backups"), TempDir: filepath.Join(root, "temp"), ExportDir: filepath.Join(root, "exports"), PluginDir: filepath.Join(root, "plugins"), CacheDir: filepath.Join(root, "cache")}
}

func TestSecureJoinRejectsArchiveTraversal(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"../escape", "../../etc/passwd"} {
		if _, err := secureJoin(root, name); err == nil {
			t.Fatalf("secureJoin accepted traversal %q", name)
		}
	}
}

func TestExternalRuntimeRemovalNeverDeletesExternalFile(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	externalRoot := filepath.Join(root, "external")
	if err := os.MkdirAll(externalRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(externalRoot, platformSteamCMDExecutableName())
	if err := os.WriteFile(executable, []byte("external"), 0o755); err != nil {
		t.Fatal(err)
	}
	record, err := service.RegisterRuntime(RegisterRuntimeRequest{Kind: RuntimeSteamCMD, Executable: executable})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RemoveRuntime(RemoveRuntimeRequest{ID: record.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(executable); err != nil {
		t.Fatalf("external runtime file was deleted: %v", err)
	}
}

func TestCustomManagedRuntimeOutsideRuntimeRootIsOnlyUnregistered(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	custom := filepath.Join(root, "custom-managed")
	if err := os.MkdirAll(custom, 0o755); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(custom, platformSteamCMDExecutableName())
	if err := os.WriteFile(executable, []byte("managed"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(custom, ".agmp-runtime.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	record := RuntimeRecord{ID: "custom-managed-test", Kind: RuntimeSteamCMD, Name: "SteamCMD", Root: custom, Executable: executable, Managed: true, Source: "test"}
	if _, err := service.upsertRuntime(record, false); err != nil {
		t.Fatal(err)
	}
	if err := service.RemoveRuntime(RemoveRuntimeRequest{ID: record.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(executable); err != nil {
		t.Fatalf("custom managed path outside RuntimeRoot must not be recursively deleted: %v", err)
	}
}

func TestRuntimeCatalogUsesServerPlatform(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	catalog := service.RuntimeCatalog()
	if catalog.Runtimes == nil || catalog.JavaMajors == nil || catalog.Warnings == nil {
		t.Fatalf("runtime catalog arrays must never serialize as null: %#v", catalog)
	}
	if catalog.Platform != runtime.GOOS || catalog.Architecture != runtime.GOARCH {
		t.Fatalf("catalog platform=%s/%s want=%s/%s", catalog.Platform, catalog.Architecture, runtime.GOOS, runtime.GOARCH)
	}
}
