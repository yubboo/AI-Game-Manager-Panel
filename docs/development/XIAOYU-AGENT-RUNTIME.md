# XiaoYu Agent Runtime

## Product principle

XiaoYu is not a fixed menu of AI features. The configured model is XiaoYu's current general reasoning brain; AGMP supplies persistent identity, professional guidance, memory, tools, execution environment and user-controlled authorization.

```text
XiaoYu
= Model Intelligence
+ Persistent Identity
+ General Capabilities
+ Expert / Skill Guidance
+ Memory / Experience
+ Agent Loop
+ Approval / Sandbox
```

The design goal is **model-intelligence preservation**: connecting a strong provider model must not reduce it to a small list of canned domain actions.

## Guided autonomy

Expert, Skill, Memory, Reflection and Experience are guidance layers. They should be consulted first because they are AGMP-tuned, but they are not a security allowlist and they do not define the outer capability boundary.

Execution preference:

```text
Expert / Skill / Memory
        ↓
Domain Tool (preferred)
        ↓ if insufficient
General Tool
        ↓ if insufficient
Controlled shell.exec
        ↓
Host Approval / RBAC / Sandbox
        ↓
Execute → Observe → Verify → Recover → Summarize
```

A specialized Tool improves reliability, auditability and recovery; its absence must not automatically mean XiaoYu is incapable when general capabilities can legally reach the same goal.

## Capability is not authority

XiaoYu may understand and plan broad actions. The existing `ask`, `risk` and `full` approval modes decide whether a concrete side effect is authorized. RBAC, sensitive-operation step-up, workspace/path checks and Host-side validation remain mandatory.

Safety therefore lives in the execution pipeline, not by making the model pretend a capability does not exist.

## What 0.2.2 implements

- model-facing Tool guidance tiers (`domain`, `general`, `control`, `fallback`);
- workspace file mutation tools with root and symlink containment;
- controlled model-facing `shell.exec` fallback, with `process.run` kept as compatibility alias;
- Runtime Manager resolve/default/remove tools, closing the Java/SteamCMD management loop;
- longer autonomous budgets and explicit verify/recover behavior;
- relevant-memory ranking and bounded Observation context projection;
- provider capability metadata and DeepSeek thinking/tool-choice compatibility handling.

## Codex ideas adopted

AGMP does not copy Codex product code. It adopts the architectural lessons visible in the open-source runtime:

- a model-facing Tool Router rather than feature-specific chat branches;
- general execution primitives alongside specialized operations;
- execution authority enforced through approval and sandbox policy;
- observations/results fed back into a continued Agent loop;
- context/thread management treated as first-class runtime behavior;
- future room for jobs, tool search and subagents.

## DeepSeek Harness ideas adopted

DeepSeek Harness demonstrates capability seams where model adapters, tools, sessions, agent loop, sandbox and approval are replaceable components. AGMP keeps its own Go/Rust boundaries but follows the same separation:

- provider model adapter is not XiaoYu identity;
- Tool schema/presentation is separate from trusted Host execution;
- durable receipts are separate from the bounded model-visible projection;
- policies wrap execution rather than being embedded in every domain feature;
- new capabilities should be mountable without rewriting the core loop.

## Next runtime stages

The following are intentionally not claimed as complete in 0.2.2:

1. PTY + long-running Job handles (`start/status/stdin/wait/stop`).
2. Multiple/parallel provider Tool calls with safe scheduling barriers.
3. Deferred Tool Search so hundreds of future tools do not consume every prompt.
4. Post-run reflection/experience distillation with validation before long-term memory.
5. Provider-native hosted capabilities (web/computer/search) where an API exposes them.
6. Subagent runtime and, later, XiaoYu Studio/Agent Teams.
7. AGMP Agent Bench for task success, false-complete rate, intervention rate and recovery success.

## 0.2.8 Agent Bench foundation

Static Gates prove that contracts and capabilities exist; they do not prove that an autonomous run behaves well. The first deterministic Agent Bench therefore measures observable runtime behavior:

- **fallback recovery**: a failed Domain Tool must not automatically end the task when a controlled general fallback can continue;
- **mutation verification**: a successful state-changing Tool is not enough to declare success when a read/status Tool exists for the same domain;
- **approval resume**: an approved action must resume the exact suspended Tool call and approval ID instead of silently replanning a different mutation.

The benchmark entry point is `go test ./internal/xiaoyu/host -run '^TestAgentBench' -count=1 -v`. New Agent Runtime behavior should add a repeatable benchmark scenario when possible. The long-term metrics remain task success rate, false-complete rate, recovery success and unnecessary-human-intervention rate; Tool count, prompt length and agent-role count are not intelligence metrics.
