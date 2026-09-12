# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.22 — Capability Scope / Web Asset Safety**

## Stable baseline

0.2.21 is the fully green GitHub baseline at commit `9349d845338e45889998ee3ebd5031c71673f050`: `safety`, `Linux Headless + Web + XiaoYu`, `Windows Helper + Encoding`, and `Windows Rust Runtime + ConPTY` all completed successfully. Capability Lease, Approved Agent → Native Terminal, Linux PTY and Windows ConPTY are CI-proven.

## 0.2.22 changes

- Replace free-form execution lease scope with the typed `process.exec:workspace-cwd` Capability Scope; unknown scopes are rejected by the Host lease store.
- Carry `capabilityScope` together with `capabilityLeaseId` over Go↔Rust `terminal/start`.
- Reject missing or drifted process scope in both the Go runtime bridge and Rust Native Terminal before process spawn.
- Keep the meaning honest: workspace-resolved CWD is enforced, but this is not filesystem or network isolation for the child process.
- Mark `cmd/aigame-manager-web/web/assets/` as generated content-hash output and keep it out of Git.
- Allow one-click push deletion protection to skip only that generated assets directory while preserving strict protection for `web/index.html` and all other `cmd/` source.
- Extend GitHub Safety / Windows Helper / Capability Lease / Terminal gates and regression tests for these rules.

## Mandatory GitHub baseline workflow

Before starting a new version, after the user reports a push, and before preparing the next source bundle, AI/developers must proactively inspect `yubboo/AI-Game-Manager-Panel` on GitHub: confirm `main` latest commit, inspect the matching GitHub Actions run/jobs/logs, and use those results as the next change baseline. Do not wait for the user to remind the AI to check GitHub.

## Current migration boundary

Still in Go for compatibility and authority:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- identity/RBAC/sensitive-action step-up/approval fingerprint enforcement;
- AGMP Domain Tool registry and game/product services;
- manual shell / `process.run` compatibility execution.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and the CI-proven cross-platform Native Terminal Runtime. Server-owned Agent `shell.exec` crosses into Native Terminal only after Go Host authorization, a single-use Capability Lease and the exact `process.exec:workspace-cwd` scope.

## Next runtime milestones

1. Typed process-exec Capability Scope — current 0.2.22 stage.
2. Filesystem capability scope with real path-operation enforcement.
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
