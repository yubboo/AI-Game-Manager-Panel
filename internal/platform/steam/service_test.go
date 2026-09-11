package steam

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseLibraryFolderPaths(t *testing.T) {
	content := `"libraryfolders"
{
    "0"
    {
        "path"        "C:\\Program Files (x86)\\Steam"
    }
    "1"
    {
        "path"        "D:\\SteamLibrary"
    }
    "2" "E:\\LegacySteamLibrary"
    "12345" "1234567890"
}`
	got := parseLibraryFolderPaths(content)
	if len(got) != 3 {
		t.Fatalf("expected 3 library paths, got %d: %#v", len(got), got)
	}
	if got[1] != `D:\SteamLibrary` {
		t.Fatalf("unexpected second path: %q", got[1])
	}
	if got[2] != `E:\LegacySteamLibrary` {
		t.Fatalf("unexpected legacy path: %q", got[2])
	}
}

func TestDetectLibrariesAndFindApp(t *testing.T) {
	root := t.TempDir()
	secondary := filepath.Join(t.TempDir(), "SteamLibrary")
	for _, dir := range []string{
		filepath.Join(root, "steamapps"),
		filepath.Join(secondary, "steamapps", "common", "Example Dedicated Server"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	vdf := fmt.Sprintf(`"libraryfolders"
{
    "0" { "path" "%s" }
    "1"
    {
        "path" "%s"
    }
}`, escapeVDF(root), escapeVDF(secondary))
	if err := os.WriteFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"), []byte(vdf), 0o644); err != nil {
		t.Fatal(err)
	}

	manifest := `"AppState"
{
    "appid" "12345"
    "name" "Example Dedicated Server"
    "installdir" "Example Dedicated Server"
}`
	manifestPath := filepath.Join(secondary, "steamapps", "appmanifest_12345.acf")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	service := newWithRoots(root)
	env, err := service.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !env.Detected {
		t.Fatal("expected Steam to be detected")
	}
	if len(env.Libraries) != 2 {
		t.Fatalf("expected 2 libraries, got %d: %#v", len(env.Libraries), env.Libraries)
	}

	app, found, err := service.FindApp(context.Background(), 12345)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected app to be found")
	}
	if app.Name != "Example Dedicated Server" {
		t.Fatalf("unexpected app name: %q", app.Name)
	}
	wantInstall := filepath.Join(secondary, "steamapps", "common", "Example Dedicated Server")
	if app.InstallPath != wantInstall {
		t.Fatalf("install path = %q, want %q", app.InstallPath, wantInstall)
	}
}

func TestSteamMissingIsNotError(t *testing.T) {
	service := newWithRoots(filepath.Join(t.TempDir(), "missing"))
	env, err := service.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if env.Detected {
		t.Fatal("missing Steam should be a normal detected=false state")
	}
	if len(env.Libraries) != 0 {
		t.Fatalf("expected no libraries, got %#v", env.Libraries)
	}
}

func escapeVDF(path string) string {
	out := ""
	for _, r := range path {
		if r == '\\' {
			out += `\\`
		} else {
			out += string(r)
		}
	}
	return out
}

