# AI Game Manager Panel · Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.12 — Rust Interactive Terminal Session**

## Stable baseline

0.2.11 is the current all-green GitHub Actions baseline: Windows Helper, Safety, Linux Headless, Go test/vet, Rust fmt/check/test, Agent Bench and Persistent Worker all pass. 0.2.12 builds only on that stable worker and does not rewrite Go Domain Services.

## 0.2.12 changes

- Rust adds a stateful `TerminalManager` for long-lived interactive stdin/stdout sessions.
- JSON-RPC adds `terminal/start|get|list|write|output|close`.
- Terminal output is bounded and incrementally readable with cursors.
- Both terminal start and every terminal input require Host authorization in Go and Rust.
- Runtime Root/cwd restrictions reuse the existing Session boundary.
- Terminal Snapshot reports `backend=stdio-pipe-v1`; 0.2.12 deliberately does **not** claim native PTY/ConPTY.
- New `check-xiaoyu-terminal.mjs` protects protocol, Host authorization, bounded output and the no-fake-PTY rule.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and Interactive Terminal Session v1. Terminal RPC remains Host-internal until Agent Bench and native PTY semantics are ready.

## Next runtime milestones

1. Native PTY backend: Windows ConPTY + Unix PTY, keeping the same Terminal protocol.
2. Wire approved interactive Agent actions to the Terminal Runtime.
3. Sandbox / capability leases / filesystem scope.
4. Apply Patch / generic filesystem mutation.
5. Reflection / experience pipeline.
6. Subagent / specialist dispatch.

## Verification status

Local packaging runs Node gates and Go compatibility checks where available. GitHub Actions remains authoritative for Rust `fmt/check/test`, Go 1.25/Wails, locked pnpm builds and Linux headless integration.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
7. `docs/PROJECT-HISTORY.md` only for historical context.
