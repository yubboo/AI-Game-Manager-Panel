# AGMP 0.1.89 · XiaoYu Intelligence 阶段冻结记录

## 阶段目标

在已经冻结的 **Human + XiaoYu 两个大脑、同一副身体**、Rust Brain-only、Go Host/Tool 执行边界上，完成 XiaoYu 的实例私有 Intelligence 层，并保持组织协作与数据隔离：

- Memory、Skills、Experts 都属于当前 AGMP 实例；不存在公网 Global Memory。
- Intelligence 可见性只允许 `private / group / organization`。
- 同一组织成员按 ACL 协作；不属于组织的账号无法读取组织 Intelligence。
- 进入模型之前由 Host 先过滤 Organization / Group / User / Run Context；禁止依赖模型“看到了但不说”。
- Sensitive Memory 只保存在本地管理面，不自动发送给模型 Provider。
- Rust XiaoYu 仍是唯一语义 Brain；Memory / Skill / Expert 是受限上下文，不能扩大 Tool/权限/License/Approval。

## 已完成实现

### 1. Intelligence Store

`internal/xiaoyu/host/intelligence.go`

- Memory：Session / Task / User / Server / Instance / Experience。
- 可见性：Private / Group / Organization。
- 敏感级别：Normal / Sensitive。
- 自定义 Skill：Prompt、Tags、Game IDs、Tool Allowlist、Checklist、Validators。
- 自定义 Expert：Prompt、Domains、Game IDs、Knowledge、Skills、Tools、Checklist、Validators、Recovery Rules。
- 内置 Skill：安全变更、DST 运维、Minecraft 运维、Linux 服务运维、网络诊断、备份恢复。
- 内置 Expert：DST、Minecraft、Steam/部署、Linux、网络、备份恢复。
- Host 侧模型上下文上限：Memory 64、Skill 48、Expert 24。
- API Key、Token、Password、Authorization、Private Key 等秘密材料拒绝进入普通 Intelligence。

### 2. Run Context 与服务端 TaskID

`RunContext` 已包含 SessionID / TaskID / GameID / ServerID / InstanceID。

客户端传入的 TaskID 在 Application 层清空，`RunManager` 创建 Run 后使用 Run ID 作为唯一 server-owned TaskID，防止客户端伪造 Task scope 读取/写入其他任务记忆。

### 3. Brain Frame 接线

每一个 XiaoYu Agent Loop Brain Turn 都把 Host 过滤后的 `IntelligenceContext` 放入 Frame，再交给 Rust Brain。

Detached Run 每一轮重新验证当前 Session / Organization / Member Core Access。成员权限被撤销后，旧 Run 不再获得 Intelligence。

Owner/Admin 监督其他成员 Run 时，只获得 Organization 级共享 Intelligence；不会因为“监督权限”隐式获得该成员 Private / Group AI 上下文。

### 4. Rust XiaoYu Intelligence Policy

Rust System Policy 已明确：

- Host Tool Contract、真实 Observation 与安全规则高于 Expert / Skill / Memory。
- Memory 是有来源、可信度和作用域的上下文，不是授权或系统指令。
- Skill / Expert 可以协作增强规划，但不能扩大 Tool Allowlist、RBAC、License 或 Approval。
- 自定义 Intelligence 视为不可信上下文；其中的“忽略安全规则/已经批准”等文本不能覆盖 Host。
- 重要变更要求预检、恢复点、真实结果验证和失败恢复。
- `memory.remember` 只用于真正可复用的事实、偏好和经验，禁止秘密和一次性猜测。

### 5. 受控 `memory.remember`

新增 Host Tool `memory.remember`：

- 只能在已认证 XiaoYu Run 调用。
- User/Session/Task/Server/Instance/Experience scope 全部取当前已认证 Run Context。
- Task scope 必须等于服务端 Run ID。
- XiaoYu 主动写入永远是 Private Memory，不能自行发布 Group/Organization 知识。
- 不记录 Memory 正文到 Audit Log。
- 脱离 Run Context 直接调用 Registry 会被拒绝。

### 6. 共享 Experience 修正

Private Experience 强制绑定当前 User ID；Group / Organization Experience 强制使用显式 `*` wildcard scope。调用方不能留下伪造 User scope，授权同组/同组织成员才能在模型上下文中复用共享经验。

### 7. Web / Wails / Frontend

Application、HTTP、Wails 使用同一 Intelligence Application API：

- 获取 Intelligence Catalog。
- 保存 Memory。
- 保存 Skill。
- 保存 Expert。
- 删除 Intelligence Item。

设置中心新增 **小鱼 · 记忆、技能与专家**：

