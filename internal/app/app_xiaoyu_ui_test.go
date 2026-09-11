package app

import (
	"context"
	"path/filepath"
	"testing"

	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
)

func TestXiaoYuUINavigationTargetsAreWhitelisted(t *testing.T) {
	ids := xiaoyuUINavigationTargetIDs()
	if len(ids) < 10 {
		t.Fatalf("expected UI navigation whitelist, got %d targets", len(ids))
	}
	for _, id := range []string{"settings", "settings.models", "settings.environment", "dashboard", "deployment"} {
		target, ok := xiaoyuUINavigationTarget(id)
		if !ok || target.Path == "" || target.Path[0] != '/' {
			t.Fatalf("missing safe UI target %q: %#v", id, target)
		}
	}
	if _, ok := xiaoyuUINavigationTarget("https://example.com"); ok {
		t.Fatal("arbitrary external URL must never be accepted as a UI target")
	}
}

func TestNormalizeXiaoYuRunContextAllowsOnlyInternalUIRoute(t *testing.T) {
	value, err := normalizeXiaoYuRunContext(xiaoyuhost.RunContext{SessionID: "s", UIRoute: "/settings?section=models"})
	if err != nil || value.UIRoute != "/settings?section=models" {
		t.Fatalf("expected safe internal route, got %#v err=%v", value, err)
	}
	for _, bad := range []string{"https://example.com", "//example.com/path"} {
		if _, err := normalizeXiaoYuRunContext(xiaoyuhost.RunContext{UIRoute: bad}); err == nil {
			t.Fatalf("expected unsafe uiRoute %q to be rejected", bad)
		}
	}
}

func TestXiaoYuSettingsThemeCapabilityRegistered(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	a := NewWithOptions(Options{Root: root, DataDir: t.TempDir()})
	for _, name := range []string{"settings.get", "settings.theme.set"} {
		spec, ok := a.xiaoyuTools.Get(name)
		if !ok || !spec.XiaoYu {
			t.Fatalf("expected XiaoYu settings capability %q, got %#v", name, spec)
		}
	}
	set, _ := a.xiaoyuTools.Get("settings.theme.set")
	if set.Risk != xiaoyucontract.RiskModify {
		t.Fatalf("theme change must be classified as a state mutation, got %s", set.Risk)
	}
	if _, err := a.xiaoyuTools.Execute(context.Background(), "settings.theme.set", map[string]any{"theme": "light"}); err != nil {
		t.Fatalf("execute theme capability: %v", err)
	}
	if got := a.Settings().Theme; got != "light" {
		t.Fatalf("theme capability did not persist setting: %s", got)
	}
}
