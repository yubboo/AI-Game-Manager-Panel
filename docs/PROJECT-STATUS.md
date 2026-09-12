# AI Game Manager Panel · Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.10 — Rust Session / Long-running Job Runtime foundation**

## Stable baseline

0.2.8 is the first all-green GitHub Actions baseline. 0.2.9 froze language ownership and reproducible dependency locks; its push passed Go, Linux Headless, Windows Helper and every architecture gate, but Safety stopped at `cargo fmt --check` because the new Rust Tool Search files were not fully rustfmt-normalized. 0.2.10 includes that exact formatting correction.

## 0.2.10 changes

- Rust now owns a real in-process Session Registry: create/get/list/close with runtime-root cwd containment.
- Rust now owns a Long-running Job primitive: start/get/list/output/cancel, PID/exit state, bounded output and cancellation.
- `jobs/start` requires explicit Host authorization and is not model-visible by default.
- `xiaoyu rpc` keeps Session/Job state for the life of the persistent RPC process.
- New `check-xiaoyu-jobs.mjs` protects the Session/Job protocol, root boundary, bounded output and Host-authorization contract.
- 0.2.9 rustfmt CI failure is corrected.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- existing Agent Loop orchestration;
- current model-visible `shell.exec` and shared Go process runtime;
- AGMP Domain Tool registry and game/product services.

The Rust Session/Job APIs are currently runtime primitives. The next migration step is a **persistent Go ↔ Rust RPC worker** so approved XiaoYu operations can use these stateful primitives without spawning a fresh Rust process per RPC.

## Next runtime milestones

1. Persistent Go ↔ Rust RPC worker / lifecycle supervision.
2. Wire approved long-running operations to Rust Jobs.
3. PTY / interactive terminal runtime.
4. Sandbox / capability leases / filesystem scope.
5. Apply Patch / generic filesystem mutation.
6. Reflection / experience pipeline.
7. Subagent / specialist dispatch.

## Verification status

Local packaging can run Node gates and Go compatibility tests. Full Rust `fmt/check/test`, locked pnpm builds, Go 1.25/Wails and Linux headless integration remain GitHub Actions authority.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
7. `docs/PROJECT-HISTORY.md` only for historical context.
