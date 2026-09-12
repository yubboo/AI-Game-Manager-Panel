# AGMP 命名与路径规范

> 本规范是 AI-Game-Manager-Panel 的长期开发约束。人类开发者与 AI 开发代理均应遵守。
> 详细语义优先通过目录、package、类型、函数和测试用例表达，不要把所有上下文塞进文件名。

## 1. 总原则

1. 名称必须**规范、简洁、稳定、可读**。
2. 禁止无意义缩写，也禁止把完整职责描述堆进一个超长文件名。
3. **目录本身就是命名空间**：文件名不要重复父目录已经表达的语义。
4. 优先使用领域名 + 职责名，例如 `runtime/service.go`、`models/store.go`。
5. 同一领域保持统一命名，不在不同目录混用同义词。
6. 文件/目录重命名必须同时更新引用、测试、文档与 Gate。

## 2. 长度规则

### 文件名

- 推荐：不超过 **32 个字符**。
- 一般上限：**40 个字符**。
- 测试文件一般上限：**48 个字符**。
- 超过上限时，应优先通过目录拆分或删除重复语义缩短。

示例：

不推荐：

```text
server_xiaoyu_models_release_test.go
```

在 `internal/bridge/httpapi/` 下推荐：

```text
models_release_test.go
```

不推荐：

```text
environment_runtime_manager_windows_release_integration_test.go
```

推荐：

```text
runtime_windows_test.go
runtime_release_test.go
```

### 完整相对路径

以仓库根目录为起点：

- 推荐：不超过 **160 个字符**。
- 超过 **180 个字符**：CI/Gate 警告。
- 超过 **220 个字符**：CI/Gate 必须失败。

这样为 Windows、ZIP、Installer、临时构建目录和 CI 工作目录预留空间。

## 3. 目录命名

- 全部使用小写英文。
- Go/Rust/前端源码目录使用简短领域名。
- 优先：

```text
internal/xiaoyu/host/
internal/ops/logs/
frontend/src/features/logs/
rust/crates/xiaoyu-core/
```

- 避免：

```text
internal/xiaoyu/model-provider-native-runtime-management-system/
```

如果职责太多，应拆成多个目录，而不是加长目录名。

## 4. Go 文件命名

使用 `snake_case.go`。

推荐：

```text
service.go
model.go
store.go
router.go
runtime.go
approval.go
models_http.go
models_store.go
```

测试：

```text
service_test.go
models_test.go
runtime_windows_test.go
```

不要重复 package / 父目录名称：

```text
internal/ops/logs/logs_service.go
```

应写为：

```text
internal/ops/logs/service.go
```

## 5. Rust 文件命名

使用 Rust 社区常规 `snake_case.rs`。

推荐：

```text
agent_loop.rs
tool_router.rs
context.rs
approval.rs
sandbox.rs
```

单文件职责过大时拆 module，不通过继续加长文件名解决。

## 6. Vue / TypeScript 命名

Vue 组件使用 PascalCase：

```text
LogsView.vue
InstancesView.vue
ModelCenter.vue
ApprovalMenu.vue
```

Composable：

```text
useLogHub.ts
useRuntime.ts
useApproval.ts
```

普通 TS 模块：

```text
backend.ts
models.ts
runtime.ts
```

避免：

```text
XiaoYuNativeProviderModelManagementSettingsSection.vue
```

应拆成目录和子组件。

## 7. 脚本命名

PowerShell：

```text
Push-Agmp.ps1
Test-Project.ps1
Build-Windows.ps1
```

但已有稳定入口可保持兼容，例如：

```text
push-agmp.ps1
```

BAT/CMD 只做薄入口，不承载大量逻辑：

```text
AGMP-GitHub.bat
AI-Game-Manager-Panel.bat
```

复杂逻辑放入 PowerShell / Go / Node 脚本。

## 8. 文档命名

长期规范文档使用稳定名称：

```text
README.md
AGENTS.md
docs/PROJECT-STATUS.md
docs/PROJECT-HISTORY.md
docs/PROJECT-ARCHITECTURE.md
docs/DEVELOPMENT-PLAN.md
docs/NAMING-CONVENTIONS.md
```

禁止继续按版本创建大量：

```text
VALIDATION-0.2.4.md
COMPLETION-0.2.4.md
0.2.4-release-notes.md
```

版本历史统一追加至：

```text
docs/PROJECT-HISTORY.md
```

## 9. 版本包命名

源码交付统一：

```text
agmp-0.2.4.zip
agmp-0.2.8.zip
```

禁止长描述包名：

```text
AI-Game-Manager-Panel-0.2.4-XiaoYu-Agent-Runtime-Full-Source-Candidate.zip
```

源码交付不仅固定文件名，也固定用户流程：

```text
agmp-<version>.zip + SHA-256
        ↓
H:\一键部署\agmp-<version>
        ↓
AGMP-Sync.bat
        ↓
H:\一键部署\AI-Game-Manager-Panel
        ↓
AGMP-GitHub.bat
        ↓
1. 一键推送
```

版本 ZIP 必须是完整源码树；禁止把 `patch`、少量脚本或长描述归档作为默认正式开发版交付。

## 10. AI 开发代理规则

所有 AI 在新增文件前必须先检查：

1. 父目录是否已经表达该语义；
2. 是否可以使用现有 `service.go / model.go / store.go / *_test.go`；
3. 文件名是否超过推荐长度；
4. 完整相对路径是否过长；
5. 是否为了“描述更完整”而重复命名；
6. 是否应通过拆目录/拆职责解决，而不是继续加长名字；
7. 开始新版本或准备交付前，是否已主动查看 GitHub `main` 最新 commit 与对应 Actions 结果。

AI 不得仅为了“看起来详细”创建超长文件名。

## 11. CI / Gate 要求

项目应维护命名 Gate：

- 扫描仓库文件名长度；
- 扫描完整相对路径长度；
- 超过 180 字符产生警告；
- 超过 220 字符失败；
- 对明显重复父目录语义的名称给出提示；
- 不对第三方生成目录、`.git/`、`node_modules/`、`target/`、构建输出执行该规则。
- 命名 Gate 只约束项目维护的源码/公开配置/文档；`runtime/*` 本机运行状态与外部环境资产不参与命名扫描，`runtime/README.md` 仍参与检查。
- 外部工具链/运行时自带文件（例如 JDK/JRE DLL）必须保留上游原名，禁止为了通过 AGMP 命名规则擅自重命名。

## 12. 重命名原则

重命名属于维护行为，不应为了“追求短”频繁制造 Git 历史噪声。

仅在以下情况重命名：

- 名称明显过长；
- 重复父目录语义；
- 术语不一致；
- 已造成 Windows/ZIP/Installer/IDE 兼容问题；
- 文件职责已发生变化。

重命名后必须确保：

```text
Go/Rust/TS 编译
测试
路由/Import
CI
文档引用
```

全部同步通过。

## 13. Release artifact names

Public build artifacts should also remain short and machine-friendly. Prefer:

```text
agmp-0.2.8-win-x64-setup.exe
agmp-0.2.8-win-x64-portable.zip
agmp-0.2.8-linux-amd64.tar.gz
```

Do not embed long marketing descriptions in filenames. Put descriptive text in Release Notes / GitHub Release metadata instead.
