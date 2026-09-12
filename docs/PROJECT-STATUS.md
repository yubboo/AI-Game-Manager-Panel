# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.20 — Approved Agent Native Terminal Wiring**

## Stable baseline

0.2.19 is now a fully green GitHub baseline: `safety`, `Linux Headless + Web + XiaoYu`, `Windows Helper + Encoding`, and `Windows Rust Runtime + ConPTY` all completed successfully. Windows ConPTY integration and Windows Rust workspace tests are CI-proven, so cross-platform Native Terminal is frozen as a stable internal primitive.

## 0.2.20 changes

- Route server-owned XiaoYu `shell.exec` calls to Rust Native Terminal only after the existing Go Host identity/RBAC/step-up/approval pipeline has authorized the exact Tool call.
- Keep workspace CWD resolution in Go, set `HostAuthorized=true` only inside the Host bridge, bound captured PTY output to 512 KiB, inherit Tool timeout, and close the terminal on every path.
- Keep manual shell calls and the compatibility `process.run` path on the existing `platform/runtime` implementation for this migration step.
- Fail closed when the Native Terminal path is unavailable; do not silently re-execute an approved Agent action through the legacy runtime.
- Add Go wiring tests and extend the Terminal/PTY Gate so future changes cannot bypass the server-owned Run / Host authorization boundary.

## Mandatory GitHub baseline workflow

Before starting a new version, after the user reports a push, and before preparing the next source bundle, AI/developers must proactively inspect `yubboo/AI-Game-Manager-Panel` on GitHub: confirm `main` latest commit, inspect the matching GitHub Actions run/jobs/logs, and use those results as the next change baseline. Do not wait for the user to remind the AI to check GitHub.

## Current migration boundary

Still in Go for compatibility and authority:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- identity/RBAC/sensitive-action step-up/approval fingerprint enforcement;
- AGMP Domain Tool registry and game/product services;
- manual shell / `process.run` compatibility execution.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and the CI-proven cross-platform Native Terminal Runtime. In 0.2.20, only server-owned Agent `shell.exec` execution crosses into Native Terminal after Go Host authorization.

## Next runtime milestones

1. Freeze Approved Agent → Native Terminal wiring on GitHub Actions.
2. Sandbox / capability leases / filesystem scope.
3. Apply Patch / generic filesystem mutation.
4. Reflection / experience pipeline.
5. Subagent / specialist dispatch.

## Verification status

Local packaging runs project Node gates and Go compatibility checks where available. If Cargo is installed on the Windows development machine, `AGMP-GitHub` additionally runs Rust fmt + workspace check before push. GitHub Actions remains authoritative for Windows ConPTY integration and cross-platform Rust tests.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