func TestAppsScansMetadataAcrossLibraries(t *testing.T) {
	root := t.TempDir()
	secondary := filepath.Join(t.TempDir(), "LibraryTwo")
	for _, dir := range []string{
		filepath.Join(root, "steamapps", "common", "Alpha Server"),
		filepath.Join(secondary, "steamapps", "common", "Zulu Server"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	vdf := fmt.Sprintf(`"libraryfolders"
{
    "0"
    {
        "path" "%s"
    }
    "1"
    {
        "path" "%s"
    }
}`, escapeVDF(root), escapeVDF(secondary))
	if err := os.WriteFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"), []byte(vdf), 0o644); err != nil {
		t.Fatal(err)
	}

	alpha := `"AppState"
{
    "appid" "100"
    "name" "Alpha Server"
    "StateFlags" "4"
    "installdir" "Alpha Server"
    "LastUpdated" "1700000000"
    "SizeOnDisk" "123456"
    "buildid" "777"
}`
	zulu := `"AppState"
{
    "appid" "200"
    "name" "Zulu Server"
    "StateFlags" "4"
    "installdir" "Zulu Server"
    "LastUpdated" "1800000000"
    "SizeOnDisk" "654321"
    "buildid" "888"
}`
	if err := os.WriteFile(filepath.Join(root, "steamapps", "appmanifest_100.acf"), []byte(alpha), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondary, "steamapps", "appmanifest_200.acf"), []byte(zulu), 0o644); err != nil {
		t.Fatal(err)
	}

	inventory, err := newWithRoots(root).Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !inventory.Detected {
		t.Fatal("expected Steam inventory to be detected")
	}
	if len(inventory.Apps) != 2 {
		t.Fatalf("expected 2 apps, got %d: %#v", len(inventory.Apps), inventory.Apps)
	}
	if inventory.Apps[0].AppID != 100 || inventory.Apps[1].AppID != 200 {
		t.Fatalf("apps were not sorted by name: %#v", inventory.Apps)
	}
	if inventory.Apps[0].BuildID != "777" || inventory.Apps[0].LastUpdated != 1700000000 || inventory.Apps[0].SizeOnDisk != 123456 {
		t.Fatalf("manifest metadata was not parsed: %#v", inventory.Apps[0])
	}
	if !inventory.Apps[0].InstallPathExists || !inventory.Apps[1].InstallPathExists {
		t.Fatalf("expected both install directories to exist: %#v", inventory.Apps)
	}
}

func TestAppsKeepsScanningWhenManifestHasBadFields(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "steamapps", "common", "Good App"), 0o755); err != nil {
		t.Fatal(err)
	}

	good := `"AppState"
{
    "appid" "10"
    "name" "Good App"
    "installdir" "Good App"
}`
	bad := `"AppState"
{
    "appid" "20"
    "name" "Needs Attention"
    "installdir" "../outside"
    "LastUpdated" "not-a-number"
}`
	if err := os.WriteFile(filepath.Join(root, "steamapps", "appmanifest_10.acf"), []byte(good), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "steamapps", "appmanifest_20.acf"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	inventory, err := newWithRoots(root).Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Apps) != 2 {
		t.Fatalf("a bad field should not abort the whole inventory, got %#v", inventory.Apps)
	}
	if len(inventory.Warnings) < 2 {
		t.Fatalf("expected validation warnings, got %#v", inventory.Warnings)
	}
	var suspicious AppInstallation
	for _, app := range inventory.Apps {
		if app.AppID == 20 {
			suspicious = app
		}
	}
	if suspicious.InstallPath != "" || suspicious.InstallPathExists {
		t.Fatalf("unsafe installdir should not become an install path: %#v", suspicious)
	}
}

func TestDuplicateAppPrefersExistingInstallDirectory(t *testing.T) {
	root := t.TempDir()
	secondary := filepath.Join(t.TempDir(), "Second")
	if err := os.MkdirAll(filepath.Join(root, "steamapps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(secondary, "steamapps", "common", "Real Copy"), 0o755); err != nil {
		t.Fatal(err)
	}
	vdf := fmt.Sprintf(`"libraryfolders"
{
    "0" { }
    "1"
    {
        "path" "%s"
    }
}`, escapeVDF(secondary))
	if err := os.WriteFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"), []byte(vdf), 0o644); err != nil {
		t.Fatal(err)
	}

	stale := `"AppState"
{
    "appid" "42"
    "name" "Stale Copy"
    "installdir" "Missing Copy"
}`
	real := `"AppState"
{
    "appid" "42"
    "name" "Real Copy"
    "installdir" "Real Copy"
}`
	if err := os.WriteFile(filepath.Join(root, "steamapps", "appmanifest_42.acf"), []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondary, "steamapps", "appmanifest_42.acf"), []byte(real), 0o644); err != nil {
		t.Fatal(err)
	}

	inventory, err := newWithRoots(root).Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory.Apps) != 1 {
		t.Fatalf("expected duplicate app id to be collapsed, got %#v", inventory.Apps)
	}
	if inventory.Apps[0].Name != "Real Copy" || !inventory.Apps[0].InstallPathExists {
		t.Fatalf("expected real installation to win: %#v", inventory.Apps[0])
	}
}

