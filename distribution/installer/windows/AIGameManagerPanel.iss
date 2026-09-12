; AI游戏管理器面板 Windows 正式安装器（Inno Setup）
; 0.1.74：全中文已有目录提示 + 版本升级识别 + 用户数据保留 + 自动更新安装器衔接
#define MyAppName "AI游戏管理器面板"
#define MyAppVersion "0.2.14"
#define MyAppVersionNumeric "0.2.14.0"
#define MyAppPublisher "AI Game Manager Panel"
#define MyAppURL "https://github.com/yubboo/AI-Game-Manager-Panel"
#define MyAppExeName "AI-Game-Manager-Panel.exe"

[Setup]
AppId={{0DFB0A2A-834E-4F69-958C-DB0B9E2E7811}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}/releases/latest
DefaultDirName={localappdata}\Programs\AI-Game-Manager-Panel
DefaultGroupName=AI游戏管理器面板
DisableWelcomePage=no
DisableProgramGroupPage=no
AllowNoIcons=yes
PrivilegesRequired=lowest
OutputDir=..\..\..\build\work\windows-wails\release-stage\installer
OutputBaseFilename=AI-Game-Manager-Panel-0.2.14-Windows-x64-Setup
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
WizardSizePercent=110
LicenseFile=EULA-zh-CN.txt
SetupLogging=yes
CloseApplications=yes
RestartApplications=no
RestartIfNeededByRun=no
UsePreviousAppDir=yes
UsePreviousGroup=yes
UsePreviousTasks=yes
DirExistsWarning=auto
UninstallDisplayName={#MyAppName} {#MyAppVersion}
UninstallDisplayIcon={app}\{#MyAppExeName}
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0.17763
VersionInfoVersion={#MyAppVersionNumeric}
VersionInfoCompany={#MyAppPublisher}
VersionInfoDescription={#MyAppName} Windows 安装程序
VersionInfoProductName={#MyAppName}
VersionInfoProductVersion={#MyAppVersion}

[Languages]
; 始终使用 Inno 自带 Default.isl，中文文本由本脚本覆盖，不依赖外部 ChineseSimplified.isl。
Name: "chinesesimp"; MessagesFile: "compiler:Default.isl"

[Messages]
SetupAppTitle=安装程序
SetupWindowTitle=安装 - %1
UninstallAppTitle=卸载程序
UninstallAppFullTitle=卸载 %1
InformationTitle=信息
ConfirmTitle=确认
ErrorTitle=错误
ExitSetupTitle=退出安装
ExitSetupMessage=安装尚未完成。如果现在退出，AI游戏管理器面板将不会被安装。%n%n确定退出安装吗？
ButtonBack=< 上一步(&B)
ButtonNext=下一步(&N) >
ButtonInstall=安装(&I)
ButtonOK=确定
ButtonCancel=取消
ButtonYes=是(&Y)
ButtonNo=否(&N)
ButtonFinish=完成(&F)
ButtonBrowse=浏览(&B)...
ButtonWizardBrowse=浏览(&B)...
ButtonNewFolder=新建文件夹(&M)
ClickNext=单击“下一步”继续，或单击“取消”退出安装。
BrowseDialogTitle=选择文件夹
BrowseDialogLabel=请选择目标文件夹，然后单击“确定”。
DirExistsTitle=文件夹已存在
DirExists=文件夹：%n%n%1%n%n已经存在。是否仍然安装到该文件夹？
DirDoesntExistTitle=文件夹不存在
DirDoesntExist=文件夹：%n%n%1%n%n尚不存在。是否创建该文件夹并继续安装？
NewFolderName=新建文件夹
WelcomeLabel1=欢迎使用 [name] 安装向导
WelcomeLabel2=本向导将把 [name/ver] 安装到您的 Windows 电脑。%n%n建议继续前关闭正在运行的旧版本程序。
WizardLicense=软件许可协议
LicenseLabel=继续安装前，请阅读以下软件许可及服务协议。
LicenseLabel3=您必须接受本协议才能继续安装。请选择“我接受本协议”，否则“下一步”将不可继续。
LicenseAccepted=我接受本协议(&A)
LicenseNotAccepted=我不同意本协议(&D)
WizardSelectDir=选择安装位置
SelectDirDesc=请选择 [name] 的安装目录。
SelectDirLabel3=安装程序将把 [name] 安装到以下文件夹。
SelectDirBrowseLabel=如需修改安装位置，请单击“浏览”；确认后单击“下一步”。
DiskSpaceMBLabel=至少需要 [mb] MB 可用磁盘空间。
WizardSelectProgramGroup=选择开始菜单文件夹
SelectStartMenuFolderDesc=请选择用于创建程序快捷方式的开始菜单文件夹。
SelectStartMenuFolderLabel3=安装程序将在以下开始菜单文件夹中创建快捷方式。
SelectStartMenuFolderBrowseLabel=确认后单击“下一步”，或输入其他文件夹名称。
NoProgramGroupCheck2=不创建开始菜单文件夹(&D)
WizardSelectTasks=选择附加任务
SelectTasksDesc=请选择安装时需要执行的附加任务。
SelectTasksLabel2=请选择需要的附加任务，然后单击“下一步”。
WizardReady=准备安装
ReadyLabel1=安装程序已经准备好在您的电脑上安装 [name]。
ReadyLabel2a=单击“安装”开始；如需修改设置，请单击“上一步”。
ReadyLabel2b=单击“安装”开始安装。
ReadyMemoDir=安装位置：
ReadyMemoGroup=开始菜单：
ReadyMemoTasks=附加任务：
WizardPreparing=正在准备安装
PreparingDesc=正在准备安装 [name]，请稍候。
CannotContinue=安装程序无法继续。请单击“取消”退出。
ApplicationsFound=以下程序正在使用需要更新的文件。建议允许安装程序自动关闭这些程序。
ApplicationsFound2=以下程序正在使用需要更新的文件。建议允许安装程序自动关闭；安装完成后将尝试恢复。
CloseApplications=自动关闭相关程序(&A)
DontCloseApplications=不要关闭相关程序(&D)
ErrorCloseApplications=无法自动关闭全部相关程序。请手动关闭正在运行的 AI游戏管理器面板 后重试。
WizardInstalling=正在安装
InstallingLabel=正在安装 [name]，请稍候。
FinishedHeadingLabel=[name] 安装完成
FinishedLabelNoIcons=[name] 已成功安装到您的电脑。
FinishedLabel=[name] 已成功安装。您可以通过开始菜单或已创建的快捷方式启动程序。
ClickFinish=单击“完成”退出安装向导。
RunEntryExec=启动 %1
SetupAborted=安装未完成。%n%n请解决问题后重新运行安装程序。
StatusClosingApplications=正在关闭相关程序...
StatusCreateDirs=正在创建目录...
StatusExtractFiles=正在复制程序文件...
StatusCreateIcons=正在创建快捷方式...
StatusCreateRegistryEntries=正在写入系统安装信息...
StatusSavingUninstall=正在保存卸载信息...
StatusRunProgram=正在完成安装...

[Tasks]
Name: "desktopicon"; Description: "创建桌面快捷方式"; GroupDescription: "附加任务："; Flags: unchecked

[Files]
; 普通用户安装器只包含完整 AGMP 的预编译运行产物。内部 AI Core/Core 二进制放入 internal 子目录，不创建用户入口；禁止加入 Rust/Cargo/MSVC/Build Tools/源码开发脚本。
Source: "..\..\..\build\work\windows-wails\bin\AI-Game-Manager-Panel.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\..\build\work\windows-wails\bin\AI-Game-Manager-Web.exe"; DestDir: "{app}\internal\core"; Flags: ignoreversion
Source: "..\..\..\build\work\windows-wails\bin\AI-Game-Manager-XiaoYu.exe"; DestDir: "{app}\internal\xiaoyu"; Flags: ignoreversion
Source: "..\..\..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "EULA-zh-CN.txt"; DestDir: "{app}\licenses"; DestName: "AGMP-EULA-zh-CN.txt"; Flags: ignoreversion
Source: "..\..\..\configs\*"; DestDir: "{app}\configs"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\..\licenses\*"; DestDir: "{app}\licenses"; Flags: ignoreversion recursesubdirs createallsubdirs

[Dirs]
; 当前 Windows 正式版按用户安装到 LocalAppData，因此 runtime 目录可直接写入。
; 升级安装不会主动删除这里的用户数据。
Name: "{app}\runtime\data"
Name: "{app}\runtime\log"
Name: "{app}\runtime\backups"
Name: "{app}\runtime\instances"
Name: "{app}\runtime\temp"
Name: "{app}\runtime\exports"
Name: "{app}\runtime\plugins"
Name: "{app}\runtime\cache"

[Icons]
Name: "{group}\AI游戏管理器面板"; Filename: "{app}\AI-Game-Manager-Panel.exe"; WorkingDir: "{app}"
Name: "{autodesktop}\AI游戏管理器面板"; Filename: "{app}\AI-Game-Manager-Panel.exe"; WorkingDir: "{app}"; Tasks: desktopicon

[Run]
Filename: "{app}\AI-Game-Manager-Panel.exe"; Description: "安装完成后启动 AI游戏管理器面板"; WorkingDir: "{app}"; Flags: nowait postinstall skipifsilent

[Code]
function GetInstalledVersion(): String;
var
  VersionText: String;
  ExePath: String;
begin
  Result := '';
  ExePath := ExpandConstant('{app}\{#MyAppExeName}');
  if FileExists(ExePath) and GetVersionNumbersString(ExePath, VersionText) then
    Result := VersionText;
end;

procedure InitializeWizard;
var
  ExistingVersion: String;
  UpdateLaunch: Boolean;
begin
  ExistingVersion := GetInstalledVersion();
  UpdateLaunch := ExpandConstant('{param:AGMPUPDATE|0}') = '1';
  if ExistingVersion <> '' then
  begin
    WizardForm.WelcomeLabel2.Caption :=
      '检测到已安装的 AI游戏管理器面板 ' + ExistingVersion + '。' + #13#10 + #13#10 +
      '本向导将升级到版本 {#MyAppVersion}。升级只替换程序文件和新版静态配置，不会主动删除 runtime 中的账号、许可证、服务器实例、备份、日志和用户设置。' + #13#10 + #13#10 +
      '继续前建议关闭正在运行的旧版本程序。';
  end
  else if UpdateLaunch then
  begin
    WizardForm.WelcomeLabel2.Caption :=
      'AI游戏管理器面板已下载版本 {#MyAppVersion} 更新。' + #13#10 + #13#10 +
      '请按向导完成升级。安装程序会保留 runtime 用户数据，并只更新程序文件。';
  end;
end;

function InitializeUninstall(): Boolean;
begin
  Result := MsgBox(
    '即将卸载 AI游戏管理器面板。' + #13#10 + #13#10 +
    '卸载程序会移除程序文件和快捷方式；为防止误删服务器资料，runtime 目录中的运行数据、日志、备份和实例信息将默认保留。' + #13#10 + #13#10 +
    '确定继续卸载吗？',
    mbConfirmation, MB_YESNO) = IDYES;
end;
