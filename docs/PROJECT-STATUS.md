# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.13 — Linux Native PTY Foundation**

## Stable baseline

0.2.11 is the latest fully green three-job baseline. 0.2.12 added the Terminal protocol and passed Windows Helper, Linux Headless, Go test/vet, Agent Bench and Terminal Gate; its only red step was two `cargo fmt --check` layout differences in `terminal.rs`. 0.2.13 fixes those differences and upgrades the Linux backend without rewriting Go Domain Services.

## 0.2.13 changes

- Linux Rust Runtime now creates a real PTY master/slave with a controlling terminal instead of a stdio-pipe emulation.
- Linux Terminal Snapshot reports `backend=linux-pty-v1`; non-Linux builds keep an explicit `stdio-pipe-v1` fallback until a native backend is verified.
- `terminal/resize` and rows/cols were added to the existing Terminal contract. Resize requires Host authorization just like terminal start and every input frame.
- Linux tests verify real TTY semantics with `test -t 0` and verify kernel window size propagation with `stty size`.
- PTY master descriptors are close-on-exec, output remains bounded, cwd remains inside Runtime Root / Session scope, and close terminates the PTY process group.
- `check-xiaoyu-terminal.mjs` now protects the Linux native PTY implementation while explicitly forbidding an unverified Windows ConPTY claim.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and Terminal Runtime. Linux has a native PTY backend; Windows remains an explicit stdio fallback in 0.2.13.

## Next runtime milestones

1. Windows ConPTY backend plus a dedicated Windows Rust compile/integration CI job.
2. Wire approved interactive Agent actions to the Terminal Runtime.
3. Sandbox / capability leases / filesystem scope.
4. Apply Patch / generic filesystem mutation.
5. Reflection / experience pipeline.
6. Subagent / specialist dispatch.

## Verification status

Local packaging runs Node gates and Go compatibility checks where available. GitHub Actions remains authoritative for Rust `fmt/check/test`, Go 1.25/Wails, locked pnpm builds, Linux native PTY integration and Linux headless integration. Windows ConPTY must not be advertised until a Windows Rust CI job builds and exercises it.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
