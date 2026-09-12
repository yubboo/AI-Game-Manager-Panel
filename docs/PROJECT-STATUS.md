# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.14 — Windows ConPTY / Cross-platform Native Terminal**

## Stable baseline

0.2.11 is the latest fully green three-job baseline. 0.2.12 and 0.2.13 advanced the Rust Terminal Runtime; 0.2.13 GitHub Actions passed Windows Helper, Linux Headless, Go test/vet and all structural gates, with the Safety job stopped only by one rustfmt import-order difference before Rust check/PTY integration could run.

## 0.2.14 changes

- Fix the remaining 0.2.13 `cargo fmt --check` import ordering difference.
- Add `pty_windows.rs` using Windows ConPTY APIs behind the existing Terminal contract.
- Windows terminal backend reports `windows-conpty-v1`; Linux remains `linux-pty-v1`; only unsupported/unverified platforms retain `stdio-pipe-v1`.
- ConPTY supports persistent input/output and Host-authorized resize without creating a second Terminal API.
- Add a dedicated `Windows Rust Runtime + ConPTY` GitHub Actions job that performs Windows Rust format/check/tests and an actual ConPTY integration test.
- Keep all Terminal start/input/resize operations behind Go Host identity, RBAC and approval authority.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and Terminal Runtime. Linux PTY is implemented; Windows ConPTY is implemented in 0.2.14 and must be considered verified only after the dedicated Windows Runner is green.

## Next runtime milestones

1. Wire approved interactive Agent actions to the native Terminal Runtime.
2. Sandbox / capability leases / filesystem scope.
3. Apply Patch / generic filesystem mutation.
4. Reflection / experience pipeline.
5. Subagent / specialist dispatch.

## Verification status

Local packaging runs Node gates and Go compatibility checks where available. GitHub Actions is authoritative for Rust `fmt/check/test`, Go 1.25/Wails, locked pnpm builds, Linux native PTY integration, Windows ConPTY integration and Linux headless integration.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
