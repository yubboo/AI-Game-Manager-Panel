# AGMP 许可证正式发行验收（0.1.72+）

## 一键自动验收

Windows 运行：

```text
scripts\tools\license\AGMP-License-Admin.bat
→ 6. 许可证完整验收
```

自动检查：GitHub Safety、项目骨架、Active Release Key、Standard/Pro Feature Matrix、错误机器码/BFID、篡改签名、IssuerKeyID、防重启丢授权、Development Build Tag 与 Windows License Cross Build。

## 正式 Release 实机验收

1. 解压新源码。正式 Release 若 `vendor_public_keys.json` 尚无 Active Key，会先尝试从 `%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys` 自动同步公开密钥。
2. 构建 Wails 或 Electron 正式 Release。
3. 启动正式 Release，确认 BFM/BFID 稳定。
4. 导入与该 BFM/BFID 匹配的 BFLC2。
5. Pro 证书必须允许：`server.basic`、`server.multi`、`backup.advanced`、`remote.agent`、`web.remote`、`automation`、`ai.workbench`、`plugin.extensions`。
6. Standard 证书默认只允许：`server.basic`。
7. 未授权、过期、解绑、错误机器码、错误 BFID、篡改证书必须拒绝高级功能。
8. 完全退出桌面端与 Core 后重新启动，LicenseID、ActivationID、BFM、BFID、Edition 与 Feature Entitlements 必须保持一致。

## 安全边界

Vue 路由上的 License Gate 只负责界面提示，不是安全边界。任何未来新增的高级 Go Core / HTTP / Wails 业务操作，都必须在执行前调用：

```go
Application.RequireLicenseFeature("feature.id")
```

或底层：

```go
license.Service.RequireFeature("feature.id")
```

正式发行私钥永远不得进入源码、GitHub、安装包或公开日志。
