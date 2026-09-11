# Windows 正式安装器

从 0.1.72 起，Windows 官方桌面发行版采用：

```text
Wails + Vue 3 + Go Core + Inno Setup
```

Electron 保留为兼容发行，不再作为 Windows 主发行方案。

## 正式安装器

最终文件：

```text
build/release/windows/wails/installer/AI-Game-Manager-Panel-<version>-Windows-x64-Setup.exe
```

构建临时文件只进入：

```text
build/work/windows-wails/
```

## 用户安装流程

正式 Setup.exe 必须经过完整 Windows 安装向导：

1. 欢迎页。
2. **软件许可及服务协议**。用户必须选择“我接受本协议”才能点击下一步；选择“我不同意本协议”不能继续。
3. 选择安装目录。
4. 选择开始菜单文件夹。
5. 选择是否创建桌面快捷方式。
6. 安装前确认页。
7. 安装程序文件与默认配置。
8. 完成页，可选择启动 AI游戏管理器面板。

## 中文策略

安装器不再依赖 Inno Setup 安装目录中的 `Languages/ChineseSimplified.isl`。

脚本始终以 Inno 自带 `Default.isl` 为基础，再由 `AIGameManagerPanel.iss` 覆盖用户可见的简体中文向导文本。因此即使开发机器没有额外中文语言包，也不会回退成英文安装器。

## 协议文件

```text
distribution/installer/windows/EULA-zh-CN.txt
```

协议同时嵌入安装器，并在安装后保存到：

```text
<安装目录>/licenses/AGMP-EULA-zh-CN.txt
```

## 升级与卸载

从 0.1.73 起，安装器补齐中文“文件夹已存在/文件夹不存在”提示，并与客户端自动更新流程联动。

- 使用固定 AppId，因此后续同产品版本可识别已有安装。
- 升级时复用用户此前安装目录、开始菜单和附加任务选择。
- 如果选择的是同一 AGMP 已安装目录，`DirExistsWarning=auto` 避免正常升级流程重复弹出无意义目录警告；手工选择其他已存在目录时，提示文本为中文。
- 从应用内触发更新时，安装器使用 `/AGMPUPDATE=1` 标识进入升级流程，并在欢迎页明确说明 runtime 数据保留。
- 安装更新时允许 Windows Restart Manager 关闭占用待更新文件的旧程序。
- 卸载前使用中文确认提示。
- 为降低误删风险，用户运行产生的 `runtime/` 数据默认保留。

## 当前安装范围

0.1.73 正式安装器仍采用**当前用户安装**：

```text
%LOCALAPPDATA%\Programs\AI-Game-Manager-Panel
```

这样应用能够安全写入当前版本的 `runtime/` 目录，不需要管理员权限。未来如果迁移到 `Program Files`，必须先将运行数据根目录独立迁移到 `%LOCALAPPDATA%` / `%PROGRAMDATA%`，不能直接把可写数据留在 Program Files。
