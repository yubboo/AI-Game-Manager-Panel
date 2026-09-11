package dedicated

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/platform/steam"
)

func makeInstall(t *testing.T, root string, bitness ...int) string {
	t.Helper()
	install := filepath.Join(root, InstallDirName)
	for _, bits := range bitness {
		var bin, exe string
		if bits == 64 {
			bin, exe = BinDirAMD64, ExecutableAMD64
		} else {
			bin, exe = BinDir386, Executable386
		}
		if err := os.MkdirAll(filepath.Join(install, bin), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(install, bin, exe), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return install
}

func TestValidateInstallDirRejectsNormalGameClient(t *testing.T) {
	root := t.TempDir()
	client := filepath.Join(root, "Don't Starve Together")
	if err := os.MkdirAll(filepath.Join(client, BinDirAMD64), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(client, BinDirAMD64, ExecutableAMD64), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ValidateInstallDir(client)
	if got.Valid {
		t.Fatal("normal game client must not be accepted as Dedicated Server")
	}
}

func TestPickBitnessPrefers64(t *testing.T) {
	install := makeInstall(t, t.TempDir(), 32, 64)
	bits, bin, exe, err := PickBitness(install)
	if err != nil {
		t.Fatal(err)
	}
	if bits != 64 || filepath.Base(bin) != BinDirAMD64 || filepath.Base(exe) != ExecutableAMD64 {
		t.Fatalf("unexpected selection: bits=%d bin=%q exe=%q", bits, bin, exe)
	}
}

func TestDiscoverInstallManualPathWins(t *testing.T) {
	manual := makeInstall(t, filepath.Join(t.TempDir(), "manual"), 64)
	steamRoot := filepath.Join(t.TempDir(), "steamlib")
	_ = makeInstall(t, filepath.Join(steamRoot, "steamapps", "common", "..", "..", ".."), 64)
	// Construct a real library candidate at the exact Steam common location.
	steamInstall := filepath.Join(steamRoot, "steamapps", "common", InstallDirName)
	if err := os.MkdirAll(filepath.Join(steamInstall, BinDirAMD64), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(steamInstall, BinDirAMD64, ExecutableAMD64), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, _ := DiscoverInstall(manual, []steam.Library{{Path: steamRoot}})
	if !got.Valid || got.Source != InstallSourceManual || got.RootDir != filepath.Clean(manual) {
		t.Fatalf("manual path should win: %+v", got)
	}
}

func TestDiscoverInstallFallsBackToSteamWhenManualInvalid(t *testing.T) {
	steamRoot := t.TempDir()
	install := filepath.Join(steamRoot, "steamapps", "common", InstallDirName)
	if err := os.MkdirAll(filepath.Join(install, BinDir386), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(install, BinDir386, Executable386), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, warnings := DiscoverInstall(filepath.Join(t.TempDir(), "missing"), []steam.Library{{Path: steamRoot}})
	if !got.Valid || got.Source != InstallSourceSteam || got.Bitness != 32 {
		t.Fatalf("expected Steam fallback: %+v", got)
	}
	if len(warnings) == 0 {
		t.Fatal("invalid remembered path should produce a warning")
	}
}

func TestResolveConfDirDefaultOmitted(t *testing.T) {
	documents := t.TempDir()
	root := filepath.Join(documents, "Klei", "DoNotStarveTogether")
	arg, err := ResolveConfDirArg(documents, root)
	if err != nil || arg != "" {
		t.Fatalf("default root should omit -conf_dir: arg=%q err=%v", arg, err)
	}
}

func TestResolveConfDirCustomRelative(t *testing.T) {
	documents := t.TempDir()
	root := filepath.Join(documents, "Klei", "CustomDST")
	arg, err := ResolveConfDirArg(documents, root)
	if err != nil {
		t.Fatal(err)
	}
	if arg != "CustomDST" {
		t.Fatalf("unexpected conf_dir: %q", arg)
	}
}

func TestResolveConfDirCrossDrive(t *testing.T) {
	_, err := ResolveConfDirArg(`C:\Users\Tester\Documents`, `D:\Klei\DoNotStarveTogether`)
	if err != ErrConfDirCrossDrive {
		t.Fatalf("expected cross-drive error, got %v", err)
	}
}

func TestBuildLaunchArgsOrderingAndQuotes(t *testing.T) {
	got, err := BuildLaunchArgs("Cluster_1", "Master", "CustomDST", `D:\Steam\steamapps\workshop`, `-foo "hello world" -bar 1`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"-conf_dir", "CustomDST",
		"-cluster", "Cluster_1", "-shard", "Master",
		"-ugc_directory", `D:\Steam\steamapps\workshop`,
		"-foo", "hello world", "-bar", "1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestBuildLaunchArgsDropsDeprecatedConsoleFromExtraArgs(t *testing.T) {
	got, err := BuildLaunchArgs("Cluster_1", "Master", "", "", `-console -foo 1`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-cluster", "Cluster_1", "-shard", "Master", "-foo", "1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deprecated -console should be removed: got=%#v want=%#v", got, want)
	}
}

func TestBuildLaunchArgsRejectsUnclosedQuote(t *testing.T) {
	if _, err := BuildLaunchArgs("Cluster_1", "Master", "", "", `-foo "broken`); err == nil {
		t.Fatal("expected malformed extra args error")
	}
}

func TestAdvanceWorldReadyMarkerMasterIgnoresReset(t *testing.T) {
	start, ready := AdvanceWorldReadyMarker("Reset() returning", true, false)
	if start || ready {
		t.Fatal("pre-start Reset() must not count")
	}
	start, ready = AdvanceWorldReadyMarker("About to start a shard with these settings:", true, start)
	if !start || ready {
		t.Fatal("start marker should start tracking without ready")
	}
	start, ready = AdvanceWorldReadyMarker("Reset() returning", true, start)
	if !start || ready {
		t.Fatal("Reset() after start is still not Master ready")
	}
	_, ready = AdvanceWorldReadyMarker("Sim paused", true, start)
	if !ready {
		t.Fatal("Sim paused should mark Master ready")
	}
}

func TestAdvanceWorldReadyMarkerSingleWorldAndSecondary(t *testing.T) {
	start, ready := AdvanceWorldReadyMarker("About to start a server with the following settings:", true, false)
	if !start || ready {
		t.Fatal("single-world start marker not recognized")
	}
	_, ready = AdvanceWorldReadyMarker("DST_Master_Ready", true, start)
	if !ready {
		t.Fatal("legacy master marker not recognized")
	}

	secondaryStart, secondaryReady := AdvanceWorldReadyMarker("About to start a shard with these settings:", false, false)
	if !secondaryStart || secondaryReady {
		t.Fatal("secondary start state mismatch")
	}
	_, secondaryReady = AdvanceWorldReadyMarker("[Shard] secondary shard LUA is now ready!", false, secondaryStart)
	if !secondaryReady {
		t.Fatal("secondary ready marker not recognized")
	}
}

func TestBuildLaunchSpecUsesValidatedExecutableAndMasterRole(t *testing.T) {
	installDir := makeInstall(t, t.TempDir(), 64)
	install := ValidateInstallDir(installDir)
	documents := t.TempDir()
	kleiRoot := filepath.Join(documents, "Klei", "DoNotStarveTogether")
	spec, err := BuildLaunchSpec(install, documents, LaunchRequest{
		ClusterName: "Cluster_1",
		ShardName:   "Master",
		KleiRoot:    kleiRoot,
		ExtraArgs:   "-foo 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if spec.Role != ShardMaster || spec.Executable != install.Executable || spec.WorkingDirectory != install.BinDir {
		t.Fatalf("unexpected launch spec: %+v", spec)
	}
	want := []string{"-cluster", "Cluster_1", "-shard", "Master", "-foo", "1"}
	if !reflect.DeepEqual(spec.Arguments, want) {
		t.Fatalf("unexpected launch arguments: %#v", spec.Arguments)
	}
}

func TestServerStatusValuesMatchPythonBaseline(t *testing.T) {
	got := []ServerStatus{StatusStarting, StatusRunning, StatusStopping, StatusStopped, StatusCrashed}
	want := []ServerStatus{"starting", "running", "stopping", "stopped", "crashed"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("status values drifted: got=%v want=%v", got, want)
	}
}

func TestExtraArgsMatchPythonShlexNonPosixMidTokenQuotes(t *testing.T) {
	got, err := splitWindowsStyleArgs(`-a="x y"`)
	if err != nil {
		t.Fatal(err)
	}
	// Python: shlex.split(`-a="x y"`, posix=False) => [`-a="x`, `y"`].
	want := []string{`-a="x`, `y"`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("non-posix shlex parity mismatch: got=%#v want=%#v", got, want)
	}
}
