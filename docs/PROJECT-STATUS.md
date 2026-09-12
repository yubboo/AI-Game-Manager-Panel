# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.15 — Native Terminal CI Convergence**

## Stable baseline

0.2.11 remains the latest fully green cross-job baseline. 0.2.14 has already proved that Windows Helper, Linux Headless, Go test/vet, Agent Bench and all static Terminal/PTY gates still pass. Its Safety and Windows Rust jobs both stopped at `cargo fmt --check` in `pty_windows.rs`, so Linux PTY and Windows ConPTY integration were skipped rather than failed.

## 0.2.15 changes

- Apply the exact two `pty_windows.rs` rustfmt changes reported by the 0.2.14 GitHub Runner.
- Keep Rust compile/integration/test steps running with `if: !cancelled()` even when format fails, so one CI run exposes all platform failures instead of hiding them behind rustfmt.
- Require the Terminal/PTY gate to preserve this CI convergence behavior for both Linux and Windows.
- Run local `cargo fmt --check` during `AGMP-GitHub` safety checks whenever Cargo exists; otherwise print an explicit CI-authoritative warning.
- Do not add Sandbox/Capability Lease or model-visible Native Terminal wiring until Linux PTY and Windows ConPTY integration actually pass.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and Terminal Runtime. Linux PTY and Windows ConPTY are implemented behind one Terminal contract, but cross-platform Native Terminal is not frozen as stable until both platform integration lanes are green.

## Next runtime milestones

1. First obtain green Rust fmt/check/tests + Linux PTY integration + Windows ConPTY integration.
2. Wire approved interactive Agent actions to the native Terminal Runtime.
3. Sandbox / capability leases / filesystem scope.
4. Apply Patch / generic filesystem mutation.
5. Reflection / experience pipeline.
6. Subagent / specialist dispatch.

## Verification status

Local packaging runs Node gates and Go compatibility checks where available. `AGMP-GitHub` additionally runs rustfmt when Cargo is installed. GitHub Actions remains authoritative for Rust `fmt/check/test`, Go 1.25/Wails, locked pnpm builds, Linux native PTY integration, Windows ConPTY integration and Linux headless integration.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`

## AI 开发阅读顺序

开始较大修改前依次阅读：`docs/PROJECT-STATUS.md` → `docs/PROJECT-ARCHITECTURE.md` → `docs/architecture/LANGUAGE-OWNERSHIP.md` → `docs/development/PROJECT-RULES.md` → `docs/DEVELOPMENT-PLAN.md`。历史细节只在需要时读取 `docs/PROJECT-HISTORY.md`。当前状态与长期规范优先于历史版本描述。
