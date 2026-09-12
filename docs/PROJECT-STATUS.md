# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.21 — Sandbox / Capability Lease**

## Stable baseline

0.2.20 is now a fully green GitHub baseline: `safety`, `Linux Headless + Web + XiaoYu`, `Windows Helper + Encoding`, and `Windows Rust Runtime + ConPTY` all completed successfully. Naming Gate runtime exclusions, Approved Agent → Native Terminal wiring, Linux PTY and Windows ConPTY are CI-proven.

## 0.2.21 changes

- Add an in-memory Host-owned Capability Lease store; outstanding leases never persist across Host restart.
- Lease `shell.exec` only after identity, RBAC, sensitive-action step-up and approval-policy checks have authorized the concrete server-owned Run action.
- Bind each lease to scope, Tool, RunID, organization/user principal and irreversible request fingerprint; default TTL is 30 seconds and Native Terminal leases are single-use.
- Require exact lease consumption before `HostAuthorized=true` can reach Native Terminal. Missing, expired, mismatched, cross-Run, cross-principal or replayed leases fail closed.
- Carry `capabilityLeaseId` over Go↔Rust `terminal/start`; Rust rejects Host-authorized starts that omit the lease marker.
- Keep command payloads, credentials and secrets out of lease state; existing workspace CWD, timeout, bounded output and terminal cleanup remain unchanged.
- Add Capability Lease static Gate and Go/Rust regression tests.

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
