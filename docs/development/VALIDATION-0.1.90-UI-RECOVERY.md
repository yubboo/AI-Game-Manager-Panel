# AGMP 0.1.90 UI Recovery + Functional Wiring 验证记录

## 修复基线

- 输入基线：`AI-Game-Manager-Panel-0.1.89-Developer-Helper-Fixed-Source.zip`
- 输出候选：`AI-Game-Manager-Panel-0.1.90-source.zip`
- 本轮范围：三栏信息架构、设置二级导航、小鱼中央工作区、刷新登录恢复、Intelligence / Runtime 空白、模型实际 Tool 调用兼容。

## 已确认根因

### 1. 刷新后强制重新登录

HTTP/Web/Electron 使用 HttpOnly `agmp_session` Cookie；浏览器 JavaScript 本来就不能读取该 Token。旧 `AuthGate` 却先以 `getSessionToken()` 是否为空决定是否展示登录页，因此刷新时会在调用 `currentUser()` 前错误跳回登录。

0.1.90 规则：只有 Wails Bridge 依赖显式 Bearer Token；HTTP/Electron 刷新时直接使用 Cookie 调用 `currentUser()` 恢复会话，真正收到 401 才回登录页。

### 2. “记忆、技能与专家”空白

空 Memory Catalog 在 Go 中可能把 `nil` slice 序列化成 `null`；前端异步加载后直接调用 `catalog.memories.length`，会触发渲染异常。

0.1.90 双层修复：Go API 永远返回数组；Vue 再把 `memories/skills/experts` 防御性归一化为 `[]`。

### 3. “运行环境与存储”空白

空 Runtime Registry 的 `runtimes` 曾可能返回 `null`，前端 `catalog?.runtimes.find(...)` 只保护了 `catalog` 本身，没有保护 `runtimes`，异步响应到达后可触发 `.find` 异常。

0.1.90 双层修复：Go Runtime Catalog 空集合返回 `[]`；Vue 对 `runtimes/javaMajors/warnings` 归一化，并使用 `runtimes?.find`。

### 4. 模型已配置但 XiaoYu 实际无法调用

实机 Observation 已显示 Provider 返回：`HTTP 400: Invalid 'tools[0].function.name'`。AGMP Host Tool 使用 `system.info` 等点号命名空间，而 DeepSeek / OpenAI-Compatible function name 只允许字母、数字、下划线和短横线。

0.1.90 在 **Provider 边界**建立稳定别名：真实 Host Tool 名不变，只在发给模型时转换为合法名称；模型返回 Tool Call 后再映射回 Host Registry 原名。Anthropic / Gemini 走同一别名规则。

连接测试也不再把 `/models` 成功当成“AI 可用”；当填写模型代码后，会继续发送一条带 XiaoYu Tool Schema 的最小生成请求。

## UI 冻结结果

- 左侧只承载一级导航；大屏默认约 `220–260px`，最大 `280px`。
- 中央只承载当前页面主内容；小鱼恢复为单一居中的对话 / Run / Composer 工作区。
- 右侧承载上下文与二级导航；大屏默认约 `250–300px`，最大 `320px`。
- 设置中心的 `模型管理 / 记忆、技能与专家 / 运行环境与存储 / ...` 已从中央左列迁移到真正的右侧栏。
- XiaoYu Runtime / Brain / 审批 / Tool / Harness / Trace 已迁移到真正的右侧上下文栏。
- 终端只进入 Bottom Panel，不再在中央或 XiaoYu 内部复制第二套侧栏/终端。
- 布局持久化键升级为 `agmp.workbench.layout.v2`，旧 0.1.89 超宽侧栏状态不会继续污染 0.1.90。

## 已执行静态验证

通过：

- 修改过的 9 个 TypeScript/Vue Script 块使用 TypeScript 5.8.3 `transpileModule` 语法检查。
- 7 个修改过的 Vue 外层 Template 进行 HTML parser 结构检查。
- `base.css`、`AIWorkbenchView.vue`、`XiaoYuRightContext.vue` 花括号数量一致。
- 修改过的 Go 文件全部 `gofmt`，`gofmt -d` 无差异。
- `frontend/package.json`、`desktop/electron/package.json`、`configs/release.json` JSON 解析通过。
- `check-updater.mjs`、`check-windows-helper.mjs` Node 语法检查通过。
- 新增模型 Tool Alias 单元测试；新增 `/models` 成功但实际 Tool Probe 失败不得误报成功的测试；补充空 Catalog 必须输出数组的断言。

## 当前环境未能完成的 Gate

### Go test

`GOPROXY=off go test ./internal/xiaoyu/host` 在依赖解析阶段停止：当前容器没有缓存 `github.com/wailsapp/wails/v2@v2.15.0`，且联网依赖下载不可用。因此本记录 **不声明 Go Test 已通过**。

### Frontend build

当前容器没有项目 `node_modules`；Corepack 获取 pnpm 时因 `registry.npmjs.org` DNS/网络不可用失败。因此本记录 **不声明 `pnpm build` 已通过**。

### Project Layout Gate

基线源码自身的 `scripts/common/check-project-layout.mjs` 仍引用当前压缩包中不存在的 `internal/xiaoyu/runtime/*`、`internal/platform/runtime/*`、`internal/games/dst/runtime/*` 等路径，因此 Gate 在本轮修改前的源码结构上即不满足。本轮没有用伪文件掩盖该问题；它应作为独立的基线/Checker 一致性任务处理。

## Windows 实机验收顺序

1. `pnpm install`
2. `pnpm --dir frontend run build`
3. `go test ./...`
4. `go vet ./...`
5. 通过开发入口启动 Web/Wails。
6. 登录一次后连续 F5 / Ctrl+R 3 次，确认会话保持。
7. 1920px、2560px 大屏分别检查三栏：中央明显为主，左右栏不再按 20% 放大；拖拽、收起、恢复正常。
8. 进入设置中心，确认设置分类只出现在外层右侧栏；中央不再有第二列导航。
9. 分别打开“记忆、技能与专家”“运行环境与存储”，在空数据情况下仍能完整渲染并可操作。
10. 保存 DeepSeek 为默认模型，先点“测试连接”，再回小鱼执行“检查本机开服环境”。不得再出现 `tools[0].function.name` Pattern HTTP 400。
11. 小鱼页中央只保留对话/任务主体；Runtime、审批、Tool、Harness、Trace 在外层右栏；完整终端从 Bottom Panel 打开。
