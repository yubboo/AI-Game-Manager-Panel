# AI Game Manager Panel · Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.11 — Persistent Rust Runtime Worker**

## Stable baseline

0.2.8 was the first all-green GitHub Actions baseline. 0.2.9 froze language ownership and dependency locks. 0.2.10 added Rust Session/Long-running Job primitives; its Linux Headless and Windows Helper jobs pass, while Safety reaches Rust formatting and reports only rustfmt-normalization differences in the newly added Rust files. 0.2.11 contains those exact formatting corrections.

## 0.2.11 changes

- Go now supervises one long-lived `xiaoyu rpc` child instead of spawning a new Rust process for every RPC.
- Tool Search, Brain policy, Session Registry and Job Runtime now share the same Rust process lifetime.
- Application startup prewarms the Rust worker; shutdown closes it.
- RPC timeout or broken stdio kills the bad worker; the next call can establish a clean worker.
- Worker stderr is bounded to 64 KiB to avoid unbounded diagnostic memory growth.
- Go exposes Host-internal Session/Job bridge methods while preserving Host authorization as mandatory.
- `sync-agmp.ps1` suppresses Robocopy OEM console output and uses a Unicode diagnostic log, eliminating mojibake for Chinese paths.
- New `check-xiaoyu-worker.mjs` protects the persistent-worker lifecycle and safety boundary.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust now owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs and their persistent worker lifetime. The next migration step is to route **approved** long-running Agent operations through Rust Jobs, then add PTY.

## Next runtime milestones

1. Wire approved long-running operations to Rust Jobs.
2. PTY / interactive terminal runtime.
3. Sandbox / capability leases / filesystem scope.
4. Apply Patch / generic filesystem mutation.
5. Reflection / experience pipeline.
6. Subagent / specialist dispatch.

## Verification status

Local packaging can run Node gates and Go compatibility checks where the toolchain permits. Full Rust `fmt/check/test`, locked pnpm builds, Go 1.25/Wails and Linux headless integration remain GitHub Actions authority.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
7. `docs/PROJECT-HISTORY.md` only for historical context.