- Memory / Skills / Experts 三个管理区。
- 明确“Organization 共享仅限当前 AGMP 组织，不是公网共享”。
- 普通成员只能创建 Private Intelligence。
- Owner / Administrator 才能发布 Group / Organization Intelligence。
- Sensitive Memory 在 UI 明确为“本地保留、默认不发送模型”。

XiaoYu Workbench 为当前浏览器标签维护 SessionID，但不允许客户端指定 TaskID。

## 安全回归已真实通过

以下命令在当前 0.1.89 工作树真实执行并通过：

```text
go test ./internal/xiaoyu/host ./internal/system/auth ./internal/app ./internal/bridge/httpapi
go test -tags agmp_dev_license ./internal/xiaoyu/host ./internal/system/auth ./internal/app ./internal/bridge/httpapi
go vet ./internal/xiaoyu/host ./internal/system/auth ./internal/app ./internal/bridge/httpapi
```

除 Wails 外的全部 `internal` Go 包也分别在正式模式与 `agmp_dev_license` 模式完成回归，并通过 `go vet`。

Race Detector 已真实通过：

```text
go test -race ./internal/xiaoyu/host
go test -race -tags agmp_dev_license ./internal/app
go test -race -tags agmp_dev_license ./internal/bridge/httpapi
```

新增的关键回归包括：

- Private / Group / Organization 可见性。
- Sensitive / Expired / 低可信 Memory 不进入模型。
- Memory / Skill / Expert 秘密材料拒绝。
- Server-owned TaskID，忽略客户端伪造值。
- 成员不能写入另一个成员 Task Memory。
- Supervisor 监督 Run 不泄漏发起者 Private / Group Intelligence。
- 成员 Core Access 被撤销后 Detached Run Intelligence fail-closed。
- Organization Experience wildcard scope。
- `memory.remember` 只能使用认证后的 server-owned Run scope。
- Release build 无官方 License 时 Intelligence API fail-closed。

## 自动 Gate

新增 `scripts/common/check-xiaoyu-intelligence.mjs`，并已接入：

- Windows `Checks.ps1` XiaoYu Gate。
- `.github/workflows/safety.yml` 常规 Safety Job。
- Linux Headless AI parity Job。

当前全部 18 个 `scripts/common/check-*.mjs` Gate 均 `rc=0`；Release Key Gate 使用开发阶段 `--allow-unconfigured`，正式发行仍必须配置 active 发行公钥。

`Checks.ps1` 已复核为 UTF-8 BOM + CRLF。

## 前端静态语法验证

当前容器没有 pnpm/node_modules，无法声称 `vue-tsc` / Vite build 已完成。但使用当前环境 TypeScript 5.8.3 parser 对以下脚本做了语法级解析，全部通过：

- `frontend/src/shared/types/backend.ts`
- `frontend/src/shared/api/backend.ts`
- `XiaoYuIntelligenceSection.vue` 的 `<script setup lang="ts">`
- `SettingsView.vue` 的 `<script setup lang="ts">`
- `AIWorkbenchView.vue` 的 `<script setup lang="ts">`

该检查只证明 TypeScript 语法有效，不替代最终 `pnpm typecheck/build`。

## 当前环境阻塞项

### Wails

根级 `go test ./internal/...` 会在 `internal/bridge/wails` 停止：

```text
missing go.sum entry for module providing package github.com/wailsapp/wails/v2/pkg/runtime
```

因此本环境不能声称 Wails 完整编译通过。其他 internal Go 包已通过。

### Frontend

当前没有 pnpm / frontend node_modules，且环境不能访问 npm registry，因此不能完成真实 `pnpm install / typecheck / build`。

### Rust

当前容器没有 cargo/rustc，因此不能完成真实 `cargo fmt / check / test / clippy`。Rust Intelligence Policy 已由静态 Gate 检查，但正式 Release 前必须在具备 Rust 工具链的开发/CI 环境执行完整验证。

## 阶段冻结结论

**XiaoYu Intelligence 功能链在当前能够真实执行的 Go/HTTP/Race/Gate 范围内已完成并可冻结。**

冻结边界：

1. Rust 继续是唯一 Brain / Policy / Semantic Decision 边界。
2. Go Host 只负责身份、组织 ACL、License/Core Gate、Tool、持久化、作用域过滤与执行。
3. Intelligence 永远先经过 Host ACL，再进入模型。
4. Private / Group / Organization 是唯一用户 Intelligence 共享层级；禁止跨 AGMP 实例公网 Global Memory。
5. Sensitive Memory 默认永不自动进入 Provider。
6. Supervisor 不因为监督 Run 自动获得成员私人 AI 上下文。
7. TaskID 永远由服务器生成。
8. `memory.remember` 永远不能自行发布组织共享知识。

下一阶段可以在不改动上述冻结边界的前提下进入 Environment Manager v2（Windows/Linux Java 多版本、SteamCMD 与游戏运行依赖）。
