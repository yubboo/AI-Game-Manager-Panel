# AGMP 0.1.88 完成度说明

## 总结

**架构方向：正确且已进入长期稳定阶段。底层：可冻结。XiaoYu Harness/模型中心：已有真实第一版，但智能能力仍在成长。产品总体功能：仍处于早中期。**

0.1.88 最大变化不是增加一个 AI 页面，而是把产品控制模型真正改造成：

`Human Brain + XiaoYu Brain -> 同一 AGMP Host -> 同一 Domain -> 同一 Shared Runtime`

人工与 XiaoYu 可以选择不同的控制方式，但不能拥有两套相互竞争的业务实现。

## 已完成的关键能力

### XiaoYu 大脑与 Harness

- Rust `xiaoyu-core` 保持 Brain-only；真实 OS/文件/游戏执行仍全部回到 Go Host/Domain。
- `xiaoyu.plugin.v1` Plugin Kernel、Capability Discovery、可逆 Tool 注册。
- Agent Loop / RunManager、Step/Tool/Failure Budget、Doom-loop 防护。
- Observation、等待审批、等待用户、取消、Pause/Takeover/Resume。
- Server-owned Run：浏览器断开不会取消 XiaoYu。
- 有界 Sequence Trace/Event Stream、断线续接基础、慢消费者保护。
- 多用户 Run Initiator 隔离与 Owner/Admin 全局监督。

### XiaoYu 大脑配置

- 系统设置 -> 模型管理成为唯一用户级大脑配置中心。
- 支持 OpenAI、DeepSeek、Claude、Gemini、OpenRouter、国内兼容 Provider、本地 Ollama/LM Studio、自定义 OpenAI-Compatible/中转站。
- 默认模型选择、模型发现、连接测试、上下文/输出 Token/思考模式/额外 JSON。
- API Key 与普通 Profile 分离；Windows DPAPI / 非 Windows 本地加密 Vault；前端不读取完整 Key。
- 没有可用默认模型或 Rust Brain 时 XiaoYu Fail Closed，不使用 Shell 冒充 AI。

### 手脚与底层

- Shared Runtime 仍是唯一 Process/stdin/stdout/stderr 实现。
- DST 已复用 Shared Runtime。
- 首批结构化 Domain Tools 已覆盖系统、文件、游戏发现、Steam/环境状态、DST Cluster/命令/日志。
- `process.run` 仍是人工/管理员兜底，默认不作为 XiaoYu 日常手脚。
- DST XiaoYu Tool 与人工启动入口开始统一复用相同 Domain Action。

### 多端

- Windows Wails / Windows Electron / Web / Linux Server Web / Docker Web 都被定义为完整 AGMP 产品目标。
- Linux Server、Linux 发行链、Docker 强制携带预编译 XiaoYu Runtime，缺少 XiaoYu 禁止出包。
- Docker Rust builder 修正为目标平台构建，避免多架构镜像混入错误架构 XiaoYu。
- Web API 已具备模型中心、Harness、Tools、Capabilities、Run Control、Approval、Trace/Event Stream。
- CI 已加入真实 Linux Headless Web -> XiaoYu Runtime HTTP 集成测试。

### 插件安全

- DSH 兼容层仍通过 Adapter，不让 AGMP 核心依赖 DSH 内部版本。
- DSH Tool JS 在独立 Node 子进程运行。
- 默认 Node Permission：只读插件自身目录，不开放文件写入和 child_process。
- 不继承 AGMP API Key/Token 等宿主秘密环境。
- 插件 root/entry/patch 使用真实 symlink 路径边界检查。
- 仍需显式 Owner 信任后才能挂载外部 DSH 插件。

## 模块状态

`configs/modules.json` 当前共 33 个模块：

- `implemented`：1
- `active`：1
- `partial`：7
- `skeleton`：24

这说明**架构成熟度仍明显高于产品功能成熟度**。0.1.88 不应被理解为“产品已经接近完成”。

## 仍需继续开发

优先级最高：

- XiaoYu Memory / 长期 Context / 任务上下文压缩；
- 更成熟的 Planner/Recovery/结果语义验证；
- Capability Router，避免未来数百 Tool Schema 每轮全部进入模型上下文；
- Backup、Instance、Minecraft、Steam Update、Network 等更多结构化 Domain Tools；
- Minecraft Adapter 与更多游戏；
- Run 持久化与 AGMP 重启后的恢复；
- 插件更强 OS 级沙箱及 DSH 更完整兼容。

底层增强但不改架构：PTY/ConPTY、Windows Job Object、外部进程 Reattach、长期压力/故障注入测试。

## 当前最重要的发行阻塞项

- 补正式 `go.sum`；
- 在线 CI 跑通 Rust + Vue + Headless XiaoYu 作业；
- Windows Wails/Electron/Installer 实机完整构建；
- Linux/Docker amd64 + arm64 实际构建；
- 至少一次真实模型 Provider + Linux Web + XiaoYu Autonomous Run 端到端验收。
