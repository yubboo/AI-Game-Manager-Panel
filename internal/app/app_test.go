package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
)

func TestNewWithOptionsKeepsAuthDataInsideInjectedRoot(t *testing.T) {
	root := t.TempDir()
	app := NewWithOptions(Options{Root: root, DataDir: "isolated-data"})

	wantDataDir := filepath.Join(root, "isolated-data")
	if got := filepath.Clean(app.Info().DataDir); got != filepath.Clean(wantDataDir) {
		t.Fatalf("DataDir = %q, want %q", got, wantDataDir)
	}

	_, err := app.CreateInitialAdministrator(authservice.CreateOwnerRequest{
		Username:    "owner",
		DisplayName: "Owner",
		Password:    "agmp-test-password",
	})
	if err != nil {
		t.Fatalf("CreateInitialAdministrator() error = %v", err)
	}

	accountPath := filepath.Join(wantDataDir, "auth", "accounts.json")
	if _, err := os.Stat(accountPath); err != nil {
		t.Fatalf("isolated account store missing at %q: %v", accountPath, err)
	}
	lockPath := filepath.Join(wantDataDir, "auth", "bootstrap.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("isolated bootstrap lock missing at %q: %v", lockPath, err)
	}
}

func TestNewWithOptionsUsesAbsoluteDataDirWithoutUserConfigFallback(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(t.TempDir(), "explicit-data")
	app := NewWithOptions(Options{Root: root, DataDir: dataDir})
	if got := filepath.Clean(app.Info().DataDir); got != filepath.Clean(dataDir) {
		t.Fatalf("DataDir = %q, want explicit %q", got, dataDir)
	}
}

func TestNewWithOptionsDefaultsWritableFilesToRuntimeDirectory(t *testing.T) {
	root := t.TempDir()
	app := NewWithOptions(Options{Root: root})

	wantDataDir := filepath.Join(root, "runtime", "data")
	if got := filepath.Clean(app.Info().DataDir); got != filepath.Clean(wantDataDir) {
		t.Fatalf("DataDir = %q, want runtime data dir %q", got, wantDataDir)
	}

	app.Startup(context.Background())
	defer app.Shutdown(context.Background())
	logPath := filepath.Join(root, "runtime", "log", "agmp", "agmp.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("runtime log missing at %q: %v", logPath, err)
	}
	if _, err := os.Stat(filepath.Join(root, "log")); !os.IsNotExist(err) {
		t.Fatalf("legacy root log directory should not be created by 0.1.65 defaults")
	}
}

func TestXiaoYuHostToolRegistryOwnsExecutableDomainTools(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "hello.txt"), []byte("xiaoyu-host-tools"), 0o600); err != nil {
		t.Fatal(err)
	}
	app := NewWithOptions(Options{Root: root})
	tools := app.xiaoyuTools.List()
	if len(tools) < 10 {
		t.Fatalf("host tool count = %d, want structured system/game hands: %#v", len(tools), tools)
	}
	for _, name := range []string{"system.info", "games.list", "steam.snapshot", "environment.status", "environment.resolve_runtime", "environment.set_default_runtime", "environment.remove_runtime", "dst.workspace", "dst.cluster.status", "dst.cluster.start", "dst.cluster.stop", "dst.command", "dst.logs.tail", "fs.read", "fs.stat", "fs.write", "fs.replace", "fs.mkdir", "fs.remove", "shell.exec", "process.run"} {
		if _, ok := app.xiaoyuTools.Get(name); !ok {
			t.Fatalf("missing XiaoYu Host tool %s", name)
		}
	}
	shellTool, ok := app.xiaoyuTools.Get("shell.exec")
	if !ok || !shellTool.XiaoYu || shellTool.Risk != xiaoyucontract.RiskSystem {
		t.Fatalf("shell.exec must be a model-visible general fallback protected by approval policy: %+v", shellTool)
	}
	processTool, ok := app.xiaoyuTools.Get("process.run")
	if !ok || processTool.XiaoYu {
		t.Fatalf("process.run must remain a manual compatibility alias: %+v", processTool)
	}
	result, err := app.xiaoyuTools.Execute(context.Background(), "fs.read", map[string]any{"path": "hello.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data == nil {
		t.Fatal("unexpected nil tool payload")
	}
	if result.Summary == "" {
		t.Fatal("tool execution should return a summary")
	}
}

func TestXiaoYuRoleAllowsRiskMatrix(t *testing.T) {
	if !xiaoyuRoleAllows(authservice.RoleOwner, xiaoyucontract.RiskSystem) || !xiaoyuRoleAllows(authservice.RoleAdministrator, xiaoyucontract.RiskDestructive) {
		t.Fatal("owner/administrator must retain privileged XiaoYu Tool access")
	}
	if !xiaoyuRoleAllows(authservice.RoleOperator, xiaoyucontract.RiskModify) {
		t.Fatal("operator should be able to use modify-level domain tools")
	}
	if xiaoyuRoleAllows(authservice.RoleOperator, xiaoyucontract.RiskSystem) || xiaoyuRoleAllows(authservice.RoleOperator, xiaoyucontract.RiskDestructive) {
		t.Fatal("operator must never gain destructive/system capability from approval mode")
	}
}
