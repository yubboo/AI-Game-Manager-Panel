# AI Game Manager Panel (AGMP) 0.4.0

> 说一句话，就能把服务器开起来。

AGMP 0.4.0 is the architecture reset to **TypeScript Agent + Rust Native**. Go/Wails core code is removed.

## XiaoYu definition

XiaoYu is **not a separate weaker AI model**. It is the AGMP product identity around the vendor model selected by the user. If the user configures GPT, Claude, Gemini, DeepSeek or another supported provider, that configured model is the reasoning brain. AGMP adds Tools, Skills, Memory/session context, safety boundaries and Minecraft expertise around that model.

## Architecture

- `packages/agent/` — TypeScript Agent Loop / RunManager
- `packages/model/` — vendor model adapters and verified model profiles
- `packages/tools/` — Tool Registry / Approval / Rust Native bridge tools
- `packages/minecraft/` — Minecraft live facts, deployment and Modrinth tools
- `packages/instances/` — persistent GameInstance shared by AI and visual management
- `apps/server/` — TypeScript HTTP/SSE core
- `crates/native-runtime/` — Rust filesystem/process/network/Java/capability kernel
- `frontend/` — existing Vue three-column UI, intentionally preserved during migration
- `desktop/electron/` — desktop shell for the TypeScript Core

## Current 0.4.0 capability slice

Implemented in the new core:

- verified model profile and real tool-call probe before a model can become the XiaoYu default brain
- vendor-model Agent Loop: model → tool → observation → same model
- pause/cancel/takeover + event trace/SSE
- Approval + single-use Rust Capability Lease
- isolated data/workspace/native runtime roots
- Mojang live version/Java facts
- managed Temurin Java provisioning in Rust
- port preflight
- Vanilla verified download/config/start/readiness/MC status ping
- Paper live resolve + SHA-256 verified server download
- Fabric live loader/installer resolution
- Modrinth search/resolve/dependency closure + install-time authoritative re-fetch/SHA-1 verification
- persistent GameInstance lifecycle shared with `/api/v1/instances`

Not claimed complete yet: Fabric server artifact installation, public tunnel delivery, full mod-source coverage, cross-platform real smoke until GitHub Actions runs.

## Source checks

```bash
npm install
npm run typecheck
npm run test:core
npm run gate
```

Rust:

```bash
cargo test --manifest-path crates/Cargo.toml --workspace
```

Frontend:

```bash
cd frontend
corepack pnpm install
corepack pnpm build
```
