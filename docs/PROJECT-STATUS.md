# AGMP Current Project Status

> Current development snapshot for humans and AI agents. Keep this file short and update it in-place; historical details belong in `PROJECT-HISTORY.md`.

## Current version

**0.2.17 — Windows ConPTY Runtime Convergence**

## Stable baseline

0.2.16 confirmed that Windows ConPTY now compiles on the real Windows GitHub Runner: `cargo fmt` and `cargo check --workspace --locked` both pass. Safety, Linux native PTY, Linux Headless and Windows Helper remain green. The remaining red lane is the Windows ConPTY runtime integration itself.

The 0.2.16 Runner exposed two concrete Windows runtime semantics: Rust canonical paths such as `\\?\D:\...` were handed to `cmd.exe` as the current directory and treated as UNC-style paths, causing CMD to fall back to `C:\Windows`; and `terminal/write` used LF for `appendNewline`, so the ConPTY shell did not reliably receive an Enter command.

## 0.2.17 changes

- Normalize Windows verbatim local-drive cwd only at the `CreateProcessW` boundary while keeping canonical paths for internal scope/security checks.
- Use backend-aware terminal newline semantics: Windows ConPTY uses CRLF; Linux PTY and fallback terminals use LF.
- Fix Terminal resize lock lifetime with lexical `MutexGuard` scope instead of dropping a shadowed `&mut TerminalProcess` reference.
- Compile the stdio `Pipe` process variant only on fallback platforms so Linux/Windows native builds do not carry dead code.
- Extend the Terminal/PTY source gate with Windows cwd normalization, CRLF and regression checks.
- Do not add new model-visible execution privileges in this release.

## Current migration boundary

Still in Go for compatibility:

- model-provider HTTP transport;
- top-level Agent Loop orchestration;
- current model-visible `shell.exec`;
- AGMP Domain Tool registry and game/product services.

Rust owns Tool Search, Brain policy primitives, Session Registry, Long-running Jobs, Persistent RPC Worker and Terminal Runtime. Linux native PTY is CI-proven. Windows ConPTY is compile-proven and is waiting for runtime integration/workspace tests to turn green.

## Next runtime milestones

1. Obtain a fully green `Windows Rust Runtime + ConPTY` lane.
2. Wire approved interactive Agent actions to the native Terminal Runtime.
3. Sandbox / capability leases / filesystem scope.
4. Apply Patch / generic filesystem mutation.
5. Reflection / experience pipeline.
6. Subagent / specialist dispatch.

## Verification status

Local packaging runs project Node gates and Go compatibility checks where available. If Cargo is installed on the Windows development machine, `AGMP-GitHub` additionally runs Rust fmt + workspace check before push. GitHub Actions remains authoritative for Windows ConPTY integration and cross-platform Rust tests.

## AI reading order

1. `AGENTS.md`
2. `docs/PROJECT-STATUS.md`
3. `docs/PROJECT-ARCHITECTURE.md`
4. `docs/architecture/LANGUAGE-OWNERSHIP.md`
5. `docs/development/PROJECT-RULES.md`
6. `docs/DEVELOPMENT-PLAN.md`
