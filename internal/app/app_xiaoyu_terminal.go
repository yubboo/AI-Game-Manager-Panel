package app

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"

	platformruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/platform/runtime"
	xiaoyucontrol "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/control"
	xiaoyuruntime "github.com/yubboo/AI-Game-Manager-Panel/internal/xiaoyu/runtime"
)

const (
	approvedAgentTerminalOutputLimit = 512 * 1024
	approvedAgentLeaseScope          = xiaoyucontrol.ScopeProcessExecWorkspaceCWD
)

// runAuthorizedShellTool preserves the existing manual shell Tool contract,
// while server-owned XiaoYu Runs use the Rust Native Terminal after the Host
// policy/approval boundary has authorized the concrete Tool call.
func (a *Application) runAuthorizedShellTool(parent context.Context, command, cwd string) (platformruntime.RunResult, error) {
	if parent != nil {
		if invocation, ok := getXiaoYuInvocationContext(parent); ok && strings.TrimSpace(invocation.RunID) != "" {
			return a.runApprovedAgentTerminal(parent, command, cwd)
		}
	}
	return a.runApprovedShell(parent, command, cwd)
}

// runApprovedAgentTerminal is the only model-facing shell path migrated to the
// Rust Native Terminal Runtime. It must be called only after xiaoyuCallToolForRun
// has completed identity, RBAC, sensitive-action step-up and approval-policy
// checks. Manual shell/process compatibility paths remain on platform/runtime.
func (a *Application) runApprovedAgentTerminal(parent context.Context, command, cwd string) (platformruntime.RunResult, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return platformruntime.RunResult{ExitCode: -1}, errors.New("命令不能为空")
	}
	if parent == nil {
		parent = context.Background()
	}
	invocation, ok := getXiaoYuInvocationContext(parent)
	if !ok || strings.TrimSpace(invocation.RunID) == "" {
		return platformruntime.RunResult{ExitCode: -1}, errors.New("Native Terminal 仅接受 server-owned XiaoYu Run 中已通过 Host 授权的动作")
	}
	if a.xiaoyuLeases == nil || invocation.Lease == nil {
		return platformruntime.RunResult{ExitCode: -1}, errors.New("Native Terminal 缺少短时 Capability Lease")
	}
	leaseHash, err := approvedAgentLeaseHash(invocation.RunID, command, cwd)
	if err != nil {
		return platformruntime.RunResult{ExitCode: -1}, fmt.Errorf("生成 Native Terminal 能力租约指纹失败：%w", err)
	}
	principal := xiaoyuLeasePrincipal(invocation.User)
	if err := a.xiaoyuLeases.Consume(invocation.Lease.ID, approvedAgentLeaseScope, "shell.exec", invocation.RunID, principal, leaseHash); err != nil {
		return platformruntime.RunResult{ExitCode: -1}, fmt.Errorf("Native Terminal Capability Lease 无效：%w", err)
	}
	if a.workspaceFiles == nil {
		return platformruntime.RunResult{ExitCode: -1}, errors.New("AGMP 工作区文件服务不可用")
	}
	if a.xiaoyuRuntime == nil {
		return platformruntime.RunResult{ExitCode: -1}, errors.New("XiaoYu Native Terminal Runtime 不可用")
	}
	resolvedCWD, err := a.workspaceFiles.Resolve(cwd)
	if err != nil {
		return platformruntime.RunResult{ExitCode: -1}, fmt.Errorf("工作目录无效：%w", err)
	}
	timeout := time.Duration(maxInt(a.platformConfig.AI.ToolTimeoutSeconds, 30)) * time.Second
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	executable, arguments := approvedAgentShell(command)
	terminal, err := a.xiaoyuRuntime.StartTerminal(ctx, xiaoyuruntime.TerminalStartRequest{
		CapabilityLeaseID: invocation.Lease.ID,
		CapabilityScope:   string(approvedAgentLeaseScope),
		Executable:        executable,
		Arguments:         arguments,
		Cwd:               resolvedCWD,
		MaxOutputBytes:    approvedAgentTerminalOutputLimit,
		Rows:              24,
		Cols:              120,
		HostAuthorized:    true,
	})
	if err != nil {
		return platformruntime.RunResult{ExitCode: -1}, fmt.Errorf("启动 XiaoYu Native Terminal 失败：%w", err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cleanupCancel()
		_, _ = a.xiaoyuRuntime.CloseTerminal(cleanupCtx, terminal.ID)
	}()
	a.observeXiaoYuRun("tool/native-terminal-started", invocation.RunID, "shell.exec", map[string]any{"terminalId": terminal.ID, "backend": terminal.Backend})

	var output strings.Builder
	cursor := uint64(0)
	truncated := terminal.OutputTruncated
	last := terminal
	for {
		chunks, outputErr := a.xiaoyuRuntime.TerminalOutput(ctx, terminal.ID, cursor, 1000)
		if outputErr != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return platformruntime.RunResult{Stdout: output.String(), ExitCode: -1, TimedOut: true, Truncated: truncated}, fmt.Errorf("命令执行超时：%w", ctx.Err())
			}
			return platformruntime.RunResult{Stdout: output.String(), ExitCode: -1, Truncated: truncated}, fmt.Errorf("读取 XiaoYu Native Terminal 输出失败：%w", outputErr)
		}
		cursor = chunks.NextCursor
		truncated = truncated || chunks.Truncated
		for _, chunk := range chunks.Chunks {
			appendBoundedTerminalOutput(&output, chunk.Text, approvedAgentTerminalOutputLimit, &truncated)
		}

		last, err = a.xiaoyuRuntime.GetTerminal(ctx, terminal.ID)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return platformruntime.RunResult{Stdout: output.String(), ExitCode: -1, TimedOut: true, Truncated: truncated}, fmt.Errorf("命令执行超时：%w", ctx.Err())
			}
			return platformruntime.RunResult{Stdout: output.String(), ExitCode: -1, Truncated: truncated}, fmt.Errorf("读取 XiaoYu Native Terminal 状态失败：%w", err)
		}
		truncated = truncated || last.OutputTruncated
		if last.State == "exited" || last.State == "closed" {
			break
		}
		select {
		case <-ctx.Done():
			return platformruntime.RunResult{Stdout: output.String(), ExitCode: -1, TimedOut: errors.Is(ctx.Err(), context.DeadlineExceeded), Truncated: truncated}, fmt.Errorf("命令执行超时：%w", ctx.Err())
		case <-time.After(20 * time.Millisecond):
		}
	}

	exitCode := -1
	if last.ExitCode != nil {
		exitCode = *last.ExitCode
	}
	result := platformruntime.RunResult{Stdout: output.String(), ExitCode: exitCode, Truncated: truncated}
	a.observeXiaoYuRun("tool/native-terminal-completed", invocation.RunID, "shell.exec", map[string]any{"terminalId": terminal.ID, "backend": terminal.Backend, "exitCode": exitCode, "truncated": truncated})
	return result, nil
}

func approvedAgentLeaseHash(runID, command, cwd string) (string, error) {
	return xiaoyucontrol.Fingerprint(
		xiaoyucontrol.KindTool,
		"shell.exec#run:"+strings.TrimSpace(runID),
		map[string]string{"command": strings.TrimSpace(command), "cwd": strings.TrimSpace(cwd)},
	)
}

func approvedAgentShell(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "powershell.exe", []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", command}
	}
	return "sh", []string{"-lc", command}
}

func appendBoundedTerminalOutput(target *strings.Builder, text string, limit int, truncated *bool) {
	if limit <= 0 || text == "" {
		return
	}
	remaining := limit - target.Len()
	if remaining <= 0 {
		*truncated = true
		return
	}
	if len(text) > remaining {
		end := remaining
		for end > 0 && end < len(text) && (text[end]&0xC0) == 0x80 {
			end--
		}
		if end > 0 {
			target.WriteString(text[:end])
		}
		*truncated = true
		return
	}
	target.WriteString(text)
}
