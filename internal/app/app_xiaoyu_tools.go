package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	environmentservice "github.com/yubboo/AI-Game-Manager-Panel/internal/deploy/environment"
	dstruntimecore "github.com/yubboo/AI-Game-Manager-Panel/internal/games/dst/runtime"
	opsfiles "github.com/yubboo/AI-Game-Manager-Panel/internal/ops/files"
	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
	systemsettings "github.com/yubboo/AI-Game-Manager-Panel/internal/system/settings"
	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
	xiaoyuruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/runtime"
)

func (a *Application) registerXiaoYuTools() {
	if a.xiaoyuTools == nil {
		a.xiaoyuTools = xiaoyucontract.New()
	}
	if a.xiaoyuHost == nil {
		a.xiaoyuHost = xiaoyuhost.New(a.xiaoyuTools)
	}

	plugins := []xiaoyuhost.Plugin{
		xiaoyuhost.PluginFunc{Meta: xiaoyuhost.Manifest{ID: "agmp.ui", Name: "AGMP UI Control", Version: Version, Source: "agmp", Kinds: []xiaoyuhost.CapabilityKind{xiaoyuhost.CapabilityTool, xiaoyuhost.CapabilityUI}, Provides: []string{"ui.navigation"}}, ApplyFn: func(_ context.Context, host *xiaoyuhost.Context) (xiaoyuhost.DisposeFunc, error) {
			err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "ui.navigate", Description: "在发起当前 XiaoYu 对话的 AGMP 客户端中打开指定页面。用户说‘打开/进入/切换到 设置中心、模型管理、运行环境、仪表盘、部署、实例、任务、文件、备份、插件、网络、日志、节点、用户权限、开发者工具’时使用本 Tool；这是受限站内导航，不允许任意 URL。", Risk: xiaoyucontract.RiskRead,
				Category: "ui", Manual: false, XiaoYu: true, Source: "agmp.ui",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"target": map[string]any{"type": "string", "enum": xiaoyuUINavigationTargetIDs()}}, "required": []string{"target"}, "additionalProperties": false},
			}, func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				invocation, ok := getXiaoYuInvocationContext(ctx)
				if !ok || strings.TrimSpace(invocation.RunID) == "" || strings.TrimSpace(invocation.RunContext.SessionID) == "" {
					return xiaoyucontract.ToolExecution{}, errors.New("当前 XiaoYu 调用没有可定位的客户端会话，无法执行 UI 导航")
				}
				target, _ := args["target"].(string)
				destination, ok := xiaoyuUINavigationTarget(strings.TrimSpace(target))
				if !ok {
					return xiaoyucontract.ToolExecution{}, fmt.Errorf("不支持的 AGMP UI 导航目标：%s", target)
				}
				if a.xiaoyuHost == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("XiaoYu Host 不可用")
				}
				a.xiaoyuHost.Observe(xiaoyuhost.Event{
					Type: "ui/navigate", RunID: invocation.RunID, Summary: "打开" + destination.Label,
					Data: map[string]any{"sessionId": invocation.RunContext.SessionID, "target": destination.ID, "path": destination.Path, "label": destination.Label},
				})
				return xiaoyucontract.ToolExecution{Summary: "已在当前 AGMP 会话打开" + destination.Label + "。", Data: map[string]any{"target": destination.ID, "path": destination.Path, "label": destination.Label}}, nil
			})
			return nil, err
		}},
		xiaoyuhost.PluginFunc{Meta: xiaoyuhost.Manifest{ID: "agmp.settings", Name: "AGMP Settings Capability", Version: Version, Source: "agmp", Kinds: []xiaoyuhost.CapabilityKind{xiaoyuhost.CapabilityTool, xiaoyuhost.CapabilityUI}, Provides: []string{"settings.preferences"}}, ApplyFn: func(_ context.Context, host *xiaoyuhost.Context) (xiaoyuhost.DisposeFunc, error) {
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "settings.get", Description: "读取 AGMP 当前外观与基础偏好。用户询问当前主题/语言/调试状态时使用。", Risk: xiaoyucontract.RiskRead,
				Category: "settings", Manual: true, XiaoYu: true, Source: "agmp.settings",
				Parameters: map[string]any{"type": "object", "additionalProperties": false},
			}, func(_ context.Context, _ map[string]any) (xiaoyucontract.ToolExecution, error) {
				value := a.Settings()
				return xiaoyucontract.ToolExecution{Summary: "已读取 AGMP 当前设置。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "settings.theme.set", Description: "直接修改 AGMP 全局主题偏好，不需要先导航到设置页。用户要求改成浅色/深色/跟随系统时使用；这是可逆设置变更，执行后必须验证结果。", Risk: xiaoyucontract.RiskModify,
				Category: "settings", Manual: true, XiaoYu: true, Source: "agmp.settings",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"theme": map[string]any{"type": "string", "enum": []string{"light", "dark", "system"}}}, "required": []string{"theme"}, "additionalProperties": false},
			}, func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				theme, _ := args["theme"].(string)
				theme = strings.ToLower(strings.TrimSpace(theme))
				if theme != "light" && theme != "dark" && theme != "system" {
					return xiaoyucontract.ToolExecution{}, fmt.Errorf("不支持的主题：%s", theme)
				}
				value := a.Settings()
				previousTheme := value.Theme
				value.Theme = theme
				if err := a.SaveSettings(systemsettings.Settings{Theme: value.Theme, Language: value.Language, Debug: value.Debug}); err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				if invocation, ok := getXiaoYuInvocationContext(ctx); ok && a.xiaoyuHost != nil {
					a.xiaoyuHost.Observe(xiaoyuhost.Event{
						Type: "settings/changed", RunID: invocation.RunID, Summary: "AGMP 主题已更新",
						Data: map[string]any{"sessionId": invocation.RunContext.SessionID, "theme": theme},
					})
				}
				return xiaoyucontract.ToolExecution{
					Summary: "AGMP 主题已切换为 " + theme + "。",
					Data:    map[string]any{"theme": theme, "previousTheme": previousTheme},
				}, nil
			}); err != nil {
				return nil, err
			}
			return nil, nil
		}},
		xiaoyuhost.PluginFunc{Meta: xiaoyuhost.Manifest{ID: "agmp.system", Name: "AGMP System Capability", Version: Version, Source: "agmp", Kinds: []xiaoyuhost.CapabilityKind{xiaoyuhost.CapabilityTool}, Provides: []string{"agmp.capability-map"}}, ApplyFn: func(_ context.Context, host *xiaoyuhost.Context) (xiaoyuhost.DisposeFunc, error) {
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "system.info", Description: "读取 AGMP 与小鱼核心的基础运行信息。", Risk: xiaoyucontract.RiskRead,
				Category: "system", Manual: true, XiaoYu: true, Source: "agmp.system",
				Parameters: map[string]any{"type": "object", "additionalProperties": false},
			}, func(_ context.Context, _ map[string]any) (xiaoyucontract.ToolExecution, error) {
				status := a.XiaoYuRuntimeStatus()
				return xiaoyucontract.ToolExecution{Summary: "已读取 AGMP 与小鱼核心运行信息。", Data: map[string]any{"application": a.Info(), "xiaoyu": status}}, nil
			}); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "agmp.capability.search", Description: "搜索 AGMP 全部产品模块与 XiaoYu 当前真实可执行 Tool。遇到‘这个系统能不能做什么/应该怎么操作/找不到下一步/直接 Tool 不明显’时先用它发现能力；返回模块实现状态，skeleton/planned 只表示路线图，不能当成已实现。", Risk: xiaoyucontract.RiskRead,
				Category: "system", Manual: false, XiaoYu: true, Source: "agmp.system",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string", "minLength": 1, "maxLength": 200}}, "required": []string{"query"}, "additionalProperties": false},
			}, func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				query, _ := args["query"].(string)
				query = strings.TrimSpace(query)
				if query == "" {
					return xiaoyucontract.ToolExecution{}, errors.New("Capability 搜索关键词不能为空")
				}
				value := a.searchXiaoYuCapabilities(ctx, query)
				return xiaoyucontract.ToolExecution{Summary: "已搜索 AGMP 产品模块与当前可执行能力。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			return nil, nil
		}},
		xiaoyuhost.PluginFunc{Meta: xiaoyuhost.Manifest{ID: "agmp.files", Name: "AGMP Workspace Files", Version: Version, Source: "agmp", Kinds: []xiaoyuhost.CapabilityKind{xiaoyuhost.CapabilityTool, xiaoyuhost.CapabilitySandbox}, Provides: []string{"workspace.files"}}, ApplyFn: func(_ context.Context, host *xiaoyuhost.Context) (xiaoyuhost.DisposeFunc, error) {
			if err := host.RegisterService("workspace.files", a.workspaceFiles); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "fs.list", Description: "列出 AGMP 工作区范围内的目录内容。", Risk: xiaoyucontract.RiskRead,
				Category: "filesystem", Manual: true, XiaoYu: true, Source: "agmp.files",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}, "additionalProperties": false},
			}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if a.workspaceFiles == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("AGMP 工作区文件服务不可用")
				}
				path, _ := args["path"].(string)
				value, err := a.workspaceFiles.List(path)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: "已列出工作区目录。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "fs.read", Description: "读取 AGMP 工作区范围内的 UTF-8 文本文件。", Risk: xiaoyucontract.RiskRead,
				Category: "filesystem", Manual: true, XiaoYu: true, Source: "agmp.files",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}, "required": []string{"path"}, "additionalProperties": false},
			}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if a.workspaceFiles == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("AGMP 工作区文件服务不可用")
				}
				path, _ := args["path"].(string)
				value, err := a.workspaceFiles.Read(path, opsfiles.DefaultReadLimit)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: "已读取工作区文本文件。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "fs.stat", Description: "检查 AGMP 工作区内路径是否存在、是否目录及大小；可用于变更前后验证。", Risk: xiaoyucontract.RiskRead,
				Category: "filesystem", Manual: true, XiaoYu: true, Source: "agmp.files",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}}, "required": []string{"path"}, "additionalProperties": false},
			}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if a.workspaceFiles == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("AGMP 工作区文件服务不可用")
				}
				path, _ := args["path"].(string)
				value, err := a.workspaceFiles.Stat(path)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: "已检查工作区路径状态。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "fs.write", Description: "在 AGMP 工作区内写入 UTF-8 文本文件。优先用于没有更专业 Domain Tool 的通用文件任务；写入后应使用 fs.read/fs.stat 或领域状态 Tool 验证。", Risk: xiaoyucontract.RiskModify,
				Category: "filesystem", Manual: true, XiaoYu: true, Source: "agmp.files",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "content": map[string]any{"type": "string"}, "createParents": map[string]any{"type": "boolean"}}, "required": []string{"path", "content"}, "additionalProperties": false},
			}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if a.workspaceFiles == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("AGMP 工作区文件服务不可用")
				}
				path, _ := args["path"].(string)
				content, _ := args["content"].(string)
				createParents, _ := args["createParents"].(bool)
				value, err := a.workspaceFiles.Write(path, content, createParents)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: "已写入工作区文本文件。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "fs.replace", Description: "在 AGMP 工作区文本文件中按精确旧文本做替换。oldText 同时是变更前置条件，找不到时会失败而不是盲目覆盖；适合配置文件的小范围可靠修改。", Risk: xiaoyucontract.RiskModify,
				Category: "filesystem", Manual: true, XiaoYu: true, Source: "agmp.files",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "oldText": map[string]any{"type": "string"}, "newText": map[string]any{"type": "string"}, "replaceAll": map[string]any{"type": "boolean"}}, "required": []string{"path", "oldText", "newText"}, "additionalProperties": false},
			}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if a.workspaceFiles == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("AGMP 工作区文件服务不可用")
				}
				path, _ := args["path"].(string)
				oldText, _ := args["oldText"].(string)
				newText, _ := args["newText"].(string)
				replaceAll, _ := args["replaceAll"].(bool)
				value, err := a.workspaceFiles.Replace(path, oldText, newText, replaceAll)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: "已按精确前置条件修改工作区文本文件。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "fs.mkdir", Description: "在 AGMP 工作区内创建目录。", Risk: xiaoyucontract.RiskModify,
				Category: "filesystem", Manual: true, XiaoYu: true, Source: "agmp.files",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "parents": map[string]any{"type": "boolean"}}, "required": []string{"path"}, "additionalProperties": false},
			}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if a.workspaceFiles == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("AGMP 工作区文件服务不可用")
				}
				path, _ := args["path"].(string)
				parents, _ := args["parents"].(bool)
				value, err := a.workspaceFiles.Mkdir(path, parents)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: "已创建工作区目录。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "fs.remove", Description: "删除 AGMP 工作区内的文件或目录。工作区根目录永远不可删除；递归删除目录属于破坏性操作并交由现有审批模式决定是否执行。", Risk: xiaoyucontract.RiskDestructive,
				Category: "filesystem", Manual: true, XiaoYu: true, Source: "agmp.files",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{"path": map[string]any{"type": "string"}, "recursive": map[string]any{"type": "boolean"}}, "required": []string{"path"}, "additionalProperties": false},
			}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if a.workspaceFiles == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("AGMP 工作区文件服务不可用")
				}
				path, _ := args["path"].(string)
				recursive, _ := args["recursive"].(bool)
				value, err := a.workspaceFiles.Remove(path, recursive)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: "已删除工作区路径。", Data: value}, nil
			}); err != nil {
				return nil, err
			}
			return nil, nil
		}},
		xiaoyuhost.PluginFunc{Meta: xiaoyuhost.Manifest{ID: "agmp.terminal", Name: "AGMP Controlled Terminal", Version: Version, Source: "agmp", Kinds: []xiaoyuhost.CapabilityKind{xiaoyuhost.CapabilityTool, xiaoyuhost.CapabilitySandbox}, Requires: []string{"workspace.files"}, Provides: []string{"general.shell"}}, ApplyFn: func(_ context.Context, host *xiaoyuhost.Context) (xiaoyuhost.DisposeFunc, error) {
			// Codex-style principle: shell is a general fallback capability, not a
			// hidden developer-only escape hatch. Domain tools remain preferred,
			// while RBAC + step-up + the existing three approval modes decide if
			// a concrete command may actually execute.
			shellHandler := func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if !a.platformConfig.Permissions.AllowArbitraryShell {
					return xiaoyucontract.ToolExecution{}, errors.New("管理员已显式关闭 XiaoYu 通用 Shell 能力")
				}
				command, _ := args["command"].(string)
				cwd, _ := args["cwd"].(string)
				result, err := a.runAuthorizedShellTool(ctx, command, cwd)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: fmt.Sprintf("受控 Shell 命令执行完成，退出码 %d。", result.ExitCode), Data: result}, nil
			}
			compatHandler := func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if !a.platformConfig.Permissions.AllowArbitraryShell {
					return xiaoyucontract.ToolExecution{}, errors.New("管理员已显式关闭 XiaoYu 通用 Shell 能力")
				}
				command, _ := args["command"].(string)
				cwd, _ := args["cwd"].(string)
				result, err := a.runApprovedShell(ctx, command, cwd)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				return xiaoyucontract.ToolExecution{Summary: fmt.Sprintf("兼容 Shell 命令执行完成，退出码 %d。", result.ExitCode), Data: result}, nil
			}
			schema := map[string]any{"type": "object", "properties": map[string]any{"command": map[string]any{"type": "string"}, "cwd": map[string]any{"type": "string"}}, "required": []string{"command"}, "additionalProperties": false}
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "shell.exec", Description: "通用受控 Shell 后备能力。没有合适的领域 Tool、或领域 Tool 无法覆盖异常情况时可自主使用；优先使用 Domain Tool。命令只能在 AGMP 工作区 CWD 中运行，真实执行仍受 RBAC、敏感操作 Step-up、三种审批模式、超时与输出限制保护。", Risk: xiaoyucontract.RiskSystem,
				Category: "terminal", Manual: true, XiaoYu: true, Source: "agmp.terminal", Parameters: schema,
			}, shellHandler); err != nil {
				return nil, err
			}
			// Compatibility alias for the existing manual terminal API. New XiaoYu
			// guidance should choose shell.exec; process.run remains available to
			// human/dev callers without duplicating the execution implementation.
			if err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "process.run", Description: "兼容旧版的受控命令入口；XiaoYu 应优先使用 shell.exec。", Risk: xiaoyucontract.RiskSystem,
				Category: "terminal", Manual: true, XiaoYu: false, Source: "agmp.terminal", Parameters: schema,
			}, compatHandler); err != nil {
				return nil, err
			}
			return nil, nil
		}},
		xiaoyuhost.PluginFunc{Meta: xiaoyuhost.Manifest{ID: "agmp.intelligence", Name: "XiaoYu Intelligence", Version: Version, Source: "agmp", Kinds: []xiaoyuhost.CapabilityKind{xiaoyuhost.CapabilityTool}, Provides: []string{"xiaoyu.memory"}}, ApplyFn: func(_ context.Context, host *xiaoyuhost.Context) (xiaoyuhost.DisposeFunc, error) {
			err := host.RegisterTool(xiaoyucontract.ToolSpec{
				Name: "memory.remember", Description: "把当前任务中明确且值得复用的信息写入当前成员的私人 XiaoYu Memory。不能发布组织知识、不能保存秘密、不能写入其他用户或其他 Run 的作用域。", Risk: xiaoyucontract.RiskModify,
				Category: "intelligence", Manual: false, XiaoYu: true, Source: "agmp.intelligence",
				Parameters: map[string]any{"type": "object", "properties": map[string]any{
					"scope":      map[string]any{"type": "string", "enum": []string{"user", "session", "task", "server", "instance", "experience"}},
					"content":    map[string]any{"type": "string", "minLength": 1, "maxLength": 16384},
					"confidence": map[string]any{"type": "number", "minimum": 0, "maximum": 1},
				}, "required": []string{"scope", "content"}, "additionalProperties": false},
			}, func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
				if a.xiaoyuIntelligence == nil {
					return xiaoyucontract.ToolExecution{}, errors.New("XiaoYu Intelligence Store 不可用")
				}
				invocation, ok := getXiaoYuInvocationContext(ctx)
				if !ok || strings.TrimSpace(invocation.User.ID) == "" || strings.TrimSpace(invocation.User.OrganizationID) == "" || strings.TrimSpace(invocation.RunID) == "" {
					return xiaoyucontract.ToolExecution{}, errors.New("memory.remember 只能在已认证、server-owned 的 XiaoYu Run 中使用")
				}
				scope, _ := args["scope"].(string)
				content, _ := args["content"].(string)
				confidence := 1.0
				if value, ok := args["confidence"].(float64); ok {
					confidence = value
				}
				request := xiaoyuhost.MemorySaveRequest{Content: content, Source: "xiaoyu", Confidence: confidence, Sensitivity: xiaoyuhost.MemoryNormal, Visibility: xiaoyuhost.VisibilityPrivate}
				switch strings.ToLower(strings.TrimSpace(scope)) {
				case "user":
					request.Kind = xiaoyuhost.MemoryUser
					request.Scope.ID = invocation.User.ID
				case "session":
					if invocation.RunContext.SessionID == "" {
						return xiaoyucontract.ToolExecution{}, errors.New("当前 Run 没有 session scope")
					}
					request.Kind = xiaoyuhost.MemorySession
					request.Scope.ID = invocation.RunContext.SessionID
				case "task":
					if invocation.RunContext.TaskID == "" || invocation.RunContext.TaskID != invocation.RunID {
						return xiaoyucontract.ToolExecution{}, errors.New("当前 Run 的 server-owned task scope 无效")
					}
					request.Kind = xiaoyuhost.MemoryTask
					request.Scope.ID = invocation.RunContext.TaskID
				case "server":
					if invocation.RunContext.ServerID == "" {
						return xiaoyucontract.ToolExecution{}, errors.New("当前 Run 没有 server scope")
					}
					request.Kind = xiaoyuhost.MemoryServer
					request.Scope.ID = invocation.RunContext.ServerID
				case "instance":
					if invocation.RunContext.InstanceID == "" {
						return xiaoyucontract.ToolExecution{}, errors.New("当前 Run 没有 instance scope")
					}
					request.Kind = xiaoyuhost.MemoryInstance
					request.Scope.ID = invocation.RunContext.InstanceID
				case "experience":
					request.Kind = xiaoyuhost.MemoryExperience
					request.Scope.ID = invocation.User.ID
				default:
					return xiaoyucontract.ToolExecution{}, errors.New("memory.remember scope 无效")
				}
				value, err := a.xiaoyuIntelligence.SaveMemory(invocation.User.OrganizationID, invocation.User.GroupID, invocation.User.ID, request)
				if err != nil {
					return xiaoyucontract.ToolExecution{}, err
				}
				a.recordOperation("info", "xiaoyu", "memory_remember", value.ID, "success", "小鱼经受控 Tool 写入私人 Memory（内容不写审计日志）", "", "")
				return xiaoyucontract.ToolExecution{Summary: "已把这条信息保存到当前成员的私人 XiaoYu Memory。", Data: map[string]any{"id": value.ID, "kind": value.Kind, "scope": value.Scope, "visibility": value.Visibility}}, nil
			})
			return nil, err
		}},
		xiaoyuhost.PluginFunc{Meta: xiaoyuhost.Manifest{ID: "agmp.games", Name: "AGMP Game Discovery", Version: Version, Source: "agmp", Kinds: []xiaoyuhost.CapabilityKind{xiaoyuhost.CapabilityTool}}, ApplyFn: func(_ context.Context, host *xiaoyuhost.Context) (xiaoyuhost.DisposeFunc, error) {
			for _, item := range []struct {
				spec    xiaoyucontract.ToolSpec
				handler xiaoyucontract.ToolHandler
			}{
				{xiaoyucontract.ToolSpec{Name: "games.list", Description: "列出当前 AGMP 已注册的游戏能力。", Risk: xiaoyucontract.RiskRead, Category: "games", Manual: true, XiaoYu: true, Source: "agmp.games", Parameters: emptyObjectSchema()}, func(_ context.Context, _ map[string]any) (xiaoyucontract.ToolExecution, error) {
					return xiaoyucontract.ToolExecution{Summary: "已读取 AGMP 游戏能力列表。", Data: a.platformConfig.Games.Templates}, nil
				}},
				{xiaoyucontract.ToolSpec{Name: "steam.snapshot", Description: "读取 Steam/SteamCMD 与已安装游戏的当前环境快照。", Risk: xiaoyucontract.RiskRead, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: emptyObjectSchema()}, func(_ context.Context, _ map[string]any) (xiaoyucontract.ToolExecution, error) {
					value, err := a.SteamSnapshot()
					return xiaoyucontract.ToolExecution{Summary: "已读取 Steam 环境快照。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.status", Description: "读取 AGMP 当前运行环境与依赖安装状态。", Risk: xiaoyucontract.RiskRead, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: emptyObjectSchema()}, func(_ context.Context, _ map[string]any) (xiaoyucontract.ToolExecution, error) {
					return xiaoyucontract.ToolExecution{Summary: "已读取 AGMP 环境状态。", Data: a.EnvironmentSetupStatus()}, nil
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.catalog", Description: "读取 Java/SteamCMD Runtime Registry 与服务端平台信息。", Risk: xiaoyucontract.RiskRead, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: emptyObjectSchema()}, func(_ context.Context, _ map[string]any) (xiaoyucontract.ToolExecution, error) {
					return xiaoyucontract.ToolExecution{Summary: "已读取 Runtime Registry。", Data: a.EnvironmentRuntimeCatalog()}, nil
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.game_profile", Description: "检查指定游戏所需 Runtime；Minecraft 可指定 Java 主版本，DST 只要求自己的 SteamCMD/系统依赖。", Risk: xiaoyucontract.RiskRead, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: map[string]any{"type": "object", "properties": map[string]any{"gameId": map[string]any{"type": "string"}, "javaMajor": map[string]any{"type": "integer"}}, "required": []string{"gameId"}, "additionalProperties": false}}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request environmentservice.GameRuntimeProfileRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					return xiaoyucontract.ToolExecution{Summary: "已检查游戏 Runtime Profile。", Data: a.EnvironmentGameRuntimeProfile(request)}, nil
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.resolve_runtime", Description: "从 AGMP Runtime Registry 解析可用 Java/SteamCMD Runtime；优先用它获取受管 Runtime，而不是猜测系统路径。", Risk: xiaoyucontract.RiskRead, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: map[string]any{"type": "object", "properties": map[string]any{"kind": map[string]any{"type": "string", "enum": []string{"java", "steamcmd"}}, "major": map[string]any{"type": "integer"}}, "required": []string{"kind"}, "additionalProperties": false}}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request environmentservice.ResolveRuntimeRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.ResolveEnvironmentRuntime(request)
					return xiaoyucontract.ToolExecution{Summary: "已解析受管 Runtime。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.set_default_runtime", Description: "把 Runtime Registry 中指定 Runtime 设为该类型默认版本。", Risk: xiaoyucontract.RiskModify, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}, "additionalProperties": false}}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request environmentservice.SetDefaultRuntimeRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.SetDefaultEnvironmentRuntime(request)
					return xiaoyucontract.ToolExecution{Summary: "默认 Runtime 已更新。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.remove_runtime", Description: "移除 Runtime Registry 中指定 Runtime。AGMP 自己管理且位于受管 RuntimeRoot 内的安装会同时安全删除对应目录；外部自定义路径只解除注册，不作为任意目录删除入口。", Risk: xiaoyucontract.RiskDestructive, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string"}}, "required": []string{"id"}, "additionalProperties": false}}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request environmentservice.RemoveRuntimeRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					if err := a.RemoveEnvironmentRuntime(request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					return xiaoyucontract.ToolExecution{Summary: "Runtime 已从 AGMP 管理范围移除。", Data: map[string]any{"id": request.ID}}, nil
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.install_java", Description: "通过 AGMP Runtime Manager 安装并校验指定 Java 版本。优先使用此结构化能力；只有它无法覆盖的特殊情况才考虑通用 Shell。", Risk: xiaoyucontract.RiskModify, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: map[string]any{"type": "object", "properties": map[string]any{"major": map[string]any{"type": "integer", "enum": []int{8, 17, 21, 25}}, "targetRoot": map[string]any{"type": "string"}, "setDefault": map[string]any{"type": "boolean"}}, "required": []string{"major"}, "additionalProperties": false}}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request environmentservice.JavaInstallRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.InstallJavaRuntime(request)
					return xiaoyucontract.ToolExecution{Summary: "Java Runtime 安装请求已执行。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.install_steamcmd", Description: "通过 AGMP Runtime Manager 安装 Windows/Linux 官方 SteamCMD。", Risk: xiaoyucontract.RiskModify, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: map[string]any{"type": "object", "properties": map[string]any{"targetRoot": map[string]any{"type": "string"}}, "additionalProperties": false}}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request environmentservice.SteamCMDInstallRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.InstallSteamCMDAt(request)
					return xiaoyucontract.ToolExecution{Summary: "SteamCMD 安装请求已执行。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "environment.install_system_prerequisite", Description: "安装 Game Runtime Profile 明确声明的 Linux 系统前置依赖。属于系统级高风险动作，本 Tool 只安装固定白名单包并经过 AGMP 审批/二次验证；若任务超出此 Tool 覆盖范围，小鱼仍可在审批策略允许时使用通用 Shell。", Risk: xiaoyucontract.RiskSystem, Category: "deployment", Manual: true, XiaoYu: true, Source: "agmp.deploy", Parameters: map[string]any{"type": "object", "properties": map[string]any{"id": map[string]any{"type": "string", "enum": []string{"steamcmd-linux-32bit-loader", "steamcmd-linux-32bit-cpp"}}}, "required": []string{"id"}, "additionalProperties": false}}, func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request environmentservice.InstallSystemPrerequisiteRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.InstallEnvironmentSystemPrerequisite(request)
					return xiaoyucontract.ToolExecution{Summary: "Linux 系统 Runtime 前置依赖安装请求已执行。", Data: value}, err
				}},
			} {
				if err := host.RegisterTool(item.spec, item.handler); err != nil {
					return nil, err
				}
			}
			return nil, nil
		}},
		xiaoyuhost.PluginFunc{Meta: xiaoyuhost.Manifest{ID: "agmp.dst", Name: "AGMP DST Capability", Version: Version, Source: "agmp", Kinds: []xiaoyuhost.CapabilityKind{xiaoyuhost.CapabilityTool}}, ApplyFn: func(_ context.Context, host *xiaoyuhost.Context) (xiaoyuhost.DisposeFunc, error) {
			clusterSchema := map[string]any{"type": "object", "properties": map[string]any{"clusterPath": map[string]any{"type": "string"}}, "required": []string{"clusterPath"}, "additionalProperties": false}
			startSchema := map[string]any{"type": "object", "properties": map[string]any{"clusterPath": map[string]any{"type": "string"}, "ugcDirectory": map[string]any{"type": "string"}}, "required": []string{"clusterPath"}, "additionalProperties": false}
			commandSchema := map[string]any{"type": "object", "properties": map[string]any{"clusterPath": map[string]any{"type": "string"}, "shardName": map[string]any{"type": "string"}, "command": map[string]any{"type": "string"}}, "required": []string{"clusterPath", "command"}, "additionalProperties": false}
			logsSchema := map[string]any{"type": "object", "properties": map[string]any{"clusterPath": map[string]any{"type": "string"}, "shardName": map[string]any{"type": "string"}, "after": map[string]any{"type": "integer"}, "limit": map[string]any{"type": "integer"}}, "required": []string{"clusterPath"}, "additionalProperties": false}
			registrations := []struct {
				spec    xiaoyucontract.ToolSpec
				handler xiaoyucontract.ToolHandler
			}{
				{xiaoyucontract.ToolSpec{Name: "dst.workspace", Description: "读取 DST 安装、存档和专用服务器工作区快照。", Risk: xiaoyucontract.RiskRead, Category: "dst", Manual: true, XiaoYu: true, Source: "agmp.dst", Parameters: emptyObjectSchema()}, func(_ context.Context, _ map[string]any) (xiaoyucontract.ToolExecution, error) {
					value, err := a.DSTWorkspaceSnapshot()
					return xiaoyucontract.ToolExecution{Summary: "已读取 DST 工作区状态。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "dst.cluster.status", Description: "读取指定 DST Cluster 的 Master/Caves 运行与端口状态。", Risk: xiaoyucontract.RiskRead, Category: "dst", Manual: true, XiaoYu: true, Source: "agmp.dst", Parameters: clusterSchema}, func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request dstruntimecore.ClusterRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.dstClusterStatus(ctx, request)
					return xiaoyucontract.ToolExecution{Summary: "已读取 DST Cluster 运行状态。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "dst.cluster.start", Description: "启动指定 DST Cluster；Master/Caves 的顺序与共享 Runtime 由 AGMP 管理。", Risk: xiaoyucontract.RiskOperate, Category: "dst", Manual: true, XiaoYu: true, Source: "agmp.dst", Parameters: startSchema}, func(ctx context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request dstruntimecore.StartClusterRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.startDSTCluster(ctx, request)
					return xiaoyucontract.ToolExecution{Summary: "DST Cluster 启动请求已执行。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "dst.cluster.stop", Description: "按 Caves → Master 顺序优雅停止指定 DST Cluster。", Risk: xiaoyucontract.RiskModify, Category: "dst", Manual: true, XiaoYu: true, Source: "agmp.dst", Parameters: clusterSchema}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request dstruntimecore.ClusterRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.StopDSTCluster(request)
					return xiaoyucontract.ToolExecution{Summary: "DST Cluster 停止请求已执行。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "dst.command", Description: "向 AGMP 管理的 DST shard 控制台发送游戏命令。", Risk: xiaoyucontract.RiskModify, Category: "dst", Manual: true, XiaoYu: true, Source: "agmp.dst", Parameters: commandSchema}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request dstruntimecore.CommandRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.SendDSTCommand(request)
					return xiaoyucontract.ToolExecution{Summary: "DST 控制台命令已发送。", Data: value}, err
				}},
				{xiaoyucontract.ToolSpec{Name: "dst.logs.tail", Description: "读取 AGMP 当前托管 DST shard 的最近控制台输出。", Risk: xiaoyucontract.RiskRead, Category: "dst", Manual: true, XiaoYu: true, Source: "agmp.dst", Parameters: logsSchema}, func(_ context.Context, args map[string]any) (xiaoyucontract.ToolExecution, error) {
					var request dstruntimecore.LogRequest
					if err := decodeToolArgs(args, &request); err != nil {
						return xiaoyucontract.ToolExecution{}, err
					}
					value, err := a.DSTProcessLogs(request)
					return xiaoyucontract.ToolExecution{Summary: "已读取 DST 控制台输出。", Data: value}, err
				}},
			}
			for _, item := range registrations {
				if err := host.RegisterTool(item.spec, item.handler); err != nil {
					return nil, err
				}
			}
			return nil, nil
		}},
	}
	for _, plugin := range plugins {
		if err := a.xiaoyuHost.Add(plugin); err != nil && !errors.Is(err, xiaoyuhost.ErrPluginExists) {
			panic(fmt.Sprintf("register XiaoYu capability plugin: %v", err))
		}
	}
	if err := a.xiaoyuHost.MountAll(context.Background()); err != nil {
		panic(fmt.Sprintf("mount XiaoYu capability plugins: %v", err))
	}
}

type xiaoyuUINavigationDestination struct {
	ID    string
	Label string
	Path  string
}

var xiaoyuUINavigationDestinations = []xiaoyuUINavigationDestination{
	{ID: "xiaoyu", Label: "小鱼", Path: "/"},
	{ID: "dashboard", Label: "仪表盘", Path: "/dashboard"},
	{ID: "deployment", Label: "一键部署", Path: "/deployment"},
	{ID: "instances", Label: "实例管理", Path: "/instances"},
	{ID: "tasks", Label: "任务中心", Path: "/tasks"},
	{ID: "files", Label: "文件管理", Path: "/files"},
	{ID: "backups", Label: "备份恢复", Path: "/backups"},
	{ID: "plugins", Label: "插件扩展", Path: "/plugins"},
	{ID: "network", Label: "网络与组网", Path: "/network"},
	{ID: "logs", Label: "日志中心", Path: "/logs"},
	{ID: "nodes", Label: "节点管理", Path: "/nodes"},
	{ID: "users", Label: "用户与权限", Path: "/users"},
	{ID: "developer", Label: "开发者工具", Path: "/developer"},
	{ID: "settings", Label: "设置中心", Path: "/settings"},
	{ID: "settings.models", Label: "设置中心 · 模型管理", Path: "/settings?section=models"},
	{ID: "settings.intelligence", Label: "设置中心 · 记忆、技能与专家", Path: "/settings?section=intelligence"},
	{ID: "settings.email", Label: "设置中心 · 邮箱服务", Path: "/settings?section=email"},
	{ID: "settings.license", Label: "设置中心 · 授权与许可证", Path: "/settings?section=license"},
	{ID: "settings.environment", Label: "设置中心 · 运行环境与存储", Path: "/settings?section=environment"},
	{ID: "settings.updates", Label: "设置中心 · 更新与升级", Path: "/settings?section=updates"},
	{ID: "settings.diagnostics", Label: "设置中心 · 开发诊断", Path: "/settings?section=diagnostics"},
}

func xiaoyuUINavigationTargetIDs() []string {
	result := make([]string, 0, len(xiaoyuUINavigationDestinations))
	for _, item := range xiaoyuUINavigationDestinations {
		result = append(result, item.ID)
	}
	return result
}

func xiaoyuUINavigationTarget(id string) (xiaoyuUINavigationDestination, bool) {
	id = strings.TrimSpace(id)
	for _, item := range xiaoyuUINavigationDestinations {
		if item.ID == id {
			return item, true
		}
	}
	return xiaoyuUINavigationDestination{}, false
}

func emptyObjectSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": false}
}

func decodeToolArgs(args map[string]any, target any) error {
	raw, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("编码 Tool 参数失败：%w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("解析 Tool 参数失败：%w", err)
	}
	return nil
}

func (a *Application) xiaoyuToolSpecs() []xiaoyuruntime.ToolSpec {
	if a.xiaoyuTools == nil {
		return []xiaoyuruntime.ToolSpec{}
	}
	specs := a.xiaoyuTools.List()
	result := make([]xiaoyuruntime.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		result = append(result, xiaoyuruntime.ToolSpec{Name: spec.Name, Description: spec.Description, Risk: string(spec.Risk), Category: spec.Category, Manual: spec.Manual, XiaoYu: spec.XiaoYu, Parameters: spec.Parameters, Source: spec.Source})
	}
	return result
}

func (a *Application) runApprovedShell(parent context.Context, command, cwd string) (platformruntime.RunResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return platformruntime.RunResult{ExitCode: -1}, errors.New("命令不能为空")
	}
	if a.workspaceFiles == nil {
		return platformruntime.RunResult{ExitCode: -1}, errors.New("AGMP 工作区文件服务不可用")
	}
	resolvedCWD, err := a.workspaceFiles.Resolve(cwd)
	if err != nil {
		return platformruntime.RunResult{ExitCode: -1}, fmt.Errorf("工作目录无效：%w", err)
	}
	if parent == nil {
		parent = context.Background()
	}
	timeout := time.Duration(maxInt(a.platformConfig.AI.ToolTimeoutSeconds, 30)) * time.Second
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	result, runErr := platformruntime.RunShell(ctx, platformruntime.ShellSpec{Command: command, WorkingDirectory: resolvedCWD, MaxOutputBytes: 512 * 1024})
	if result.TimedOut {
		return result, fmt.Errorf("命令执行超时：%w", ctx.Err())
	}
	if errors.Is(runErr, platformruntime.ErrOutputTruncated) {
		// A verbose command is still useful evidence. Keep the bounded output and
		// let XiaoYu narrow the next query instead of turning truncation into an
		// artificial task failure.
		result.Truncated = true
		return result, nil
	}
	if runErr != nil && result.ExitCode < 0 {
		return result, runErr
	}
	return result, nil
}
