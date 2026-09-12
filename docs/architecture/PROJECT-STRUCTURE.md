# AGMP 0.4 Project Structure

- `apps/server/` — TypeScript HTTP/SSE product core
- `packages/agent/` — Agent Loop + RunManager
- `packages/model/` — vendor model adapters and verified model profiles
- `packages/tools/` — tool registry, approval, Native tools
- `packages/minecraft/` — Minecraft live-fact and domain tools
- `packages/instances/` — shared GameInstance state
- `packages/skills/` + `skills/` — on-demand decision guides
- `crates/native-protocol/` — TS/Rust Native RPC contract
- `crates/native-runtime/` — Rust safety/system kernel
- `frontend/` — existing Vue/Codex UI; preserved during migration
- `desktop/electron/` — native desktop shell starting TypeScript Core

Runtime separation: `runtime/data` (auth/model secrets/sessions), `runtime/workspace` (model-managed servers), `runtime/native` (Rust-managed runtimes such as Java).
