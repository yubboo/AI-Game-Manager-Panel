# AI Game Manager Panel GitHub 提交安全规范

> 自 AI Game Manager Panel 0.1.67 起生效；0.1.70 增加发行密钥环与 Release Key Gate。目标仓库：`yubboo/AI-Game-Manager-Panel`。仓库可以公开源码，但绝不能公开本机秘密、授权签发材料或用户运行数据。

## 可以上传

- `cmd/`、`internal/`、`frontend/`、`desktop/` 中的源码；
- `configs/` 中不含真实凭据的默认模板；
- `distribution/` 中的安装器源码、Docker/打包定义、第三方许可证文本；
- `docs/`、`scripts/`、`README.md`、`AGENTS.md`、`go.mod`、`wails.json`；
- **许可证公钥 / 公开发行密钥环**：`internal/system/license/vendor_public_keys.json`。公钥只是验签信任根，可以正常提交并编译进客户端源码。

## 永远禁止上传

1. **许可证发行私钥**：`agmp-release-private.key`、`agmp-license-private.key`、任何 `*.key/*.priv/*.seed/*.pem/*.p12/*.pfx/*.jks/*.keystore` 私钥容器。
2. **真实 CDK / 许可证证书**：实际签发的 BFLC/BFCDK、`activation.json`、LicenseID/ActivationID 批量发行文件。
3. **管理员与认证状态**：`accounts.json`、`bootstrap.lock`、下载的安全密钥文件、会话/凭据导出。
4. **设备身份**：`install.id`、`device.id`；MachineCode/InstallID 不应作为仓库样例数据长期公开。
5. **DST / Steam 真实机密**：`cluster_token.txt`、服务器 Token、真实 `cluster.ini` 密码、Steam 登录凭据。
6. **运行数据**：`runtime/*`（仅 `runtime/README.md` 除外）、旧版 `data/log/backups/instances/temp/exports/plugins/cache`。
7. **日志/崩溃转储**：真实日志可能包含目录、用户名、Token、IP、服务器名和其他隐私信息。
8. **构建产物与依赖缓存**：`build/`、`node_modules/`、`dist/`、EXE/ZIP/安装包。正式二进制应走 Release/CI，而不是直接混进源码历史。
9. **环境文件和云端凭据**：`.env*`、`credentials.json`、`secrets.json`、GitHub/API/云平台 Token。

## 为什么不能把离线 CDK 私钥放到 GitHub

AI Game Manager Panel 客户端只需要**公钥**验证许可证。真正能生成有效离线许可证的是发行私钥；一旦私钥进入公开 Git 历史，任何人都可以伪造官方许可证。即使之后删除文件，旧 Commit 仍可能保留副本，因此必须在第一次 Push 前阻断。

## 开发模式与正式授权分离

0.1.67 起，本地源码开发通过 Go Build Tag `agmp_dev_license` 使用“开发许可证模式”，允许开发调试全部功能，不需要真实 CDK，也不会生成一个可传播的万能 CDK。

- `AI-Game-Manager-Panel.bat -> 开发模式 -> Wails/Web/Electron` 会自动使用开发模式；
- 正式 Wails/Electron/Web Release **不会**加入该 Build Tag，仍按正式许可证逻辑工作；
- 公开源码的人当然可以自行修改或重新编译源码，因此 AI Game Manager Panel 的商业授权边界应以官方签名发行二进制/服务端能力为准，而不是把秘密硬编码在公开客户端源码里。


## 0.1.70 发行密钥轮换

第一次正式发行前运行 `scripts/tools/license/AGMP-License-Admin.bat`。工具把私钥生成在 `%LOCALAPPDATA%\AI-Game-Manager-Panel\ReleaseKeys`（或 `AGMP_RELEASE_KEY_DIR` 指定的仓库外路径），源码中只更新 `vendor_public_keys.json`。

公开公钥环允许同时保留 `active / retired / legacy` 公钥；旧公钥继续用于验证已经签发的历史 BFLC2。正式 Release 使用 `node scripts/common/check-release-key.mjs` 强制要求存在一个合法 active 公钥。

## AI 主动查看 GitHub 基线

GitHub 不只是最终 Push 目标，也是版本与 CI 的事实来源。AI/开发者在以下场景必须主动检查仓库 `yubboo/AI-Game-Manager-Panel` 的 `main`，不等待用户提醒：开始新版本、用户报告 Push 完成、修复 CI、准备下一版源码包。

检查顺序固定为：最新 commit SHA/消息 → 对应 Actions run → Job/Step 状态 → 失败日志。只有真实 Runner 输出能证明平台行为；本地源码推断不能替代 Windows/Linux CI 证据。

## 每次 Push 前

优先运行：

```text
AI-Game-Manager-Panel.bat -> 3. 项目检查
```

也可单独运行：

```text
node scripts/common/check-github-safety.mjs
```

检查器会验证 `.gitignore` 基线，并扫描即将可能进入 Git 的文件名以及典型私钥/访问令牌格式。发现风险时必须停止提交。

## 如果秘密已经误传

不要只做“删除文件再 Commit”。应立即：

- 停止继续使用已经公开的私钥/Token；
- 轮换相应密钥或凭据；
- 清理 Git 历史中的敏感对象；
- 对 License 发行私钥而言，需要更换发行密钥，并在后续正式版本中更新客户端公钥信任根。

## 仓库端二次门禁

`.github/workflows/safety.yml` 会在 Push/PR 后自动执行安全扫描、结构检查和 Go Test/Vet。本地门禁用于阻止“准备提交”的秘密，Actions 用于防止团队成员或其他入口绕过本地检查。


## 0.2.3 推送原则

除真实密钥/凭据、本机运行数据、用户服务器数据、依赖缓存和构建产物外，AGMP 项目内容默认进入 GitHub，包括源码、公开配置、CI、测试、Gate、文档、安装/发布脚本以及 `Cargo.lock` / `pnpm-lock.yaml` 等可复现构建锁文件。

根运行数据兼容规则必须使用 `/logs/`、`/instances/` 这类根目录锚点；禁止使用会误伤 `internal/ops/logs/` 或 `frontend/src/features/instances/` 的全局 `logs/` / `instances/` 规则。

`AGMP-GitHub.bat` 必须保持 ASCII + CRLF + 无 BOM；中文菜单由 `push-agmp.ps1`（UTF-8 BOM + CRLF）输出。

## Headless Web 生成资产边界

`cmd/aigame-manager-web/web/assets/` 是 `frontend/dist` 经 `scripts/common/sync-web-assets.mjs` 生成的 content-hash 构建资产，不是源码。仓库通过 `.gitignore` 阻止它们重新进入 Git；已有旧 hash 允许在一键推送时被清理。删除白名单必须严格限制在这个 assets 目录，`web/index.html` 与其余 `cmd/` 文件仍属于关键源码保护范围。
