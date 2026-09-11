package app

import (
	"testing"

	"github.com/yubboo/AI-Game-Manager-Panel/internal/config"
)

func TestSelectXiaoYuModulesUsesProductMapAndStatus(t *testing.T) {
	modules := []config.ModuleConfig{
		{ID: "ai", Name: "小鱼", Status: "partial", Route: "/", Features: []string{"Agent"}},
		{ID: "environment", Name: "运行环境与存储", Category: "系统", Status: "partial", Route: "/settings?section=environment", Features: []string{"SteamCMD", "Java"}},
		{ID: "minecraft", Name: "Minecraft", Category: "游戏", Status: "skeleton", Route: "/deployment", Features: []string{"Minecraft 部署"}},
	}
	got := selectXiaoYuModules(modules, "帮我检查 SteamCMD 环境", 8)
	foundEnvironment := false
	for _, item := range got {
		if item.ID == "environment" {
			foundEnvironment = true
			if item.Status != "partial" || item.Route == "" {
				t.Fatalf("module knowledge lost status/route: %+v", item)
			}
		}
	}
	if !foundEnvironment {
		t.Fatalf("environment module not selected: %#v", got)
	}
}

func TestXiaoYuSystemModuleCatalogKeepsWholeProductMap(t *testing.T) {
	a := &Application{platformConfig: config.PlatformConfig{Modules: config.ModulesConfig{Modules: []config.ModuleConfig{
		{ID: "ai", Name: "小鱼", Status: "partial", Route: "/"},
		{ID: "settings", Name: "设置中心", Status: "partial", Route: "/settings"},
		{ID: "backups", Name: "备份恢复", Status: "skeleton", Route: "/backups"},
	}}}}
	catalog := a.xiaoyuSystemModuleCatalog()
	if len(catalog) != 3 {
		t.Fatalf("full module catalog must preserve every product module, got %d", len(catalog))
	}
	seen := map[string]bool{}
	for _, item := range catalog {
		seen[item.ID] = true
	}
	for _, id := range []string{"ai", "settings", "backups"} {
		if !seen[id] {
			t.Fatalf("module %s missing from full XiaoYu system catalog: %#v", id, catalog)
		}
	}
}
