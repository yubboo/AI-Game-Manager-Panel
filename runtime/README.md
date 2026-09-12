# AGMP Runtime

Local runtime data is not source code.

- `runtime/data/` — auth, model profiles/secrets, sessions and GameInstance state.
- `runtime/workspace/` — game server files visible to approved Agent tools.
- `runtime/native/` — Rust-managed runtimes such as Temurin Java.

These directories are intentionally separated so ordinary workspace tools cannot read model secrets or modify managed runtimes.
