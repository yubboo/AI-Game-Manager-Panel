package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestVersionComparisonUsesNumericSegments(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.73", "0.1.72", 1},
		{"0.1.100", "0.1.99", 1},
		{"0.2.0", "0.1.100", 1},
		{"0.1.73", "0.1.73", 0},
	}
	for _, tc := range cases {
		got, err := compareVersions(tc.a, tc.b)
		if err != nil || got != tc.want {
			t.Fatalf("compare %s %s = %d err=%v want=%d", tc.a, tc.b, got, err, tc.want)
		}
	}
}

func TestCheckFindsStableWindowsInstaller(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			fmt.Fprintf(w, `{"tag_name":"v0.1.73","name":"0.1.73","body":"更新说明","html_url":"https://github.com/yubboo/AI-Game-Manager-Panel/releases/tag/v0.1.73","draft":false,"prerelease":false,"assets":[{"name":"AI-Game-Manager-Panel-0.1.73-Windows-x64-Setup.exe","browser_download_url":"https://github.com/yubboo/AI-Game-Manager-Panel/releases/download/v0.1.73/setup.exe","size":123,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	s := New(Options{CurrentVersion: "0.1.72", CacheDir: t.TempDir(), Config: Config{Enabled: true, Provider: "github-releases", Repository: "yubboo/AI-Game-Manager-Panel", APIBaseURL: server.URL, AssetPattern: "AI-Game-Manager-Panel-{version}-Windows-x64-Setup.exe", RequireSHA256: true}, HTTPClient: server.Client()})
	status, err := s.Check(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if !status.UpdateAvailable || status.LatestVersion != "0.1.73" || status.InstallerSHA256 == "" {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestCheckIgnoresOlderRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v0.1.72","name":"same","body":"","html_url":"https://github.com/yubboo/AI-Game-Manager-Panel","draft":false,"prerelease":false,"assets":[]}`)
	}))
	defer server.Close()
	s := New(Options{CurrentVersion: "0.1.73", CacheDir: t.TempDir(), Config: Config{Enabled: true, Provider: "github-releases", Repository: "yubboo/AI-Game-Manager-Panel", APIBaseURL: server.URL}, HTTPClient: server.Client()})
	status, err := s.Check(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	if status.UpdateAvailable {
		t.Fatalf("older/same release must not update: %+v", status)
	}
}

func TestDownloadRejectsWrongSHA256(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("installer")) }))
	defer server.Close()
	dest := filepath.Join(t.TempDir(), "setup.exe")
	err := downloadFile(context.Background(), server.Client(), server.URL, dest, int64(len("installer")), strings.Repeat("0", 64))
	if err == nil {
		t.Fatal("expected sha256 rejection")
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Fatalf("destination should not exist: %v", statErr)
	}
}

func TestLaunchInstallerPlatformGuard(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows guard test")
	}
	s := New(Options{})
	if err := s.LaunchInstaller(PreparedUpdate{InstallerPath: "x.exe"}); err == nil {
		t.Fatal("expected Windows-only error")
	}
}
