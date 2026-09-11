# AGMP 0.2.2 Validation

## Scope

This release is the first XiaoYu Agent Runtime / Guided Autonomy pass. It changes capability exposure and context handling, so validation focuses on Tool Registry, approval boundaries, provider-native request compatibility, workspace file safety and Agent-loop regression.

## Required checks

- Go unit tests for `internal/xiaoyu/host`, `internal/ops/files`, `internal/app`.
- All non-Wails internal Go tests and `go vet` where the local toolchain can resolve dependencies.
- Project layout, module, XiaoYu core, model center and updater gates.
- Frontend `vue-tsc --noEmit` / build when pnpm dependencies are available.
- Rust `cargo test` when the Rust toolchain is installed.

## Key regressions covered

- DeepSeek thinking mode must not force incompatible `tool_choice`.
- DeepSeek reasoning content must still be replayed across a Tool turn.
- XiaoYu Tool Registry must expose `shell.exec` and workspace file mutation tools.
- `process.run` remains a manual compatibility alias rather than the model-facing Shell name.
- Workspace file mutation must reject symlink escape and workspace-root deletion.
- Old/large Observations must be compacted for model context without deleting durable Run receipts.
- Version must be consistent at `0.2.2` across Go, Vue, Electron, Wails, Rust workspace and installer metadata.
