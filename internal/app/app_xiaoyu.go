package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	authservice "github.com/yubboo/AI-Game-Manager-Panel/internal/system/auth"
	xiaoyucontract "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/contract"
	xiaoyucontrol "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/control"
	xiaoyuhost "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/host"
	xiaoyuruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/runtime"
)

func (a *Application) XiaoYuRuntimeStatus() xiaoyuruntime.Status {
	status := a.xiaoyuRuntime.Status()
	if a.xiaoyuTools != nil {
		status.ToolCount = len(a.xiaoyuTools.List())
	}
	return status
}

func (a *Application) requireXiaoYuMember(token string) (authservice.User, error) {
	return a.RequireLicensedMemberFeature(token, "ai.workbench")
}

// XiaoYuTools 返回 AI Core Tool Registry。Tool 的业务能力仍必须落在所属模块，Agent 只能通过公开 Tool/Service 契约调用。
func (a *Application) XiaoYuTools(token string) ([]xiaoyuruntime.ToolSpec, error) {
	if _, err := a.requireXiaoYuMember(token); err != nil {
		return nil, err
	}
	return a.xiaoyuToolSpecs(), nil
}

// XiaoYuApprovalState 返回后端唯一可信的 AI 审批模式。前端本地值不能提升 Agent 权限。
func (a *Application) XiaoYuApprovalState(token string) (xiaoyucontrol.Snapshot, error) {
	if _, err := a.requireXiaoYuMember(token); err != nil {
		return xiaoyucontrol.Snapshot{}, err
	}
	if err := a.xiaoyuApprovals.Ready(); err != nil {
		return xiaoyucontrol.Snapshot{}, fmt.Errorf("AI 审批状态不可用：%w", err)
	}
	return a.xiaoyuApprovals.Snapshot(), nil
}

// SetXiaoYuApprovalMode 只允许最高管理员切换“请求批准 / 帮我批准 / 完全访问权限”。
func (a *Application) SetXiaoYuApprovalMode(token string, request xiaoyucontrol.SetModeRequest) (xiaoyucontrol.Snapshot, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyucontrol.Snapshot{}, err
	}
	if user.Role != authservice.RoleOwner {
		return xiaoyucontrol.Snapshot{}, authservice.ErrForbidden
	}
	mode := xiaoyucontrol.Mode(strings.TrimSpace(request.Mode))
	if !xiaoyucontrol.ValidMode(mode) {
		return xiaoyucontrol.Snapshot{}, fmt.Errorf("未知审批模式：%s", request.Mode)
	}
	if mode == xiaoyucontrol.ModeFull && !a.platformConfig.Permissions.AllowFullAccessMode {
		return xiaoyucontrol.Snapshot{}, errors.New("当前平台策略禁止开启完全访问权限")
	}
	state, err := a.xiaoyuApprovals.SetMode(mode, user.Username)
	if err != nil {
		return xiaoyucontrol.Snapshot{}, err
	}
	a.recordOperation("info", "agent", "approval_mode", string(mode), "success", "最高管理员已更新 AI 审批模式", "", "")
	return state, nil
}

// XiaoYuPendingApprovals 返回仍在等待人工决定的请求。它们没有自动超时，未决定前 Agent 不得继续对应步骤。
func (a *Application) XiaoYuPendingApprovals(token string) ([]xiaoyucontrol.Request, error) {
	user, err := a.requireXiaoYuApprover(token)
	if err != nil {
		return nil, err
	}
	_ = user
	return a.xiaoyuApprovals.Pending()
}

// ResolveXiaoYuApproval 对一条待审批请求做明确批准/拒绝。批准只绑定原请求指纹且只能消费一次。
func (a *Application) ResolveXiaoYuApproval(token, id string, request xiaoyucontrol.ResolveRequest) (xiaoyucontrol.Request, error) {
	user, err := a.requireXiaoYuApprover(token)
	if err != nil {
		return xiaoyucontrol.Request{}, err
	}
	decision := strings.TrimSpace(strings.ToLower(request.Decision))
	if decision != "approve" && decision != "reject" {
		return xiaoyucontrol.Request{}, errors.New("审批决定必须是 approve 或 reject")
	}
	item, err := a.xiaoyuApprovals.Resolve(id, decision == "approve", user.Username)
	if err != nil {
		return xiaoyucontrol.Request{}, err
	}
	a.recordOperation("info", "agent", "approval", item.Subject, "success", fmt.Sprintf("AI 审批已%s", map[bool]string{true: "批准", false: "拒绝"}[decision == "approve"]), "", "")
	return item, nil
}

