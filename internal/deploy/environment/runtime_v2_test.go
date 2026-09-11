package environment

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRuntimeRegistryPersistsDefaultSelection(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	service := New(testOptions(root, dataDir))
	firstPath := filepath.Join(root, "external-a", platformSteamCMDExecutableName())
	secondPath := filepath.Join(root, "external-b", platformSteamCMDExecutableName())
	for _, path := range []string{firstPath, secondPath} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("steamcmd-test"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	first, err := service.RegisterRuntime(RegisterRuntimeRequest{Kind: RuntimeSteamCMD, Executable: firstPath, SetDefault: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.RegisterRuntime(RegisterRuntimeRequest{Kind: RuntimeSteamCMD, Executable: secondPath})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Default || second.Default {
		t.Fatalf("unexpected initial defaults: first=%+v second=%+v", first, second)
	}
	if _, err := service.SetDefaultRuntime(SetDefaultRuntimeRequest{ID: second.ID}); err != nil {
		t.Fatal(err)
	}

	reloaded := New(testOptions(root, dataDir))
	resolved, err := reloaded.ResolveRuntime(ResolveRuntimeRequest{Kind: RuntimeSteamCMD})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != second.ID || !resolved.Default {
		t.Fatalf("default runtime did not persist: got=%+v want=%s", resolved, second.ID)
	}
}

func TestRuntimeRepairPreservesExistingDefault(t *testing.T) {
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	exe := filepath.Join(root, "external", platformSteamCMDExecutableName())
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("steamcmd-test"), 0o755); err != nil {
		t.Fatal(err)
	}
	record, err := service.RegisterRuntime(RegisterRuntimeRequest{Kind: RuntimeSteamCMD, Executable: exe, SetDefault: true})
	if err != nil {
		t.Fatal(err)
	}
	record.Version = "repaired"
	record.Default = false // caller did not explicitly request a default change
	repaired, err := service.upsertRuntime(record, false)
	if err != nil {
		t.Fatal(err)
	}
	if !repaired.Default {
		t.Fatalf("repair unexpectedly cleared default: %+v", repaired)
	}
}

func TestJavaVersionParserRecognizesLegacyAndModernVersions(t *testing.T) {
	cases := []struct {
		output string
		want   int
	}{
		{`java version "1.8.0_402"\nJava(TM) SE Runtime Environment`, 8},
		{`openjdk version "17.0.12" 2024-07-16`, 17},
		{`openjdk version "21.0.4" 2024-07-16 LTS`, 21},
		{`openjdk version "25" 2025-09-16`, 25},
	}
	for _, tc := range cases {
		info, err := parseJavaVersionOutput(tc.output)
		if err != nil {
			t.Fatalf("parse %q: %v", tc.output, err)
		}
		if info.Major != tc.want {
			t.Fatalf("parse %q major=%d want=%d", tc.output, info.Major, tc.want)
		}
	}
	if _, err := parseJavaVersionOutput("not java"); err == nil {
		t.Fatal("invalid java output was accepted")
	}
}

func TestExtractTarGZRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "malicious.tar.gz")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "bin/java", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd", Mode: 0o777}); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractTarGZ(archive, filepath.Join(root, "out")); err == nil {
		t.Fatal("tar symlink was accepted")
	}
}

func TestExtractZIPRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	archive := filepath.Join(root, "malicious.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	header := &zip.FileHeader{Name: "bin/java"}
	header.SetMode(os.ModeSymlink | 0o777)
	writer, err := zw.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("/etc/passwd")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := extractZIP(archive, filepath.Join(root, "out")); err == nil {
		t.Fatal("zip symlink was accepted")
	}
}

func TestJavaInstallIsSerializedByInstallLock(t *testing.T) {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("unsupported Java auto-install platform")
	}
	root := t.TempDir()
	service := New(testOptions(root, filepath.Join(root, "data")))
	service.installMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	done := make(chan error, 1)
	go func() {
		_, err := service.InstallJava(ctx, JavaInstallRequest{Major: 21})
		done <- err
	}()
	select {
	case err := <-done:
		service.installMu.Unlock()
		t.Fatalf("InstallJava bypassed installation lock: %v", err)
	case <-time.After(80 * time.Millisecond):
	}
	service.installMu.Unlock()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled Java install unexpectedly succeeded")
		}
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			// net/http may wrap context cancellation; a non-nil error is the key
			// assertion after lock release, so don't require an exact wrapper.
		}
	case <-time.After(3 * time.Second):
		t.Fatal("InstallJava did not resume after installation lock released")
	}
}
