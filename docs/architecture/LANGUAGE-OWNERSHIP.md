# Language Ownership — AGMP 0.4+

| Layer | Owner | Rule |
|---|---|---|
| Agent Loop / Model Providers / Tools registry / Sessions / MCP / Plugins / HTTP/SSE | TypeScript | Primary product logic; rapid Agent evolution |
| Web UI / shared frontend contracts | TypeScript + Vue | Existing three-column UI stays visually frozen during core migration |
| Process / PTY / filesystem / sandbox / capability / Java / verified downloads / MC protocol | Rust | Native safety and deterministic system work |
| Go | None | Removed from 0.4 core |

The configured vendor model is the reasoning brain. TypeScript is the harness; Rust is the body/safety kernel.
