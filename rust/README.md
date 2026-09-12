# AGMP XiaoYu Rust Runtime

0.2.9 起，Rust 的长期职责从“Brain-only”升级为 **XiaoYu Agent Runtime / Native Execution / Security Boundary**。

Rust 负责 XiaoYu 的通用 Agent 能力，Go 继续负责 AGMP 游戏/产品 Domain Service。详细语言职责见 `docs/architecture/LANGUAGE-OWNERSHIP.md`。

## 当前 crate

```text
rust/crates/xiaoyu-core       # Agent Runtime 主实现
rust/crates/xiaoyu-protocol   # xiaoyu.v1 协议
```

当前已经包含：

- provider-neutral Brain Policy / Decision Grammar；
- Guided Autonomy；
- Session Registry（create/get/list/close，0.2.10）；
- Approval hint；
- Tool Search / Capability Discovery（0.2.9）；
- Long-running Job Runtime（start/status/list/output/cancel，0.2.10）；
- bounded output / cancellation / Runtime Root cwd boundary；
- JSON-RPC stdio Runtime。

后续 Rust-first 能力：

- Agent Loop / Goal State；
- Context / Thread；
- PTY；
- Generic Shell / File / Process；
- Apply Patch；
- Sandbox / Capability Lease；
- Reflection / Experience；
- Subagent。

## Domain 边界

Rust 不复制：

- Steam / SteamCMD 业务；
- DST / Minecraft 业务规则；
- Instance / Backup / License / Updater 业务。

这些仍由 Go Domain Service 提供结构化 Tool。

## 安全

Rust Native Runtime 不等于无限权限。`jobs/start` 只是 Host 授权后的内部原语，不能直接等同于模型执行权。任何真实执行都必须保持：身份/RBAC、三种审批模式、Capability Scope、Sandbox、路径/参数限制、秘密保护、审计和执行后验证。

普通用户不需要安装 Rust/Cargo/MSVC。发行构建机/CI 预编译 XiaoYu Runtime，并把它作为 AGMP 内部组件随完整产品发布。

## 0.2.11 Persistent Worker

AGMP Go Host now keeps one supervised `xiaoyu rpc` process alive. Tool Search, Brain policy, Session Registry and Long-running Jobs share this process lifetime. The Worker remains an embedded AGMP component; Host RBAC/approval stays authoritative.
