# AGMP Development Contract — 0.4+

## Locked architecture
- TypeScript is the primary product language: Agent Loop, model adapters, context/session, MCP/plugins, workflows, Web API and product orchestration.
- Rust is the Native Kernel: process/PTY, filesystem confinement, downloads/hash, Java runtime, networking, Minecraft protocol, sandbox and Capability enforcement.
- Go core is removed. Do not reintroduce Go/Wails into the product core without an explicit architecture decision.
- Existing Codex-style three-column UI is frozen during the 0.4 core refactor: preserve layout, resize/collapse/adsorb behavior. Wire it to the new APIs; do not redesign it.

## XiaoYu intelligence rule
XiaoYu is the product name for the user's configured **vendor model**. It is not a locally invented model and not a weaker planning engine. If the user configures GPT, Claude, Gemini, DeepSeek, etc., that provider/model performs the reasoning in the Agent Loop. AGMP adds tools, Skills, Memory, permissions and domain facts; it must not replace the vendor model's decisions with a hard-coded deployment state machine.

Model readiness requires a real generation call and a real tool-call probe. Never show Ready because a config object merely exists. Secrets are stored separately from public model profiles. API-key and subscription/coding-plan adapters share the same model-management abstraction; an adapter that is not implemented must report that explicitly rather than pretend to work.

## Agent-first rule
`Model -> Tool Call -> approval/capability -> execution -> Observation -> same Model` is the only task execution loop. Skills are decision guides, not workflows. Changing facts (versions, loader builds, mods, artifacts, hashes) come from live tools/APIs, never prompt memory. GameInstance/Profile is shared state and reuse data, not a process gate.

## Safety
All non-read Native side effects require TypeScript Approval followed by a short-lived, single-use Rust Capability Lease bound to run, tool, scope and exact payload. User can pause/cancel/take over. EULA and later legal/security-sensitive actions may use `confirm: always` and cannot be bypassed by full-auto mode.

## Platform contract
Web means browser-accessible Web. Desktop means native desktop application for that OS. Windows does not run a Linux product endpoint as a substitute; Linux uses the Linux endpoint/server, macOS uses macOS desktop, Windows uses Windows desktop.

## GitHub self-check workflow
Repository: https://github.com/yubboo/AI-Game-Manager-Panel.git
Before calling a version complete, inspect the latest GitHub commit and Actions yourself. Cross-platform Rust/Windows claims require GitHub Runner evidence when the current development environment cannot run Cargo or Windows.

## Fixed handoff workflow
Development package -> user extracts to `H:\一键部署\agmp-<version>` -> `AGMP-Sync.bat` -> destination `H:\一键部署\AI-Game-Manager-Panel` -> `AGMP-GitHub.bat` -> `1. 一键推送`. Packages must include SHA-256. Sync scripts must be Unicode-safe and must never corrupt the existing UI or user runtime data.
