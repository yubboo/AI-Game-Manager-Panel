# Windows 自动更新架构

从 0.1.73 起，Windows Wails 正式版提供稳定通道更新检查。

## 更新来源

- Provider：GitHub Releases
- Repository：`yubboo/AI-Game-Manager-Panel`
- Endpoint：公开仓库 `releases/latest`
- Tag 建议：`v0.1.73`、`v0.1.74` ...
- Setup Asset：`AI-Game-Manager-Panel-{version}-Windows-x64-Setup.exe`
- 校验 Asset：同名 `.sha256`

公开仓库读取 latest release 不要求普通用户配置 GitHub Token。

## 客户端流程

1. 启动后进行低频检查，默认 6 小时内复用缓存结果。
2. 只接受 stable latest release，忽略 draft / prerelease。
3. 使用数字三段版本比较，支持 `0.1.100 -> 0.2.0`。
4. 用户主动点击“下载并安装更新”。
5. 下载 Setup 到 `runtime/cache/updates/<version>/`。
6. 校验文件大小与 SHA256；失败时禁止执行。
7. 启动 Inno Setup，并传入 `/AGMPUPDATE=1`。
8. Wails 主程序退出，由安装器继续完成交互式中文升级。
9. 固定 AppId + UsePreviousAppDir 复用原安装位置。
10. 升级覆盖 EXE 与静态配置，但不主动删除 `runtime`。

## 发布者流程

正式 Wails Release 构建结束后会输出：

- `AI-Game-Manager-Panel-X.Y.Z-Windows-x64-Setup.exe`
- `AI-Game-Manager-Panel-X.Y.Z-Windows-x64-Setup.exe.sha256`

创建 GitHub Release 时必须同时上传这两个文件，并填写 Release Notes。
