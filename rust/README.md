# AGMP 小鱼核心 Runtime

0.1.74 开始，Rust 是 AI Game Manager Panel 智能 Agent 的本地执行运行时，而不是复制 Go 游戏业务。

职责边界：

- Rust：Agent CLI、Tool Runtime、审批策略、本地命令执行、未来 PTY/沙箱/会话协议。
- Go：游戏服务器、Steam/DST、实例、授权、更新、Web API 与现有业务 Service。
- Vue：小鱼优先的人机交互；传统可视化页面作为手动兜底。

当前协议：`xiaoyu.v1`，本地 CLI 支持 `doctor`、`tools`、`tool`、`exec` 和 JSON-RPC stdio `rpc`。

> 0.1.74 是 XiaoYu Runtime Foundation。小鱼模型 Provider、多轮自动规划、流式 Tool Call、PTY 长会话与 Tauri 桌面壳在后续阶段继续接入。
