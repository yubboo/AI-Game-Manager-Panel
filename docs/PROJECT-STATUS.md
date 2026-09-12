# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.16 — Windows ConPTY ABI Convergence**

## Stable baseline

0.2.15 proved the Linux side end-to-end: Safety is fully green, including `cargo fmt`, `cargo check`, Rust workspace tests and the real Linux native PTY integration test. Windows Helper and Linux Headless are also green. The only remaining red lane is `Windows Rust Runtime + ConPTY`, where the Windows Runner reached real Rust compilation and exposed two `windows-sys 0.61.2` ABI type mismatches in `pty_windows.rs`.

## 0.2.16 changes

- Cast `PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE` from its generated `u32` constant type to the `usize` required by `UpdateProcThreadAttribute`.
- Initialize `HPCON` with integer zero because `windows-sys 0.61.2` defines the alias as `isize`, not a raw pointer.
- Extend the Terminal/PTY source gate so these two Windows ABI contracts cannot silently regress.
- Add `pty_linux.rs` / `pty_windows.rs` to the local push helper's required source set.
- When Cargo exists locally, `AGMP-GitHub` now runs both `cargo fmt --check` and `cargo check --workspace --locked` before commit/push.
- Do not add new Agent permissions in this release; first require the Windows Runner to compile and execute the ConPTY integration test.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and Terminal Runtime. Linux native PTY is now CI-proven. Windows ConPTY is implemented and statically gated, but is not frozen as stable until the Windows Rust lane passes `cargo check`, `windows_terminal_` integration and workspace tests.

## Next runtime milestones

1. Obtain a fully green `Windows Rust Runtime + ConPTY` lane.
2. Wire approved interactive Agent actions to the native Terminal Runtime.
3. Sandbox / capability leases / filesystem scope.
4. Apply Patch / generic filesystem mutation.
5. Reflection / experience pipeline.
6. Subagent / specialist dispatch.

## Verification status

Local packaging runs project Node gates and Go compatibility checks where available. If Cargo is installed on the Windows development machine, `AGMP-GitHub` additionally runs Rust fmt + workspace check before push. GitHub Actions remains authoritative for Linux PTY and Windows ConPTY platform integration.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`

## AI 开发阅读顺序

开始较大修改前依次阅读：`docs/PROJECT-STATUS.md` → `docs/PROJECT-ARCHITECTURE.md` → `docs/architecture/LANGUAGE-OWNERSHIP.md` → `docs/development/PROJECT-RULES.md` → `docs/DEVELOPMENT-PLAN.md`。历史细节只在需要时读取 `docs/PROJECT-HISTORY.md`。当前状态与长期规范优先于历史版本描述。
