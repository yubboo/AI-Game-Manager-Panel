package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Loader 负责从统一 configs/ 目录加载平台静态配置。
// Loader 不读取 API Key、Token、密码等秘密；秘密将由后续凭据存储模块管理。
type Loader struct {
	root string
}

func NewLoader(root string) *Loader {
	return &Loader{root: filepath.Clean(root)}
}

func (l *Loader) ConfigDir() string {
	return filepath.Join(l.root, "configs")
}

// LoadAll 一次性读取平台配置快照，确保 Desktop 与 Web 使用同一份配置来源。
func (l *Loader) LoadAll() (PlatformConfig, error) {
	var out PlatformConfig
	out.Update = UpdateConfig{
		Enabled: true, Provider: "github-releases", Repository: "yubboo/AI-Game-Manager-Panel",
		APIBaseURL: "https://api.github.com", Channel: "stable", CheckOnStartup: true,
		MinimumCheckIntervalMinutes: 360, AssetPattern: "AI-Game-Manager-Panel-{version}-Windows-x64-Setup.exe",
		RequireSHA256: true, InstallMode: "interactive", PreserveRuntimeData: true,
	}
	steps := []struct {
		name string
		dst  any
	}{
		{"app.json", &out.App},
		{"ui.json", &out.UI},
		{"ai.json", &out.AI},
		{"permissions.json", &out.Permissions},
		{"paths.json", &out.Paths},
		{"logging.json", &out.Logging},
		{"games.json", &out.Games},
		{"modules.json", &out.Modules},
		{"server.json", &out.Server},
	}
	for _, step := range steps {
		if err := l.readJSON(step.name, step.dst); err != nil {
			return PlatformConfig{}, err
		}
	}
	if err := l.readJSONOptional("update.json", &out.Update); err != nil {
		return PlatformConfig{}, err
	}
	if err := validate(out); err != nil {
		return PlatformConfig{}, err
	}
	return out, nil
}

func (l *Loader) readJSONOptional(name string, dst any) error {
	path := filepath.Join(l.ConfigDir(), name)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取平台配置失败 %s: %w", path, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("解析平台配置失败 %s: %w", path, err)
	}
	return nil
}

func (l *Loader) readJSON(name string, dst any) error {
	path := filepath.Join(l.ConfigDir(), name)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取平台配置失败 %s: %w", path, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("解析平台配置失败 %s: %w", path, err)
	}
	return nil
}

func validate(value PlatformConfig) error {
	if strings.TrimSpace(value.App.ProductName) == "" {
		return fmt.Errorf("configs/app.json 缺少 productName")
	}
	if strings.TrimSpace(value.App.DefaultRoute) == "" {
		return fmt.Errorf("configs/app.json 缺少 defaultRoute")
	}
	if len(value.UI.Navigation) == 0 {
		return fmt.Errorf("configs/ui.json 未配置 navigation")
	}
	if strings.TrimSpace(value.Server.Listen) == "" {
		return fmt.Errorf("configs/server.json 缺少 listen")
	}
	if len(value.Server.Architectures) == 0 {
		return fmt.Errorf("configs/server.json 未配置 architectures")
	}
	if value.Update.Enabled {
		if strings.TrimSpace(value.Update.Provider) == "" || strings.TrimSpace(value.Update.Repository) == "" {
			return fmt.Errorf("configs/update.json 缺少 provider/repository")
		}
	}
	if value.AI.Harness.EverythingAsPlugin {
		if strings.TrimSpace(value.AI.Harness.Kernel) == "" {
			return fmt.Errorf("configs/ai.json harness.kernel 不能为空")
		}
		if strings.TrimSpace(value.AI.Harness.TraceMode) != "append-only" {
			return fmt.Errorf("configs/ai.json harness.traceMode 必须为 append-only")
		}
	}
	if value.AI.Harness.DSH.Enabled {
		if strings.TrimSpace(value.AI.Harness.DSH.Mode) != "tool-bridge-v1" {
			return fmt.Errorf("configs/ai.json deepseekHarness.mode 当前只支持 tool-bridge-v1")
		}
		if value.AI.Harness.DSH.AutoMount {
			return fmt.Errorf("configs/ai.json deepseekHarness.autoMount 必须为 false；第三方插件禁止自动执行")
		}
	}
	if len(value.Permissions.ApprovalModes) == 0 {
		return fmt.Errorf("configs/permissions.json 未配置 approvalModes")
	}
	expectedApprovalModes := map[string]string{
		"ask":  "请求批准",
		"risk": "帮我批准",
		"full": "完全访问权限",
	}
	if len(value.Permissions.ApprovalModes) != len(expectedApprovalModes) {
		return fmt.Errorf("configs/permissions.json approvalModes 必须且只能包含：请求批准 / 帮我批准 / 完全访问权限")
	}
	for _, mode := range value.Permissions.ApprovalModes {
		want, ok := expectedApprovalModes[strings.TrimSpace(mode.ID)]
		if !ok || strings.TrimSpace(mode.Label) != want {
			return fmt.Errorf("configs/permissions.json 审批模式命名不可修改：ask=请求批准, risk=帮我批准, full=完全访问权限")
		}
	}
	for name, policies := range map[string]map[string]string{
		"policies.ask":  value.Permissions.Policies.Ask,
		"policies.risk": value.Permissions.Policies.Risk,
		"policies.full": value.Permissions.Policies.Full,
	} {
		for _, risk := range []string{"read", "operate", "modify", "destructive", "system"} {
			decision := strings.TrimSpace(policies[risk])
			if decision != "allow" && decision != "confirm" && decision != "deny" {
				return fmt.Errorf("configs/permissions.json %s.%s 必须是 allow/confirm/deny", name, risk)
			}
		}
	}
	seen := map[string]struct{}{}
	for _, module := range value.Modules.Modules {
		id := strings.TrimSpace(module.ID)
		if id == "" {
			return fmt.Errorf("configs/modules.json 存在空 module id")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("configs/modules.json 存在重复 module id: %s", id)
		}
		seen[id] = struct{}{}
	}
	return nil
}
