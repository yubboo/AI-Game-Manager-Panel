package host

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
)

//go:embed dsh_tool_bridge.mjs
var dshToolBridgeScript string

type DSHBundle struct {
	PackageName   string `json:"packageName"`
	Version       string `json:"version"`
	DirectoryName string `json:"directory"`
	Kind          string `json:"kind"`
	Patch         string `json:"patch,omitempty"`
	Directory     string `json:"-"`
	Entry         string `json:"-"`
}

type dshPackageJSON struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Main    string `json:"main"`
	Module  string `json:"module"`
	DSH     struct {
		Bundle *struct {
			Patch string `json:"patch"`
		} `json:"bundle"`
	} `json:"dsh"`
}

type DSHBridgeOptions struct {
	NodeExecutable string
	Timeout        time.Duration
}

type DSHBridge struct {
	node    string
	timeout time.Duration
}

type DSHExternalTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type DSHPolicy struct {
	Risk   xiaoyucontract.RiskLevel
	Manual bool
	XiaoYu bool
}

func NewDSHBridge(options DSHBridgeOptions) *DSHBridge {
	node := strings.TrimSpace(options.NodeExecutable)
	if node == "" {
		node = "node"
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &DSHBridge{node: node, timeout: timeout}
}

// DiscoverDSHBundles recognizes pre-installed DeepSeek Harness bundle packages.
// It does not run npm/pnpm or install scripts; installation remains an explicit
// administrator action because package prepare/install hooks execute code.
func DiscoverDSHBundles(root string) ([]DSHBundle, error) {
	root = filepath.Clean(root)
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []DSHBundle{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]DSHBundle, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bundle, loadErr := LoadDSHBundle(filepath.Join(root, entry.Name()))
		if loadErr == nil {
			result = append(result, bundle)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].PackageName < result[j].PackageName })
	return result, nil
}

// ResolveDSHBundle only accepts one package-directory name below the configured
// DSH root. This prevents callers from turning plugin discovery into arbitrary
// filesystem access.
func ResolveDSHBundle(root, directoryName string) (DSHBundle, error) {
	root, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return DSHBundle{}, err
	}
	if realRoot, evalErr := filepath.EvalSymlinks(root); evalErr == nil {
		root = realRoot
	}
	directoryName = strings.TrimSpace(directoryName)
	if directoryName == "" || directoryName == "." || directoryName == ".." || filepath.Base(directoryName) != directoryName {
		return DSHBundle{}, errors.New("invalid DSH plugin directory")
	}
	candidate := filepath.Join(root, directoryName)
	realCandidate, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return DSHBundle{}, err
	}
	if !pathWithin(root, realCandidate) {
		return DSHBundle{}, errors.New("DSH plugin directory escapes configured plugin root through a symlink")
	}
	return LoadDSHBundle(realCandidate)
}

type DSHMountRequest struct {
	Directory string         `json:"directory"`
	Trusted   bool           `json:"trusted"`
	XiaoYu    bool           `json:"xiaoyu"`
	Config    map[string]any `json:"config,omitempty"`
}

