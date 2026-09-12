# Project Rules

1. Ship working increments, not architecture-only proposals.
2. No Go core and no Wails dependency in 0.4+.
3. Do not redesign the frozen Codex three-column UI while core migration is underway.
4. Never fabricate model intelligence. The selected vendor model owns planning and recovery decisions.
5. Facts that can change must be queried. Hashes and compatibility are deterministic Tool work.
6. Side effects use Approval + Rust Capability. No direct Node filesystem/process bypass for Agent actions.
7. GitHub Actions is the cross-platform source of truth when local Cargo/Windows is unavailable.
8. Every handoff package includes sync helper, GitHub helper and SHA-256 and follows the fixed `H:\一键部署` workflow.

## Version / handoff naming

- An unpushed, unfrozen version keeps the same version number while fixes are made. Rebuild and replace `agmp-X.Y.Z.zip`; do not invent a suffix.
- Once `X.Y.Z` is pushed/frozen, any corrective source change becomes the next patch version.
- The only source handoff artifact names are `agmp-X.Y.Z.zip` and `agmp-X.Y.Z.sha256.txt`.
- Forbidden handoff suffixes include `hotfix`, `final`, `fixed`, `new`, `v2`, timestamps, and date tags.
- The SHA-256 companion must be regenerated every time the package bytes change.

### Windows helper UX rule

- `AGMP-GitHub.bat` must keep the console open after both success and failure so the user can read the final result.
- The recommended `1. 一键推送` action must be visually highlighted in the PowerShell menu.
- Do not rely on the user reopening a closed console to discover push or Gate results.
