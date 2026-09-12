# AGMP Electron Desktop — 0.4.x

Electron is the native desktop shell for the existing Vue three-pane workbench.
The shell starts the TypeScript AGMP Core and points it at the Rust Native Runtime.

Source ownership remains:

- Vue/TypeScript: UI and Agent/product logic
- Rust: native process/filesystem/network/security primitives
- Go/Wails: retired

The current Codex-style three-pane UI remains frozen during the 0.4 architecture reset.