func LoadDSHBundle(directory string) (DSHBundle, error) {
	directory, err := filepath.Abs(filepath.Clean(directory))
	if err != nil {
		return DSHBundle{}, err
	}
	directory, err = filepath.EvalSymlinks(directory)
	if err != nil {
		return DSHBundle{}, err
	}
	raw, err := os.ReadFile(filepath.Join(directory, "package.json"))
	if err != nil {
		return DSHBundle{}, err
	}
	var pkg dshPackageJSON
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return DSHBundle{}, fmt.Errorf("parse DSH package.json: %w", err)
	}
	if strings.TrimSpace(pkg.Name) == "" {
		return DSHBundle{}, fmt.Errorf("%w: package name is empty", ErrUnsupportedPlugin)
	}
	kind := "plugin"
	patch := ""
	if pkg.DSH.Bundle != nil && strings.TrimSpace(pkg.DSH.Bundle.Patch) != "" {
		kind = "bundle-entry"
		patch = strings.TrimSpace(pkg.DSH.Bundle.Patch)
		patchPath := filepath.Join(directory, filepath.FromSlash(patch))
		patchPath, err = filepath.Abs(filepath.Clean(patchPath))
		if err != nil {
			return DSHBundle{}, err
		}
		patchPath, err = filepath.EvalSymlinks(patchPath)
		if err != nil {
			return DSHBundle{}, err
		}
		if !pathWithin(directory, patchPath) {
			return DSHBundle{}, errors.New("DSH bundle patch escapes its package directory")
		}
		if info, statErr := os.Stat(patchPath); statErr != nil || info.IsDir() {
			return DSHBundle{}, fmt.Errorf("DSH bundle patch is unavailable: %s", patchPath)
		}
	}
	entry := strings.TrimSpace(pkg.Module)
	if entry == "" {
		entry = strings.TrimSpace(pkg.Main)
	}
	if entry == "" {
		return DSHBundle{}, fmt.Errorf("%w: package has no executable main/module entry; patch-only profile bundles require the full DSH runtime", ErrUnsupportedPlugin)
	}
	entryPath := filepath.Join(directory, filepath.FromSlash(entry))
	entryPath, err = filepath.Abs(filepath.Clean(entryPath))
	if err != nil {
		return DSHBundle{}, err
	}
	entryPath, err = filepath.EvalSymlinks(entryPath)
	if err != nil {
		return DSHBundle{}, err
	}
	if !pathWithin(directory, entryPath) {
		return DSHBundle{}, errors.New("DSH plugin entry escapes its package directory")
	}
	info, err := os.Stat(entryPath)
	if err != nil || info.IsDir() {
		return DSHBundle{}, fmt.Errorf("DSH plugin entry is unavailable: %s", entryPath)
	}
	return DSHBundle{PackageName: strings.TrimSpace(pkg.Name), Version: strings.TrimSpace(pkg.Version), DirectoryName: filepath.Base(directory), Kind: kind, Directory: directory, Entry: entryPath, Patch: patch}, nil
}

func (b *DSHBridge) List(ctx context.Context, bundle DSHBundle) ([]DSHExternalTool, error) {
	return b.ListConfigured(ctx, bundle, nil)
}

func (b *DSHBridge) ListConfigured(ctx context.Context, bundle DSHBundle, config map[string]any) ([]DSHExternalTool, error) {
	var result struct {
		Tools []DSHExternalTool `json:"tools"`
	}
	if err := b.invoke(ctx, bundle, map[string]any{"action": "list", "entry": bundle.Entry, "config": config}, &result); err != nil {
		return nil, err
	}
	return result.Tools, nil
}

func (b *DSHBridge) Call(ctx context.Context, bundle DSHBundle, tool string, arguments map[string]any) (any, error) {
	return b.CallConfigured(ctx, bundle, tool, arguments, nil)
}

func (b *DSHBridge) CallConfigured(ctx context.Context, bundle DSHBundle, tool string, arguments map[string]any, config map[string]any) (any, error) {
	var result struct {
		Value any `json:"value"`
	}
	err := b.invoke(ctx, bundle, map[string]any{"action": "call", "entry": bundle.Entry, "tool": tool, "arguments": arguments, "config": config}, &result)
	return result.Value, err
}