// XiaoYuCallTool 是 Planner 与 AGMP 模块能力之间的受控入口。
// 权限模式只读取后端持久状态；任何客户端传来的“full/approved”字段都不会被信任。
func (a *Application) XiaoYuCallTool(token string, request xiaoyuruntime.ToolCallRequest) (xiaoyuruntime.ToolCallResult, error) {
	return a.xiaoyuCallToolForRun(a.Context(), token, "", request)
}

func (a *Application) xiaoyuCallToolForRun(ctx context.Context, token, runID string, request xiaoyuruntime.ToolCallRequest) (xiaoyuruntime.ToolCallResult, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return xiaoyuruntime.ToolCallResult{}, err
	}
	if err := a.xiaoyuApprovals.Ready(); err != nil {
		return xiaoyuruntime.ToolCallResult{}, fmt.Errorf("AI 审批状态不可用：%w", err)
	}
	if a.xiaoyuTools == nil {
		return xiaoyuruntime.ToolCallResult{}, errors.New("AGMP Tool Registry 不可用")
	}
	spec, ok := a.xiaoyuTools.Get(request.Tool)
	if !ok {
		return xiaoyuruntime.ToolCallResult{}, fmt.Errorf("未注册 AGMP Tool：%s", strings.TrimSpace(request.Tool))
	}
	xiaoyuCallable := spec.XiaoYu
	// 兼容旧版 process.run；0.2.2 起模型应使用 shell.exec。通用
	// Shell 是 XiaoYu 的基础后备能力，但管理员仍可用配置作为全局 kill
	// switch。具体命令是否执行继续由 RBAC / step-up / Approval 决定。
	if spec.Name == "process.run" && a.platformConfig.Permissions.AllowArbitraryShell {
		xiaoyuCallable = true
	}
	if spec.Name == "shell.exec" && !a.platformConfig.Permissions.AllowArbitraryShell {
		return xiaoyuruntime.ToolCallResult{}, errors.New("管理员已显式关闭 XiaoYu 通用 Shell 能力")
	}
	if !xiaoyuCallable {
		return xiaoyuruntime.ToolCallResult{}, fmt.Errorf("Tool %s 未开放给小鱼 Planner；请使用当前可用能力", spec.Name)
	}

	risk := spec.Risk
	if !xiaoyuRoleAllows(user.Role, risk) {
		a.observeXiaoYuRun("tool/rbac-denied", runID, spec.Name, map[string]any{"risk": string(risk), "role": user.Role})
		return xiaoyuruntime.ToolCallResult{Tool: spec.Name, Decision: "deny", Summary: "当前账号角色没有执行该级别操作的权限。", Risk: string(risk)}, nil
	}
	if risk == xiaoyucontract.RiskDestructive || risk == xiaoyucontract.RiskSystem {
		if _, stepErr := a.auth.RequireSensitiveAction(token, "XiaoYu 高风险 Tool："+spec.Name); stepErr != nil {
			a.observeXiaoYuRun("tool/step-up-required", runID, spec.Name, map[string]any{"risk": string(risk), "role": user.Role})
			return xiaoyuruntime.ToolCallResult{Tool: spec.Name, Decision: "deny", Summary: stepErr.Error(), Risk: string(risk)}, nil
		}
	}
	a.observeXiaoYuRun("tool/request", runID, spec.Name, map[string]any{"risk": string(risk), "source": spec.Source, "role": user.Role})
	mode := a.xiaoyuApprovals.Mode()
	decision := a.xiaoyuPermissionPolicy(mode).Decide(risk)
	if decision == xiaoyucontrol.DecisionDeny {
		a.observeXiaoYuRun("tool/denied", runID, spec.Name, map[string]any{"risk": string(risk)})
		return xiaoyuruntime.ToolCallResult{Tool: spec.Name, Decision: "deny", Summary: "当前 AGMP 安全策略禁止执行该操作。", Risk: string(risk)}, nil
	}

	// Approval is turn/run scoped, matching Codex-style approval semantics:
	// a pending approval created for one Run must never authorize an identical
	// Tool call from another Run.
	fingerprintSubject := spec.Name
	if strings.TrimSpace(runID) != "" {
		fingerprintSubject += "#run:" + strings.TrimSpace(runID)
	}
	fingerprint, err := xiaoyucontrol.Fingerprint(xiaoyucontrol.KindTool, fingerprintSubject, request.Arguments)
	if err != nil {
		return xiaoyuruntime.ToolCallResult{}, fmt.Errorf("生成 Tool 审批指纹失败：%w", err)
	}
	if decision == xiaoyucontrol.DecisionConfirm {
		if strings.TrimSpace(request.ApprovalID) == "" {
			pending, err := a.xiaoyuApprovals.CreatePending(xiaoyucontrol.KindTool, spec.Name, string(risk), "AI 请求执行 AGMP 模块操作，等待用户确认。", fingerprint)
			if err != nil {
				return xiaoyuruntime.ToolCallResult{}, err
			}
			a.observeXiaoYuRun("tool/waiting-approval", runID, spec.Name, map[string]any{"risk": string(risk), "approvalId": pending.ID})
			return xiaoyuruntime.ToolCallResult{Tool: spec.Name, Decision: "confirm", Summary: "需要用户批准后才能继续；未回答时将保持等待。", ApprovalID: pending.ID, Pending: true, Risk: string(risk)}, nil
		}
		if err := a.xiaoyuApprovals.ConsumeApproval(request.ApprovalID, fingerprint); err != nil {
			if errors.Is(err, xiaoyucontrol.ErrApprovalPending) {
				return xiaoyuruntime.ToolCallResult{Tool: spec.Name, Decision: "confirm", Summary: "仍在等待用户批准。", ApprovalID: request.ApprovalID, Pending: true, Risk: string(risk)}, nil
			}
			if errors.Is(err, xiaoyucontrol.ErrApprovalRejected) {
				a.observeXiaoYuRun("tool/rejected", runID, spec.Name, map[string]any{"risk": string(risk), "approvalId": request.ApprovalID})
				return xiaoyuruntime.ToolCallResult{Tool: spec.Name, Decision: "deny", Summary: "用户已拒绝该操作。", ApprovalID: request.ApprovalID, Risk: string(risk)}, nil
			}
			return xiaoyuruntime.ToolCallResult{}, err
		}
	}

	if ctx == nil {
		ctx = a.Context()
	}
	// Domain Tool handlers receive only the already-authenticated principal and
	// the server-owned Run context. They never trust user-supplied identity or
	// task scope fields inside Tool arguments.
	runContext := xiaoyuhost.RunContext{}
	if strings.TrimSpace(runID) != "" && a.xiaoyuRuns != nil {
		if run, ok := a.xiaoyuRuns.Get(runID); ok {
			runContext = run.Context
		}
	}
	ctx = withXiaoYuInvocationContext(ctx, xiaoyuInvocationContext{User: user, RunID: strings.TrimSpace(runID), RunContext: runContext})
	execution, err := a.xiaoyuTools.Execute(ctx, spec.Name, request.Arguments)
	if err != nil {
		a.recordOperation("warn", "agent", "tool", spec.Name, "failed", err.Error(), "", "")
		a.observeXiaoYuRun("tool/failed", runID, spec.Name, map[string]any{"risk": string(risk), "error": safeXiaoYuTraceText(err.Error())})
		return xiaoyuruntime.ToolCallResult{Tool: spec.Name, Decision: "allow", Risk: string(risk)}, err
	}
	result := xiaoyuruntime.ToolCallResult{
		Tool: spec.Name, Decision: "allow", Summary: execution.Summary, Data: execution.Data, Risk: string(risk),
	}
	a.recordOperation("info", "agent", "tool", spec.Name, "success", "AGMP 模块 Tool 执行完成", "", "")
	a.observeXiaoYuRun("tool/completed", runID, spec.Name, map[string]any{"risk": string(risk)})
	return result, nil
}

