# Build 目录说明（0.1.67+）

## 只需要记住两层

- `build/release/`：**最终发布产物**。这里的文件才适合运行、发给用户或上传 GitHub Release。
- `build/work/`：**构建工作区**。Electron Builder、Wails Release Stage、Go sidecar、检查程序等中间文件都放这里，可安全清理。

## Windows Wails

```text
build/release/windows/wails/
├─ portable/AI-Game-Manager-Panel-Windows-x64-Portable.zip
└─ installer/AI-Game-Manager-Panel-<version>-Windows-x64-Setup.exe
```

对应工作区：`build/work/windows-wails/`。Wails Desktop、Web 支撑二进制和 AI Core Runtime 原始构建物都保留在 work 中；它们是完整产品打包所需内部组件，不单独作为普通用户 Release。

## Windows Electron

```text
build/release/windows/electron/
├─ installer/AI-Game-Manager-Panel-<version>-x64-Setup.exe
└─ portable/AI-Game-Manager-Panel-<version>-x64-Portable.exe
```

对应工作区：

- `build/work/electron-core/`：Electron 完整产品的内部 Go HTTP Core / AI Core Runtime 构建物；
- `build/work/electron-builder/`：electron-builder 临时输出、unpacked 文件、blockmap 等。

## build/bin 为什么可能为空

`build/bin/` 是 Wails CLI 自带的默认临时输出位置，不再是 AI Game Manager Panel 的正式发布目录。Electron 发布不会往这里写文件；Wails 构建后脚本也会把 EXE转移并清理空目录。

因此看到 `build/bin` 为空并不代表构建失败。判断成功与否请看 `build/release/` 和控制台最终状态。

## 文件被占用

0.1.67 起发布阶段不再整体移动旧 Release 目录。每个最终文件单独发布：

1. 先复制为临时 incoming 文件；
2. 尝试替换目标，最多重试；
3. 如果旧目标仍被运行中的 EXE、Defender 或其他进程占用，则把新文件保存为 `-Rebuild-时间戳`；
4. 其他产物继续发布，不让整个 Release 因一个旧文件锁失败。
