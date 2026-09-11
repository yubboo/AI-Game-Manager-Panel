# AGMP 0.1.84 Validation Report

## Result

**Shared Runtime hardening: PASS in the available environment.**

The architecture remains frozen from 0.1.83; 0.1.84 validates the common process/terminal foundation and removes remaining duplicate process launch paths.

## Go tests and vet

42 internal Go packages were enumerated directly, excluding only `internal/bridge/wails` because this source snapshot still has no `go.sum` and the isolated environment cannot fetch the external Wails module.

- `go test`: PASS for all 42 available internal packages.
- `go vet`: PASS for all 42 available internal packages.
- `internal/platform/runtime`: PASS, including stream source separation, environment overlay, timeout/output bounds and Manager lifecycle tests.
- `internal/games/dst/runtime`: PASS after continued shared Runtime use.
- `internal/xiaoyu/runtime`: PASS after migrating one-shot process execution to `platformruntime.Run`.
- `internal/deploy/updater`: PASS after migrating installer launch to `StartDetached`.

## Build validation

- `go build ./cmd/aigame-manager-web`: PASS.
- `go build ./cmd/aigame-manager-license-admin`: PASS.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/platform/runtime`: PASS.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/games/dst/runtime`: PASS.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/xiaoyu/runtime`: PASS.
- `GOOS=windows GOARCH=amd64 go test -c ./internal/deploy/updater`: PASS.

## Architecture / safety gates

PASS:

- Distribution Boundary Gate
- GitHub Safety Gate
- Module Gate
- Product Architecture Gate
- Project Layout / Version Gate
- Release Key Gate in development mode (`--allow-unconfigured`)
- Updater Gate
- Windows Helper / Encoding / Frontend Import Gate
- Windows Installer Gate
- XiaoYu Core Gate

The Release Key Gate correctly warns that no active publisher key is configured; this is allowed for source development and must be configured before a formal Release build.

## Process ownership invariant

Repository scan found no active direct process creation outside `internal/platform/runtime`:

- no business `exec.Command(...)`;
- no business `exec.CommandContext(...)`;
- no business `StdinPipe/StdoutPipe/StderrPipe` ownership.

`exec.LookPath` may still be used only to discover executable paths.

The Module Gate now enforces this invariant.

## Environment-limited validation not claimed

The current execution environment does not provide:

- Cargo / Rust toolchain;
- pnpm;
- Wails CLI;
- `go.sum` for the Wails external dependency.

Therefore the following are **not claimed as passed** here:

- `cargo check/test --workspace`;
- Vue production build;
- Wails full desktop build;
- Electron full desktop build;
- Windows installer real execution;
- real Windows PTY/ConPTY behavior.

These remain Windows/local CI acceptance items. No checksum or dependency file was fabricated.