func TestSafeInstallDirRejectsTraversalAndAbsolutePaths(t *testing.T) {
	invalid := []string{
		"",
		".",
		"..",
		"../outside",
		`..\outside`,
		`C:\outside`,
		`\\server\share`,
		"/outside",
	}
	for _, value := range invalid {
		if safeInstallDir(value) {
			t.Fatalf("expected unsafe install dir to be rejected: %q", value)
		}
	}
	valid := []string{"Example Dedicated Server", "Server Folder", "Games/Subdir"}
	for _, value := range valid {
		if !safeInstallDir(value) {
			t.Fatalf("expected install dir to be accepted: %q", value)
		}
	}
}

func TestSnapshotDiscoversSteamOnlyOnce(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "steamapps", "common", "Example App"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `"AppState"
{
    "appid" "77"
    "name" "Example App"
    "installdir" "Example App"
}`
	if err := os.WriteFile(filepath.Join(root, "steamapps", "appmanifest_77.acf"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	calls := 0
	service := &Service{candidateRoots: func(context.Context) []string {
		calls++
		return []string{root}
	}}

	snapshot, err := service.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("snapshot should discover Steam exactly once, candidateRoots calls = %d", calls)
	}
	if !snapshot.Environment.Detected || !snapshot.Inventory.Detected {
		t.Fatalf("expected detected snapshot, got %#v", snapshot)
	}
	if len(snapshot.Inventory.Apps) != 1 || snapshot.Inventory.Apps[0].AppID != 77 {
		t.Fatalf("unexpected snapshot inventory: %#v", snapshot.Inventory)
	}
	if snapshot.ScannedAt <= 0 || snapshot.DurationMs < 0 {
		t.Fatalf("invalid snapshot timing metadata: %#v", snapshot)
	}
}

func TestDecodeRegistryUTF16LE(t *testing.T) {
	// "D:\\Steam" encoded as UTF-16LE with a terminating NUL.
	data := []byte{0x44, 0x00, 0x3a, 0x00, 0x5c, 0x00, 0x53, 0x00, 0x74, 0x00, 0x65, 0x00, 0x61, 0x00, 0x6d, 0x00, 0x00, 0x00}
	if got := decodeRegistryUTF16LE(data); got != `D:\Steam` {
		t.Fatalf("decoded registry string = %q", got)
	}
}

func TestFindAppsOnlyReadsRequestedAppIDs(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "steamapps", "common", "Wanted Game"),
		filepath.Join(root, "steamapps", "common", "Wanted Server"),
		filepath.Join(root, "steamapps", "common", "Unrelated App"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	manifests := map[string]string{
		"appmanifest_322330.acf": `"AppState" { "appid" "322330" "name" "Wanted Game" "installdir" "Wanted Game" }`,
		"appmanifest_343050.acf": `"AppState" { "appid" "343050" "name" "Wanted Server" "installdir" "Wanted Server" }`,
		"appmanifest_999999.acf": `"AppState" { "appid" "999999" "name" "Unrelated App" "installdir" "Unrelated App" }`,
	}
	for name, content := range manifests {
		if err := os.WriteFile(filepath.Join(root, "steamapps", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	snapshot, err := newWithRoots(root).FindApps(context.Background(), []AppID{322330, 343050})
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Environment.Detected {
		t.Fatal("expected Steam to be detected")
	}
	if len(snapshot.Apps) != 2 {
		t.Fatalf("expected only requested apps, got %#v", snapshot.Apps)
	}
	for _, app := range snapshot.Apps {
		if app.AppID == 999999 {
			t.Fatal("unrelated app should not be loaded into targeted snapshot")
		}
	}
}
