package platformfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRootPrefersProjectCWD(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := resolveRoot(root, filepath.Join(t.TempDir(), "AI-Game-Manager-Panel.exe"))
	if filepath.Clean(got) != filepath.Clean(root) {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestResolveRootFindsProjectAboveBuildBin(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "build", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}

	got := resolveRoot(t.TempDir(), filepath.Join(bin, "AI-Game-Manager-Panel.exe"))
	if filepath.Clean(got) != filepath.Clean(root) {
		t.Fatalf("got %q want %q", got, root)
	}
}

func TestResolveRootPortableFallsBackToExecutableDir(t *testing.T) {
	dir := t.TempDir()
	got := resolveRoot(t.TempDir(), filepath.Join(dir, "AI-Game-Manager-Panel.exe"))
	if filepath.Clean(got) != filepath.Clean(dir) {
		t.Fatalf("got %q want %q", got, dir)
	}
}

func TestResolveRootUsesEnvironmentOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "AI Game Manager PanelData")
	t.Setenv("AGMP_ROOT", want)
	if got := ResolveRoot(); got != filepath.Clean(want) {
		t.Fatalf("AGMP_ROOT override = %q, want %q", got, want)
	}
}
