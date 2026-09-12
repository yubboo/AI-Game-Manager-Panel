# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.19 — Windows ConPTY Input Pipe Convergence**

## Stable baseline

0.2.18 confirmed that the complete source-delivery workflow, Windows PowerShell/helper encoding, Go tests/vet, Linux headless/Web/XiaoYu, Rust formatting/checking and Linux native PTY are green on GitHub. Windows ConPTY also starts `cmd.exe` in the correct repository cwd and the CR newline unit rule passes.

The remaining Windows failure is now isolated to process stdio routing under the GitHub/Rust test harness. The Runner log prints the `cmd.exe` banner and prompt directly into the parent test output while commands written to the ConPTY input pipe are ignored. Microsoft Terminal documents this redirected-parent case: without `STARTF_USESTDHANDLES`, Windows can duplicate the parent's standard handles into a console child even when `bInheritHandles` is false, bypassing the pseudoconsole communication pipes.

## 0.2.19 changes

- Build Windows ConPTY `STARTUPINFOEXW` with `STARTF_USESTDHANDLES`.
- Explicitly keep `hStdInput`, `hStdOutput` and `hStdError` null so the child cannot fall back to redirected/captured parent stdio.
- Preserve `bInheritHandles = false`, ConPTY cwd normalization, CR Enter semantics, resize locking and fallback cfg boundaries.
- Add a Windows unit regression for the startup flags/null std handles and freeze the rule in `check-xiaoyu-terminal.mjs`.
- Do not add new model-visible execution privileges in this release.

## Mandatory GitHub baseline workflow

Before starting a new version, after the user reports a push, and before preparing the next source bundle, AI/developers must proactively inspect `yubboo/AI-Game-Manager-Panel` on GitHub: confirm `main` latest commit, inspect the matching GitHub Actions run/jobs/logs, and use those results as the next change baseline. Do not wait for the user to remind the AI to check GitHub.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and Terminal Runtime. Linux native PTY is CI-proven. Windows ConPTY is compile/start/cwd-proven; 0.2.19 targets the remaining redirected-parent stdio isolation required for real input/output integration.

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