func (b *DSHBridge) invoke(parent context.Context, bundle DSHBundle, request any, target any) error {
	payload, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, b.timeout)
	defer cancel()
	result, err := platformruntime.Run(ctx, platformruntime.RunSpec{Spec: platformruntime.Spec{
		Executable: b.node,
		Arguments: []string{
			"--permission",
			"--allow-fs-read=" + bundle.Directory,
			"--input-type=module",
			"--eval", dshToolBridgeScript,
		},
		WorkingDirectory:   bundle.Directory,
		Environment:        dshBridgeEnvironment(),
		ReplaceEnvironment: true,
	}, Stdin: string(payload), MaxOutputBytes: 2 * 1024 * 1024})
	if err != nil {
		detail := strings.TrimSpace(result.Stderr)
		if detail == "" {
			detail = err.Error()
		}
		return fmt.Errorf("DSH compatibility bridge failed: %s", detail)
	}
	var envelope struct {
		OK     bool            `json:"ok"`
		Error  string          `json:"error"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &envelope); err != nil {
		return fmt.Errorf("parse DSH bridge response: %w", err)
	}
	if !envelope.OK {
		return fmt.Errorf("DSH plugin error: %s", envelope.Error)
	}
	if target == nil {
		return nil
	}
	return json.Unmarshal(envelope.Result, target)
}

// dshBridgeEnvironment deliberately does not inherit the AGMP process environment.
// API keys, Steam tokens and other host credentials must never become ambient
// variables inside third-party JavaScript. Plugins receive only the explicit
// configuration supplied for that plugin plus a minimal process environment.
func dshBridgeEnvironment() []string {
	allowed := map[string]struct{}{
		"PATH": {}, "PATHEXT": {}, "SYSTEMROOT": {}, "WINDIR": {}, "COMSPEC": {},
		"TEMP": {}, "TMP": {}, "TMPDIR": {}, "HOME": {}, "USERPROFILE": {},
		"LANG": {}, "LC_ALL": {}, "TZ": {},
	}
	result := make([]string, 0, len(allowed))
	for _, item := range os.Environ() {
		name, _, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		if _, keep := allowed[strings.ToUpper(strings.TrimSpace(name))]; keep {
			result = append(result, item)
		}
	}
	return result
}

func pathWithin(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return false
	}
	return true
}

var dshSegmentPattern = regexp.MustCompile(`[^a-z0-9_-]+`)

func dshSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "@")
	value = strings.ReplaceAll(value, "/", "_")
	value = dshSegmentPattern.ReplaceAllString(value, "_")
	value = strings.Trim(value, "_-")
	if value == "" || value[0] < 'a' || value[0] > 'z' {
		value = "plugin_" + value
	}
	return value
}

// DSHPlugin adapts a DeepSeek Harness tool-only bundle into XiaoYu Host. The
// external package still executes in Node; AGMP owns risk/approval and exposes
// each imported tool under a dsh.<package>.<tool> namespace to prevent hijacks.
type DSHPlugin struct {
	Bundle DSHBundle
	Bridge *DSHBridge
	Policy DSHPolicy
	Config map[string]any
}

func (p DSHPlugin) Manifest() Manifest {
	return Manifest{ID: "dsh." + dshSegment(p.Bundle.PackageName), Name: p.Bundle.PackageName, Version: p.Bundle.Version, Source: "deepseek-harness", Kinds: []CapabilityKind{CapabilityTool, CapabilityBridge}, Experimental: true, Description: "DeepSeek Harness tool-plugin compatibility bridge"}
}

func (p DSHPlugin) Apply(ctx context.Context, host *Context) (DisposeFunc, error) {
	bridge := p.Bridge
	if bridge == nil {
		bridge = NewDSHBridge(DSHBridgeOptions{})
	}
	tools, err := bridge.ListConfigured(ctx, p.Bundle, p.Config)
	if err != nil {
		return nil, err
	}
	risk := p.Policy.Risk
	if risk == "" {
		risk = xiaoyucontract.RiskSystem
	}
	packageSegment := dshSegment(p.Bundle.PackageName)
	for _, external := range tools {
		external := external
		if strings.TrimSpace(external.Name) == "" {
			continue
		}
		toolName := "dsh." + packageSegment + "." + dshSegment(external.Name)
		spec := xiaoyucontract.ToolSpec{Name: toolName, Description: external.Description, Risk: risk, Category: "dsh", Manual: p.Policy.Manual, XiaoYu: p.Policy.XiaoYu, Parameters: external.Parameters, Source: "deepseek-harness:" + p.Bundle.PackageName}
		if err := host.RegisterTool(spec, func(callCtx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
			value, callErr := bridge.CallConfigured(callCtx, p.Bundle, external.Name, args, p.Config)
			if callErr != nil {
				return xiaoyucontract.ToolExecution{}, callErr
			}
			return xiaoyucontract.ToolExecution{Summary: "DeepSeek Harness 插件 Tool 执行完成。", Data: value}, nil
		}); err != nil {
			return nil, err
		}
	}
	return nil, nil
}
