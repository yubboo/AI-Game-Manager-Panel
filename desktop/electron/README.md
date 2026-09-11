# AI Game Manager Panel · Electron Desktop

Electron 是 AI Game Manager Panel 的 Windows 桌面适配器之一，与 Wails 共用同一 Go Core 和 Vue 工作台，不复制业务逻辑。

## Runtime

```text
Electron BrowserWindow
  -> preload 白名单 IPC
  -> Vue UI（127.0.0.1 loopback）
  -> HTTP Bridge
  -> AI-Game-Manager-Core.exe
  -> Application / Service / Game / Platform
```

Production 会在随机 `127.0.0.1` 端口启动打包的 `AI-Game-Manager-Core.exe`。Renderer 不直接获得 Node、进程或任意文件系统能力。

安全默认值：`nodeIntegration: false`、`contextIsolation: true`、`sandbox: true`，权限请求默认拒绝，外部 HTTP(S) 链接交给系统浏览器。

## Windows 菜单

运行仓库根目录 `AI-Game-Manager-Panel.bat`：

- `2 -> Electron`：Electron 开发模式；
- `5`：Electron Windows Release；
- `7`：预览已构建 Electron；
- `10 -> 2`：一键 Electron Windows Release。

最终文件只看 `build/release/windows/electron/`；`build/work/electron-*` 是中间工作区。

## Package Size

Electron 自带 Chromium + Node.js，因此明显大于 Wails。0.1.67 已启用 ASAR、maximum 压缩、语言包裁剪（zh-CN/en-US）以及 Go Core 去调试符号。若更看重小体积，优先发布 Wails 版本。
