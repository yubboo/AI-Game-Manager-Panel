# AGMP 0.2.0 — XiaoYu Native Model Harness

0.2.0 upgrades XiaoYu from a lowest-common-denominator model transport into a provider-aware Harness.

- OpenAI official preset uses Responses API and preserves encrypted reasoning replay.
- DeepSeek uses a dedicated native protocol path and preserves `reasoning_content` / thinking settings.
- Claude preserves thinking/signature blocks across Tool turns.
- Gemini preserves provider thought/function parts, including `thoughtSignature` fields returned by the provider.
- Compatible gateways retain a separate OpenAI-compatible path and multi-step Tool replay.
- Model Center exposes adapter/capability metadata and explicit reasoning effort.
- XiaoYu Workbench supports direct image attachments for Vision-capable adapters; raw bytes are excluded from normal Run JSON serialization.
- Model inference uses bounded retry for transient 408/429/5xx failures; Tool side effects remain behind Host/RBAC/Approval.
- Built-in one-click game-server deployment intelligence adds Preflight, data protection, dependency preparation, install/update, configuration, start, verification, and recovery workflow guidance.
