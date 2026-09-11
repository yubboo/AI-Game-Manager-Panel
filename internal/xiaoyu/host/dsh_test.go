package host

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
)

func makeDSHFixture(t *testing.T) DSHBundle {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"dsh-demo","version":"1.2.3","type":"module","main":"index.mjs","dsh":{"bundle":{"patch":"./cordis.patch.yml"}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	source := `export const name='demo'; export const inject=['tools']; export function apply(ctx){ ctx.tools.register({ name:'greet', description:'greet a name', parameters:{name:{type:'string',required:true}}, async execute(args){ return 'hello '+args.name } }) }`
	if err := os.WriteFile(filepath.Join(dir, "index.mjs"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cordis.patch.yml"), []byte("- insert: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadDSHBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestDSHToolBridgeListsAndCallsTool(t *testing.T) {
	bundle := makeDSHFixture(t)
	bridge := NewDSHBridge(DSHBridgeOptions{})
	tools, err := bridge.List(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "greet" {
		t.Fatalf("unexpected tools: %#v", tools)
	}
	value, err := bridge.Call(context.Background(), bundle, "greet", map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	if value != "hello Ada" {
		t.Fatalf("unexpected value: %#v", value)
	}
}

func TestDSHPluginMountsNamespacedToolAndRevokes(t *testing.T) {
	bundle := makeDSHFixture(t)
	registry := xiaoyucontract.New()
	kernel := New(registry)
	plugin := DSHPlugin{Bundle: bundle, Policy: DSHPolicy{Risk: xiaoyucontract.RiskRead, Manual: true, XiaoYu: true}}
	if err := kernel.Add(plugin); err != nil {
		t.Fatal(err)
	}
	if err := kernel.MountAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	name := "dsh.dsh-demo.greet"
	if _, ok := registry.Get(name); !ok {
		t.Fatalf("missing %s", name)
	}
	out, err := registry.Execute(context.Background(), name, map[string]any{"name": "小鱼"})
	if err != nil {
		t.Fatal(err)
	}
	if out.Data != "hello 小鱼" {
		t.Fatalf("unexpected data: %#v", out.Data)
	}
	if err := kernel.Unmount(context.Background(), "dsh.dsh-demo"); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.Get(name); ok {
		t.Fatal("DSH tool should be revoked on unload")
	}
}

func TestLoadDSHDirectToolPluginWithoutBundleManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"direct-tool","version":"0.2.0","type":"module","module":"plugin.mjs"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.mjs"), []byte(`export const inject=['tools']; export function apply(ctx){ctx.tools.register({name:'ping',parameters:{value:{type:'string',required:true}},execute(args){return args.value}})}`), 0600); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadDSHBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Kind != "plugin" || bundle.Patch != "" {
		t.Fatalf("unexpected direct plugin metadata: %+v", bundle)
	}
	tools, err := NewDSHBridge(DSHBridgeOptions{}).List(context.Background(), bundle)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("unexpected tools: %#v", tools)
	}
	schema := tools[0].Parameters
	if schema["type"] != "object" {
		t.Fatalf("DSH parameter DSL was not normalized: %#v", schema)
	}
	required, ok := schema["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "value" {
		t.Fatalf("unexpected required schema: %#v", schema["required"])
	}
}

func TestDSHToolBridgePassesMountConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"config-tool","version":"0.1.0","type":"module","main":"index.mjs"}`), 0600); err != nil {
		t.Fatal(err)
	}
	source := `export const inject=['tools']; export function apply(ctx, config){ctx.tools.register({name:'who',parameters:{},execute(){return config.label || 'none'}})}`
	if err := os.WriteFile(filepath.Join(dir, "index.mjs"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadDSHBundle(dir)
	if err != nil {
		t.Fatal(err)
	}
	bridge := NewDSHBridge(DSHBridgeOptions{})
	value, err := bridge.CallConfigured(context.Background(), bundle, "who", map[string]any{}, map[string]any{"label": "xiaoyu"})
	if err != nil {
		t.Fatal(err)
	}
	if value != "xiaoyu" {
		t.Fatalf("config was not passed: %#v", value)
	}
}

func TestResolveDSHBundleRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "package.json"), []byte(`{"name":"escape","version":"1.0.0","type":"module","main":"index.mjs"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "index.mjs"), []byte(`export function apply(){}`), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable on this platform: %v", err)
	}
	if _, err := ResolveDSHBundle(root, "escape"); err == nil {
		t.Fatal("DSH plugin root symlink must not escape configured plugin root")
	}
}

func TestDSHBridgeEnvironmentDoesNotInheritSecrets(t *testing.T) {
	t.Setenv("AGMP_TEST_API_KEY", "must-not-leak")
	t.Setenv("PATH", os.Getenv("PATH"))
	for _, item := range dshBridgeEnvironment() {
		if strings.HasPrefix(item, "AGMP_TEST_API_KEY=") {
			t.Fatalf("secret environment leaked into DSH child: %q", item)
		}
	}
}