// XiaoYuRunCommand 是人工终端与内部 Runtime 共用的受控 Shell 入口。
// 完整命令绝不进入操作审计；审批文件也只保存不可逆请求指纹。
func (a *Application) XiaoYuRunCommand(token string, request xiaoyuruntime.CommandRequest) (xiaoyuruntime.CommandResult, error) {
	if _, err := a.requireXiaoYuApprover(token); err != nil {
		return xiaoyuruntime.CommandResult{}, err
	}
	if _, err := a.auth.RequireSensitiveAction(token, "手动系统终端命令"); err != nil {
		return xiaoyuruntime.CommandResult{}, err
	}
	if err := a.xiaoyuApprovals.Ready(); err != nil {
		return xiaoyuruntime.CommandResult{}, fmt.Errorf("AI 审批状态不可用：%w", err)
	}
	mode := a.xiaoyuApprovals.Mode()
	decision := a.xiaoyuPermissionPolicy(mode).Decide(xiaoyucontract.RiskSystem)
	if decision == xiaoyucontrol.DecisionDeny {
		return xiaoyuruntime.CommandResult{Command: request.Command, ExitCode: -1, Decision: "deny", Stderr: "当前 AGMP 安全策略禁止执行系统命令。"}, nil
	}
	fingerprint, err := xiaoyucontrol.Fingerprint(xiaoyucontrol.KindCommand, "manual.shell", map[string]string{
		"command": strings.TrimSpace(request.Command),
		"cwd":     strings.TrimSpace(request.WorkingDirectory),
	})
	if err != nil {
		return xiaoyuruntime.CommandResult{}, fmt.Errorf("生成命令审批指纹失败：%w", err)
	}
	if decision == xiaoyucontrol.DecisionConfirm {
		if strings.TrimSpace(request.ApprovalID) == "" {
			pending, err := a.xiaoyuApprovals.CreatePending(xiaoyucontrol.KindCommand, "manual.shell", string(xiaoyucontract.RiskSystem), "请求执行手动终端命令，等待用户确认。", fingerprint)
			if err != nil {
				return xiaoyuruntime.CommandResult{}, err
			}
			return xiaoyuruntime.CommandResult{Command: request.Command, ExitCode: -1, Decision: "confirm", Stderr: "需要用户批准后才能继续；未回答时将保持等待。", ApprovalID: pending.ID, Pending: true}, nil
		}
		if err := a.xiaoyuApprovals.ConsumeApproval(request.ApprovalID, fingerprint); err != nil {
			if errors.Is(err, xiaoyucontrol.ErrApprovalPending) {
				return xiaoyuruntime.CommandResult{Command: request.Command, ExitCode: -1, Decision: "confirm", Stderr: "仍在等待用户批准。", ApprovalID: request.ApprovalID, Pending: true}, nil
			}
			if errors.Is(err, xiaoyucontrol.ErrApprovalRejected) {
				return xiaoyuruntime.CommandResult{Command: request.Command, ExitCode: -1, Decision: "deny", Stderr: "用户已拒绝该命令。", ApprovalID: request.ApprovalID}, nil
			}
			return xiaoyuruntime.CommandResult{}, err
		}
	}
	runResult, err := a.runApprovedShell(a.Context(), request.Command, request.WorkingDirectory)
	result := xiaoyuruntime.CommandResult{
		Command: request.Command, Cwd: strings.TrimSpace(request.WorkingDirectory), ExitCode: runResult.ExitCode,
		Stdout: runResult.Stdout, Stderr: runResult.Stderr, Decision: "allow",
	}
	if a.workspaceFiles != nil {
		if resolved, resolveErr := a.workspaceFiles.Resolve(request.WorkingDirectory); resolveErr == nil {
			result.Cwd = resolved
		}
	}
	if err != nil {
		a.recordOperation("warn", "agent", "command", "manual.shell", "failed", "手动终端命令执行失败", "", "")
		return result, err
	}
	a.recordOperation("info", "agent", "command", "manual.shell", "success", fmt.Sprintf("共享 Runtime exit=%d", result.ExitCode), "", "")
	return result, nil
}

