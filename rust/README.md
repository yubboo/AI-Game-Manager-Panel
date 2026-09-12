# AGMP XiaoYu Rust Runtime

0.2.9 起，Rust 的长期职责从“Brain-only”升级为 **XiaoYu Agent Runtime / Native Execution / Security Boundary**。

Rust 负责 XiaoYu 的通用 Agent 能力，Go 继续负责 AGMP 游戏/产品 Domain Service。详细语言职责见 `docs/architecture/LANGUAGE-OWNERSHIP.md`。

## 当前 crate

```text
rust/crates/xiaoyu-core       # Agent Runtime 主实现
rust/crates/xiaoyu-protocol   # xiaoyu.v1 协议
```

当前已经包含：

- provider-neutral Brain Policy / Decision Grammar；
- Guided Autonomy；
- Session Registry（create/get/list/close，0.2.10）；
- Approval hint；
- Tool Search / Capability Discovery（0.2.9）；
- Long-running Job Runtime（start/status/list/output/cancel，0.2.10）；
- bounded output / cancellation / Runtime Root cwd boundary；
- JSON-RPC stdio Runtime。

后续 Rust-first 能力：

- Agent Loop / Goal State；
- Context / Thread；
- PTY；
- Generic Shell / File / Process；
- Apply Patch；
- Sandbox / Capability Lease；
- Reflection / Experience；
- Subagent。

## Domain 边界

Rust 不复制：

- Steam / SteamCMD 业务；
- DST / Minecraft 业务规则；
- Instance / Backup / License / Updater 业务。

这些仍由 Go Domain Service 提供结构化 Tool。

## 安全

Rust Native Runtime 不等于无限权限。`jobs/start` 只是 Host 授权后的内部原语，不能直接等同于模型执行权。任何真实执行都必须保持：身份/RBAC、三种审批模式、Capability Scope、Sandbox、路径/参数限制、秘密保护、审计和执行后验证。

普通用户不需要安装 Rust/Cargo/MSVC。发行构建机/CI 预编译 XiaoYu Runtime，并把它作为 AGMP 内部组件随完整产品发布。

## 0.2.11 Persistent Worker

AGMP Go Host now keeps one supervised `xiaoyu rpc` process alive. Tool Search, Brain policy, Session Registry and Long-running Jobs share this process lifetime. The Worker remains an embedded AGMP component; Host RBAC/approval stays authoritative.

## 0.2.17 Windows ConPTY Runtime Convergence

0.2.16 passed Windows `cargo check` and moved the remaining failure into the real ConPTY integration test. The runner showed two Windows-only semantics: canonical Rust paths such as `\\?\D:\...` are not suitable as `cmd.exe` current directories, and ConPTY interactive Enter should be represented as CRLF rather than the LF used by Unix PTYs. 0.2.17 normalizes the cwd only at the Win32 process-spawn boundary, makes newline handling backend-aware, removes a stale mutex-drop pattern, and cfg-gates the pipe-only process variant. No new Agent privilege is introduced.

## 0.2.15 Native Terminal CI Convergence

0.2.15 intentionally adds no new Terminal permissions. It applies the exact rustfmt fixes reported by the 0.2.14 runners and changes CI so `cargo check`, Linux/Windows native terminal integration, and workspace tests still run when formatting has already failed, unless the workflow is cancelled.

This prevents a formatting error from hiding a Windows compile or ConPTY runtime error. Cross-platform Native Terminal remains provisional until both Linux PTY and Windows ConPTY integration lanes are actually green.

## 0.2.14 Windows ConPTY / Cross-platform Native Terminal

`xiaoyu-core::terminal` now has native terminal backends on the two primary platforms: Linux keeps `linux-pty-v1`, while Windows uses `windows-conpty-v1` implemented with `CreatePseudoConsole`, extended startup attributes and `ResizePseudoConsole`. The public JSON-RPC Terminal contract stays unchanged.

A dedicated `Windows Rust Runtime + ConPTY` CI job builds the Rust workspace on `windows-latest` and runs a ConPTY integration probe. Start, input and resize remain Host-authorized; native terminal support is an execution primitive, not a permission bypass.

## 0.2.13 Linux Native PTY Foundation

`xiaoyu-core::terminal` keeps the same stateful Terminal RPC contract introduced in 0.2.12, but Linux now uses a real PTY backend (`linux-pty-v1`) created directly by Rust. The child gets its own session and controlling terminal; output remains bounded, input stays Host-authorized, and `terminal/resize` updates the kernel window size. Linux tests verify both TTY detection and resize behavior.

Windows and other non-Linux builds still report the explicit `stdio-pipe-v1` fallback. 0.2.13 does **not** claim Windows ConPTY support. The next step is a dedicated Windows Rust CI lane plus a verified ConPTY backend behind this same protocol.
