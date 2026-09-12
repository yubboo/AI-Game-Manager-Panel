package minecraft

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	environment "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/environment"
	serverinstance "github.com/yubboo/AI-Game-Manager-Panel/internal/server/instance"
)

func TestDeployPersistsVerifiedInstanceUsingManagedRuntime(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX shell only to emulate java -version")
	}
	root := t.TempDir()
	artifactBody := []byte("fake paper server jar")
	sum := sha256.Sum256(artifactBody)
	artifactHash := hex.EncodeToString(sum[:])

	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mc/game/version_manifest_v2.json":
			fmt.Fprintf(w, `{"latest":{"release":"1.21.1"},"versions":[{"id":"1.21.1","type":"release","url":%q}]}`, base+"/version.json")
		case "/version.json":
			fmt.Fprint(w, `{"javaVersion":{"majorVersion":21},"downloads":{"server":{"url":"https://example.invalid/vanilla.jar","sha1":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1}}}`)
		case "/projects/paper/versions/1.21.1/builds/latest":
			fmt.Fprintf(w, `{"id":42,"downloads":{"server:default":{"name":"paper.jar","url":%q,"size":%d,"checksums":{"sha256":%q}}}}`, base+"/paper.jar", len(artifactBody), artifactHash)
		case "/paper.jar":
			w.Header().Set("Content-Length", fmt.Sprint(len(artifactBody)))
			_, _ = w.Write(artifactBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	java := filepath.Join(root, "fake-java")
	if err := os.WriteFile(java, []byte("#!/bin/sh\nif [ \"$1\" = \"-version\" ]; then echo 'openjdk version \"21.0.1\"' >&2; exit 0; fi\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "data")
	instancesDir := filepath.Join(root, "instances")
	env := environment.New(environment.Options{
		Root: root, DataDir: dataDir, InstanceDir: instancesDir,
		BackupDir: filepath.Join(root, "backups"), TempDir: filepath.Join(root, "temp"),
		ExportDir: filepath.Join(root, "exports"), PluginDir: filepath.Join(root, "plugins"),
		CacheDir: filepath.Join(root, "cache"), HTTPClient: srv.Client(),
	})
	if _, err := env.RegisterRuntime(environment.RegisterRuntimeRequest{Kind: environment.RuntimeJava, Executable: java, Major: 21, SetDefault: true}); err != nil {
		t.Fatal(err)
	}
	store, err := serverinstance.NewStore(filepath.Join(dataDir, "instances", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	svc := New(Options{
		Root: root, InstancesRoot: instancesDir, HTTPClient: srv.Client(), Environment: env, Store: store,
		Resolver: Resolver{Client: srv.Client(), MojangBase: base, PaperBase: base, FabricBase: base},
	})
	online := true
	result, err := svc.Deploy(context.Background(), PlanRequest{
		Name: "CI Paper", Version: "1.21.1", Software: SoftwarePaper, MemoryMB: 1024, Port: 25565,
		OnlineMode: &online, Whitelist: false, EULAAccepted: true, AutoInstallJava: false, StartAfterDeploy: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Instance.GameID != GameID || result.Instance.GameVersion != "1.21.1" || result.Instance.ServerType != "paper" || result.Instance.RuntimeState != "stopped" {
		t.Fatalf("unexpected instance: %#v", result.Instance)
	}
	jar, err := os.ReadFile(filepath.Join(result.Instance.InstallPath, "server.jar"))
	if err != nil || string(jar) != string(artifactBody) {
		t.Fatalf("verified artifact not persisted: err=%v body=%q", err, string(jar))
	}
	props, err := os.ReadFile(filepath.Join(result.Instance.InstallPath, "server.properties"))
	if err != nil || !strings.Contains(string(props), "online-mode=true") || !strings.Contains(string(props), "server-port=25565") {
		t.Fatalf("unexpected properties: err=%v body=%q", err, string(props))
	}
	eula, err := os.ReadFile(filepath.Join(result.Instance.InstallPath, "eula.txt"))
	if err != nil || !strings.Contains(string(eula), "eula=true") {
		t.Fatalf("EULA output missing: err=%v body=%q", err, string(eula))
	}
	persisted, err := store.Get(result.Instance.ID)
	if err != nil || persisted.InstallPath != result.Instance.InstallPath {
		t.Fatalf("GameInstance not persisted: err=%v instance=%#v", err, persisted)
	}
}

func TestDeployRollsBackInstallWhenInstancePersistenceFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX shell only to emulate java -version")
	}
	root := t.TempDir()
	artifactBody := []byte("fake paper server jar")
	sum := sha256.Sum256(artifactBody)
	artifactHash := hex.EncodeToString(sum[:])
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mc/game/version_manifest_v2.json":
			fmt.Fprintf(w, `{"latest":{"release":"1.21.1"},"versions":[{"id":"1.21.1","type":"release","url":%q}]}`, base+"/version.json")
		case "/version.json":
			fmt.Fprint(w, `{"javaVersion":{"majorVersion":21},"downloads":{"server":{"url":"https://example.invalid/vanilla.jar","sha1":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1}}}`)
		case "/projects/paper/versions/1.21.1/builds/latest":
			fmt.Fprintf(w, `{"id":42,"downloads":{"server:default":{"name":"paper.jar","url":%q,"size":%d,"checksums":{"sha256":%q}}}}`, base+"/paper.jar", len(artifactBody), artifactHash)
		case "/paper.jar":
			_, _ = w.Write(artifactBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	java := filepath.Join(root, "fake-java")
	if err := os.WriteFile(java, []byte("#!/bin/sh\necho 'openjdk version \"21.0.1\"' >&2\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "data")
	instancesDir := filepath.Join(root, "instances")
	env := environment.New(environment.Options{Root: root, DataDir: dataDir, InstanceDir: instancesDir, BackupDir: filepath.Join(root, "backups"), TempDir: filepath.Join(root, "temp"), ExportDir: filepath.Join(root, "exports"), PluginDir: filepath.Join(root, "plugins"), CacheDir: filepath.Join(root, "cache"), HTTPClient: srv.Client()})
	if _, err := env.RegisterRuntime(environment.RegisterRuntimeRequest{Kind: environment.RuntimeJava, Executable: java, Major: 21, SetDefault: true}); err != nil {
		t.Fatal(err)
	}
	storeParent := filepath.Join(root, "broken-store")
	store, err := serverinstance.NewStore(filepath.Join(storeParent, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(storeParent, []byte("not-a-directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	svc := New(Options{Root: root, InstancesRoot: instancesDir, HTTPClient: srv.Client(), Environment: env, Store: store, Resolver: Resolver{Client: srv.Client(), MojangBase: base, PaperBase: base, FabricBase: base}})
	online := true
	plan, err := svc.Plan(context.Background(), PlanRequest{Name: "Rollback Paper", Version: "1.21.1", Software: SoftwarePaper, MemoryMB: 1024, Port: 25565, OnlineMode: &online, EULAAccepted: true})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Deploy(context.Background(), PlanRequest{Name: "Rollback Paper", Version: "1.21.1", Software: SoftwarePaper, MemoryMB: 1024, Port: 25565, OnlineMode: &online, EULAAccepted: true})
	if err == nil {
		t.Fatal("expected GameInstance persistence failure")
	}
	if _, statErr := os.Stat(plan.InstallPath); !os.IsNotExist(statErr) {
		t.Fatalf("orphan install directory was not rolled back: %v", statErr)
	}
	if _, getErr := store.Get(plan.ID); getErr != serverinstance.ErrNotFound {
		t.Fatalf("failed deployment leaked GameInstance into memory: %v", getErr)
	}
}

func TestDeployStopsProcessWhenPingValidationFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a POSIX shell to emulate a long-running Java server")
	}
	root := t.TempDir()
	artifactBody := []byte("fake paper server jar")
	sum := sha256.Sum256(artifactBody)
	artifactHash := hex.EncodeToString(sum[:])
	var base string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mc/game/version_manifest_v2.json":
			fmt.Fprintf(w, `{"latest":{"release":"1.21.1"},"versions":[{"id":"1.21.1","type":"release","url":%q}]}`, base+"/version.json")
		case "/version.json":
			fmt.Fprint(w, `{"javaVersion":{"majorVersion":21},"downloads":{"server":{"url":"https://example.invalid/vanilla.jar","sha1":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","size":1}}}`)
		case "/projects/paper/versions/1.21.1/builds/latest":
			fmt.Fprintf(w, `{"id":42,"downloads":{"server:default":{"name":"paper.jar","url":%q,"size":%d,"checksums":{"sha256":%q}}}}`, base+"/paper.jar", len(artifactBody), artifactHash)
		case "/paper.jar":
			_, _ = w.Write(artifactBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	base = srv.URL

	java := filepath.Join(root, "fake-java")
	script := `#!/bin/sh
if [ "$1" = "-version" ]; then
  echo 'openjdk version "21.0.1"' >&2
  exit 0
fi
echo '[Server thread/INFO]: Done (0.100s)! For help, type "help"'
while IFS= read -r line; do
  if [ "$line" = "stop" ]; then exit 0; fi
done
`
	if err := os.WriteFile(java, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Join(root, "data")
	instancesDir := filepath.Join(root, "instances")
	env := environment.New(environment.Options{Root: root, DataDir: dataDir, InstanceDir: instancesDir, BackupDir: filepath.Join(root, "backups"), TempDir: filepath.Join(root, "temp"), ExportDir: filepath.Join(root, "exports"), PluginDir: filepath.Join(root, "plugins"), CacheDir: filepath.Join(root, "cache"), HTTPClient: srv.Client()})
	if _, err := env.RegisterRuntime(environment.RegisterRuntimeRequest{Kind: environment.RuntimeJava, Executable: java, Major: 21, SetDefault: true}); err != nil {
		t.Fatal(err)
	}
	store, err := serverinstance.NewStore(filepath.Join(dataDir, "instances", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	svc := New(Options{Root: root, InstancesRoot: instancesDir, HTTPClient: srv.Client(), Environment: env, Store: store, Resolver: Resolver{Client: srv.Client(), MojangBase: base, PaperBase: base, FabricBase: base}})
	online := true
	result, err := svc.Deploy(context.Background(), PlanRequest{Name: "Validation Stop", Version: "1.21.1", Software: SoftwarePaper, MemoryMB: 1024, Port: port, OnlineMode: &online, EULAAccepted: true, StartAfterDeploy: true})
	if err == nil || !strings.Contains(err.Error(), "Minecraft Ping 验证失败") {
		t.Fatalf("expected ping validation failure, got result=%#v err=%v", result, err)
	}
	if snap := svc.Status(result.Instance.ID); snap.State != "stopped" {
		t.Fatalf("failed validation left process running: %#v", snap)
	}
	persisted, getErr := store.Get(result.Instance.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if persisted.DesiredState != "stopped" || persisted.Health != "unhealthy" {
		t.Fatalf("failed validation state was not recorded safely: %#v", persisted)
	}
}