func xiaoyuRoleAllows(role string, risk xiaoyucontract.RiskLevel) bool {
	switch role {
	case authservice.RoleOwner, authservice.RoleAdministrator:
		return true
	case authservice.RoleOperator:
		return risk == xiaoyucontract.RiskRead || risk == xiaoyucontract.RiskOperate || risk == xiaoyucontract.RiskModify
	default:
		return false
	}
}

func (a *Application) xiaoyuPermissionPolicy(mode xiaoyucontrol.Mode) xiaoyucontrol.Policy {
	cfg := a.platformConfig.Permissions
	return xiaoyucontrol.Policy{
		Mode: mode,
		Ask:  cfg.Policies.Ask,
		Risk: cfg.Policies.Risk,
		Full: cfg.Policies.Full,
	}
}

func (a *Application) observeXiaoYu(kind, summary string, data map[string]any) {
	if a.xiaoyuHost == nil {
		return
	}
	a.xiaoyuHost.Observe(xiaoyuhost.Event{Type: kind, Summary: safeXiaoYuTraceText(summary), Data: data})
}

func (a *Application) observeXiaoYuRun(kind, runID, summary string, data map[string]any) {
	if a.xiaoyuHost == nil {
		return
	}
	a.xiaoyuHost.Observe(xiaoyuhost.Event{Type: kind, RunID: strings.TrimSpace(runID), Summary: safeXiaoYuTraceText(summary), Data: data})
}

