# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.18 — Windows ConPTY Input Convergence**

## Stable baseline

0.2.17 proved that the Windows ConPTY ABI, `CreateProcessW` cwd normalization, resize lock scope and native/fallback compile boundaries reach the real Windows GitHub Runner. Windows `cargo fmt` and `cargo check --workspace --locked` pass. Safety, Linux native PTY, Linux Headless and Windows Helper remain green.

The remaining failure is narrower: the ConPTY process starts and shows the correct repository cwd, but commands written with CRLF are not executed as Enter. The official ConPTY/terminal input behavior uses a single CR (`\r`, `0x0D`) for the Enter key.

## 0.2.18 changes

- Windows ConPTY `appendNewline` sends a single CR (`\r`) instead of CRLF.
- Linux PTY and fallback backends continue to use LF (`\n`).
- Rename the Windows newline unit test to freeze CR semantics and update the Terminal/PTY source Gate accordingly.
- Preserve the 0.2.17 cwd normalization, resize lock lifetime and fallback cfg fixes.
- Freeze the development handoff: inspect GitHub first, deliver a complete `agmp-<version>.zip` + SHA-256, run `AGMP-Sync.bat`, then `AGMP-GitHub.bat -> 1. 一键推送`.
- Do not add new model-visible execution privileges in this release.

## Mandatory GitHub baseline workflow

Before starting a new version, after the user reports a push, and before preparing the next source bundle, AI/developers must proactively inspect `yubboo/AI-Game-Manager-Panel` on GitHub: confirm `main` latest commit, inspect the matching GitHub Actions run/jobs/logs, and use those results as the next change baseline. Do not wait for the user to remind the AI to check GitHub.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and Terminal Runtime. Linux native PTY is CI-proven. Windows ConPTY is compile/start/cwd-proven and is waiting for input integration/workspace tests to turn green.

## Next runtime milestones

1. Obtain a fully green `Windows Rust Runtime + ConPTY` lane.
2. Wire approved interactive Agent actions to the native Terminal Runtime.
3. Sandbox / capability leases / filesystem scope.
4. Apply Patch / generic filesystem mutation.
5. Reflection / experience pipeline.
6. Subagent / specialist dispatch.

## Verification status

Local packaging runs project Node gates and Go compatibility checks where available. If Cargo is installed on the Windows development machine, `AGMP-GitHub` additionally runs Rust fmt + workspace check before push. GitHub Actions remains authoritative for Windows ConPTY integration and cross-platform Rust tests.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
