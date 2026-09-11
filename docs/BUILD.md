# 构建与发布

AI Game Manager Panel 0.1.67 起采用明确的两区构建模型：`build/work` 为中间文件，`build/release` 为最终文件。完整目录说明见 `BUILD-DIRECTORIES.md`。

## Windows

运行：

```text
AI-Game-Manager-Panel.bat
```

常用菜单：

- `3` 项目检查；
- `4` Wails Windows Release（推荐小体积）；
- `5` Electron Windows Release；
- `10 -> 1/2/3` 一键 Wails / Electron / 全部发布。

Wails 最终产物：`build/release/windows/wails/`。  
Electron 最终产物：`build/release/windows/electron/`。

## Release 文件锁保护

Windows 上旧 Portable/Setup 可能正被运行、浏览器、杀毒软件或索引服务持有。0.1.67 使用逐文件 `Publish-AGMPArtifact`：先重试覆盖，仍被占用时使用带时间戳的新文件名发布，从而避免“构建已经成功，但最后晋升 Release 失败”。

## Electron 体积优化

- `asar: true`；
- `compression: maximum`；
- 只保留 `zh-CN` 和 `en-US` Electron Runtime 语言包；
- Go Core 使用 `-trimpath -ldflags "-s -w"`；
- Electron 包只包含 Renderer `dist`、`package.json`、共享 Go Core 与默认 JSON 配置。

Electron 本身包含 Chromium/Node.js，体积仍会显著大于 Wails。小体积优先使用 Wails。
