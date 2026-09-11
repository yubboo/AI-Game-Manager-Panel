# AGMP 0.2.0 Validation

## Candidate

- Baseline: `AI-Game-Manager-Panel-0.1.100-Codex-Turn-Approval-Fix-Source.zip`
- Candidate: `AI-Game-Manager-Panel-0.2.0-Native-Model-Harness-Source.zip`
- Scope: XiaoYu Native Model Harness / Provider Reasoning / Tool Replay / Vision / one-click server deployment intelligence.

## Passed

- All `scripts/common/check-*.mjs` gates passed except `check-release-key.mjs`, which is publisher-only and intentionally skipped in a source candidate environment.
- `go test ./internal/xiaoyu/... ./internal/app ./internal/bridge/httpapi ./internal/config ./internal/system/settings ./internal/deploy/updater` passed.
- `go vet` for the same packages passed.
- TypeScript parser check passed for `AIWorkbenchView.vue`, `ModelManagementSection.vue`, and `shared/types/backend.ts`.
- Native model regression tests cover OpenAI Responses encrypted reasoning replay, DeepSeek reasoning_content replay, native protocol selection, Vision serialization, and explicit rejection of unsupported image input.
- GitHub Safety Gate passed.

## Environment-blocked checks

- `go test ./internal/...` cannot complete in this sandbox because Go cannot download `github.com/wailsapp/wails/v2@v2.15.0` from `proxy.golang.org` (network/DNS blocked). This is not a source compile error in the tested packages.
- `corepack pnpm install` cannot download `pnpm@11.17.0` from npm registry because outbound DNS/network is blocked; therefore `pnpm build` / `vue-tsc` full project build cannot be honestly marked PASS here.
- `cargo` / `rustc` are not installed in this execution environment; Rust XiaoYu Core requires local Windows/CI compilation before release publishing.
- Release Key Gate is intentionally not run for a source candidate; formal publisher release still requires the private release key.

## Required local final verification

Run `AGMP.bat` menu 3 full project check, then Wails/Electron candidate build, and execute Rust/Frontend builds on the normal Windows development machine before menu 10 formal publication.
