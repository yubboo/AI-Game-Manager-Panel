# AGMP 0.2.1 Validation

## 修复目标

Web 一体服务开发在 `pnpm run typecheck` 阶段失败：`ModelManagementSection.vue` 使用了 `form.reasoningEffort`，但 `XiaoYuSaveModelRequest` 没有声明该字段。

## 根因

Go Core 的 `SaveModelRequest`、`ModelProfile`、`ModelConnectionRequest` 已经支持 `reasoningEffort`；前端共享类型 `frontend/src/shared/types/backend.ts` 的 `XiaoYuSaveModelRequest` 漏同步，形成前后端契约漂移。

## 0.2.1 修复

- `XiaoYuSaveModelRequest` 增加 `reasoningEffort: string`。
- XiaoYu Model Center Gate 增加该字段的契约检查，防止再次漏同步。
- 版本统一为 `0.2.1`：Go、Vue、Electron、Wails、Rust Workspace、Windows Installer、release manifest。
- Updater Gate 去掉当前版本的硬编码断言，继续保留 `0.1.100 -> 0.2.0` 的百小版本边界测试。
- 版本规则：`0.2.1 ... 0.2.100 -> 0.3.0`。

## 本轮已执行

- 18 个非 Release-Key `scripts/common/check-*.mjs`：PASS。
- 44 个非 Wails `internal` Go package：`go test` PASS。
- 44 个非 Wails `internal` Go package：`go vet` PASS。

## 当前环境未执行

- `pnpm run typecheck`
- `pnpm run build`
- Wails 完整编译

原因：执行环境无法访问 npm / Go 外部依赖源，且没有可复用的 pnpm node_modules。Windows 开发机已能安装依赖，因此请重新运行“开发模式 -> Web 一体服务开发”；预期原先 3 个 `reasoningEffort` TypeScript 错误将被消除。