func safeXiaoYuTraceText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 240 {
		return value[:240] + "…"
	}
	return value
}

func (a *Application) requireXiaoYuApprover(token string) (authservice.User, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return authservice.User{}, err
	}
	if user.Role != authservice.RoleOwner && user.Role != authservice.RoleAdministrator {
		return authservice.User{}, authservice.ErrForbidden
	}
	return user, nil
}

// AuthBootstrapStatus 返回首次启动认证状态。公开注册仅在最高管理员不存在时开放。

// XiaoYuCapabilities exposes the current host-side capability graph. This is
// what lets XiaoYu discover which "hands and feet" are actually mounted now.
func (a *Application) XiaoYuCapabilities(token string) (xiaoyuhost.CapabilitySnapshot, error) {
	if _, err := a.requireXiaoYuMember(token); err != nil {
		return xiaoyuhost.CapabilitySnapshot{}, err
	}
	if a.xiaoyuHost == nil {
		return xiaoyuhost.CapabilitySnapshot{}, errors.New("XiaoYu Host 不可用")
	}
	return a.xiaoyuHost.Capabilities(), nil
}

// XiaoYuTrace returns sanitized Harness lifecycle/run summaries. Raw secrets and
// complete command arguments are deliberately not stored in this trace.
func (a *Application) XiaoYuTrace(token string, after uint64, limit int) ([]xiaoyuhost.Event, error) {
	user, err := a.requireXiaoYuMember(token)
	if err != nil {
		return nil, err
	}
	if a.xiaoyuHost == nil {
		return nil, errors.New("XiaoYu Host 不可用")
	}
	return a.xiaoyuHost.Trace().ListFiltered(after, limit, a.xiaoyuEventFilter(user)), nil
}

// XiaoYuDSHPlugins discovers pre-installed DeepSeek Harness bundles without
// executing them. Mount/install remains an explicit privileged operation.
func (a *Application) XiaoYuDSHPlugins(token string) ([]xiaoyuhost.DSHBundle, error) {
	if _, err := a.requireXiaoYuApprover(token); err != nil {
		return nil, err
	}
	if !a.platformConfig.AI.Harness.DSH.Enabled {
		return []xiaoyuhost.DSHBundle{}, nil
	}
	return xiaoyuhost.DiscoverDSHBundles(a.xiaoyuDSHRoot)
}

