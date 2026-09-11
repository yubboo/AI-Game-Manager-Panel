package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoaderLoadAll(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"app.json":         `{"productName":"AI游戏管理器面板","defaultRoute":"/"}`,
		"ui.json":          `{"navigation":[{"id":"main","items":[{"id":"ai","to":"/","label":"小鱼","icon":"ai"}]}]}`,
		"ai.json":          `{}`,
		"permissions.json": `{"approvalModes":[{"id":"ask","label":"请求批准","description":"测试"},{"id":"risk","label":"帮我批准","description":"测试"},{"id":"full","label":"完全访问权限","description":"测试"}],"policies":{"ask":{"read":"allow","operate":"confirm","modify":"confirm","destructive":"confirm","system":"confirm"},"risk":{"read":"allow","operate":"allow","modify":"allow","destructive":"confirm","system":"confirm"},"full":{"read":"allow","operate":"allow","modify":"allow","destructive":"allow","system":"allow"}}}`,
		"paths.json":       `{}`,
		"logging.json":     `{}`,
		"games.json":       `{"templates":[]}`,
		"modules.json":     `{"modules":[{"id":"ai","name":"小鱼"}]}`,
		"server.json":      `{"listen":"127.0.0.1:17890","architectures":["amd64","arm64"]}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(configDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	value, err := NewLoader(root).LoadAll()
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if value.App.ProductName != "AI游戏管理器面板" {
		t.Fatalf("ProductName = %q", value.App.ProductName)
	}
	if len(value.Modules.Modules) != 1 {
		t.Fatalf("Modules = %d", len(value.Modules.Modules))
	}
}
