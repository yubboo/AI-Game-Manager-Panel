# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.3.1 — Minecraft Web Build Hotfix**

## Stable baseline

0.2.23 is the fully green GitHub baseline at commit `a1892cae9e0412e56e3d57bbc2044b103c7e6985` (workflow run `34678472325`): `safety`, `Linux Headless + Web + XiaoYu`, `Windows Helper + Encoding`, and `Windows Rust Runtime + ConPTY` all completed successfully. Platform / Model Provider / Game Pack / GameInstance contracts are therefore frozen as the product-control baseline.

## 0.3.0 changes

- Promote `minecraft.java` from planned to supported on Windows/Linux as the first real Game Pack vertical slice.
- Add live Mojang/Paper/Fabric fact resolution, including Mojang `javaVersion`, latest stable selection and exact-version rejection.
- Add Vanilla/Paper/Fabric artifact resolution and integrity handling: official SHA1/SHA256 where upstream provides it, explicit local SHA256 recording where Fabric does not provide a full-jar hash.
- Reuse Environment Manager for managed Java resolution/installation and write `eula.txt`, `server.properties` and an AGMP manifest into an isolated instance directory.
- Add persistent cross-client `GameInstance` store plus Minecraft start/stop/status/logs/probe lifecycle. Host restart clears stale live health presentation instead of trusting persisted process state.
- Add local port preflight, canonical `Done (...)! For help` readiness evidence and Minecraft Java status protocol Ping; only Ready + Ping marks the deployment healthy.
- Add shared Visual/XiaoYu `game.deploy.plan` and `game.deploy` path. `onlineMode` is explicit, offline requires whitelist, and XiaoYu may never accept the Minecraft EULA on the user's behalf.
- Add Web/Wails APIs and Vue deployment/instance controls for plan preview, deploy, start/stop, logs and protocol status.
- Add Minecraft Vertical Slice Gate to Safety and Linux Headless CI.

## 0.2.23 changes

- Add a real Platform Contract exposed through Application, HTTP and Wails: Web stays browser/control-only; execution belongs to the target native Node Runtime.
- Mark current platform truth honestly: Web/Windows Desktop/Windows Runtime/Linux Runtime are current supported surfaces; macOS Desktop/Runtime remain planned until native build/CI exists.
- Add Model Provider auth/kind contracts: API Key, subscription and local authorization are no longer conflated.
- Add the `openai-codex` subscription Provider with official CLI `login status` probing only. It is intentionally `brainEligible=false` until Codex app-server mediation can preserve AGMP Tool/Approval/Lease authority.
- Add shared Game Pack and GameInstance contracts. Visual UI and XiaoYu expose the same resource records through HTTP/Wails and read-only Tools.
- Adopt real detected DST clusters into `GameInstance` instead of leaving the generic Instances page as a placeholder.
- Add Product Contracts Gate and Model Center regression rules to GitHub Safety/Headless CI.

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

## Next product milestones

1. Minecraft first real Game Pack vertical slice — current 0.3.0 stage: facts → managed Java → verified install → config → native start → Ready/Ping → GameInstance.
2. Minecraft Mod/Plugin capability and compatibility/dependency resolution.
3. Public connectivity/tunnel capability plus end-to-end external reachability evidence.
4. Game Pack SDK and second-game validation; adding a game must not require rewriting XiaoYu Core or the whole UI.
5. Filesystem/network sandbox scope continues where concrete enforcement is required; security work remains mandatory but does not block validated product slices.

## Verification status

Local packaging runs project Node gates and Go compatibility checks where available. If Cargo is installed on the Windows development machine, `AGMP-GitHub` additionally runs Rust fmt + workspace check before push. GitHub Actions remains authoritative for Windows ConPTY integration and cross-platform Rust tests.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