// MountXiaoYuDSHPlugin mounts a pre-installed DSH tool plugin. Loading external
// JavaScript executes third-party code, so this action is owner-only and the
// caller must explicitly acknowledge trust. Imported tools still pass through
// AGMP's normal Tool permission/approval path on every call.
func (a *Application) MountXiaoYuDSHPlugin(token string, request xiaoyuhost.DSHMountRequest) (xiaoyuhost.PluginSnapshot, error) {
	user, err := a.requireXiaoYuApprover(token)
	if err != nil {
		return xiaoyuhost.PluginSnapshot{}, err
	}
	if user.Role != authservice.RoleOwner {
		return xiaoyuhost.PluginSnapshot{}, authservice.ErrForbidden
	}
	if err := a.RequireLicenseFeature("plugin.extensions"); err != nil {
		return xiaoyuhost.PluginSnapshot{}, err
	}
	if !a.platformConfig.AI.Harness.DSH.Enabled {
		return xiaoyuhost.PluginSnapshot{}, errors.New("DeepSeek Harness 兼容桥未启用")
	}
	if !request.Trusted {
		return xiaoyuhost.PluginSnapshot{}, errors.New("加载第三方 DSH 插件前必须显式确认已信任其源码")
	}
	bundle, err := xiaoyuhost.ResolveDSHBundle(a.xiaoyuDSHRoot, request.Directory)
	if err != nil {
		return xiaoyuhost.PluginSnapshot{}, err
	}
	bridge := xiaoyuhost.NewDSHBridge(xiaoyuhost.DSHBridgeOptions{NodeExecutable: a.platformConfig.AI.Harness.DSH.NodeExecutable, Timeout: time.Duration(maxInt(a.platformConfig.AI.ToolTimeoutSeconds, 30)) * time.Second})
	plugin := xiaoyuhost.DSHPlugin{Bundle: bundle, Bridge: bridge, Policy: xiaoyuhost.DSHPolicy{Risk: xiaoyucontract.RiskSystem, Manual: true, XiaoYu: request.XiaoYu}, Config: request.Config}
	meta := plugin.Manifest()
	if err := a.xiaoyuHost.Add(plugin); err != nil {
		if !errors.Is(err, xiaoyuhost.ErrPluginExists) {
			return xiaoyuhost.PluginSnapshot{}, err
		}
		// A disposed/failed external plugin may be reloaded from disk with a
		// newer version or XiaoYu exposure policy. Active plugins stay stable.
		for _, existing := range a.xiaoyuHost.Plugins() {
			if existing.Manifest.ID != meta.ID {
				continue
			}
			if existing.State == xiaoyuhost.StateActive {
				return existing, nil
			}
			if err := a.xiaoyuHost.Forget(meta.ID); err != nil {
				return xiaoyuhost.PluginSnapshot{}, err
			}
			if err := a.xiaoyuHost.Add(plugin); err != nil {
				return xiaoyuhost.PluginSnapshot{}, err
			}
			break
		}
	}
	if err := a.xiaoyuHost.Mount(a.Context(), meta.ID); err != nil {
		return xiaoyuhost.PluginSnapshot{}, err
	}
	for _, item := range a.xiaoyuHost.Plugins() {
		if item.Manifest.ID == meta.ID {
			a.recordOperation("warning", "xiaoyu", "mount_dsh_plugin", meta.ID, "success", "最高管理员已加载受信任 DSH 兼容插件", "", "")
			_ = user
			return item, nil
		}
	}
	return xiaoyuhost.PluginSnapshot{}, errors.New("DSH 插件已加载但未找到状态快照")
}

func (a *Application) UnmountXiaoYuPlugin(token, id string) error {
	user, err := a.requireXiaoYuApprover(token)
	if err != nil {
		return err
	}
	if user.Role != authservice.RoleOwner {
		return authservice.ErrForbidden
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("插件 ID 不能为空")
	}
	if strings.HasPrefix(id, "agmp.") {
		return errors.New("AGMP 内置核心能力不能通过外部插件接口卸载")
	}
	if err := a.xiaoyuHost.Unmount(a.Context(), id); err != nil {
		return err
	}
	if err := a.xiaoyuHost.Forget(id); err != nil {
		return err
	}
	a.recordOperation("warning", "xiaoyu", "unmount_plugin", id, "success", "已卸载并释放 XiaoYu 外部能力插件", "", "")
	return nil
}
