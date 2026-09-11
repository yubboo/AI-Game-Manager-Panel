import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const fail = []
const read = (p) => fs.readFileSync(path.join(root, p), 'utf8')
const exists = (p) => fs.existsSync(path.join(root, p))

const appGo = read('internal/app/app.go')
const version = appGo.match(/Version\s*=\s*"([^"]+)"/)?.[1]?.trim()
if (!version) fail.push('无法读取 AGMP 版本')

const iss = read('distribution/installer/windows/AIGameManagerPanel.iss')
const wails = read('scripts/windows/lib/Wails.ps1')
const eulaPath = 'distribution/installer/windows/EULA-zh-CN.txt'
const eula = exists(eulaPath) ? read(eulaPath) : ''

const requireText = (label, haystack, needle) => {
  if (!haystack.includes(needle)) fail.push(`${label}: 缺少 ${needle}`)
}

if (!exists(eulaPath)) fail.push('缺少 EULA-zh-CN.txt')
if (eula.length < 1000) fail.push('EULA 内容过短或为空')
requireText('EULA', eula, '软件许可及服务协议')
requireText('EULA', eula, '只有在安装程序中选择“我接受本协议”后，您才能继续安装')

requireText('Installer', iss, 'LicenseFile=EULA-zh-CN.txt')
requireText('Installer', iss, 'LicenseAccepted=我接受本协议')
requireText('Installer', iss, 'LicenseNotAccepted=我不同意本协议')
requireText('Installer', iss, '您必须接受本协议才能继续安装')
requireText('Installer', iss, 'Name: "chinesesimp"; MessagesFile: "compiler:Default.isl"')
requireText('Installer', iss, 'DisableWelcomePage=no')
requireText('Installer', iss, 'DisableProgramGroupPage=no')
requireText('Installer', iss, 'SetupLogging=yes')
requireText('Installer', iss, 'CloseApplications=yes')
requireText('Installer', iss, 'UsePreviousAppDir=yes')
requireText('Installer', iss, 'UsePreviousTasks=yes')
requireText('Installer', iss, 'DirExistsTitle=文件夹已存在')
requireText('Installer', iss, 'DirExists=文件夹：')
requireText('Installer', iss, 'DirDoesntExistTitle=文件夹不存在')
requireText('Installer', iss, 'DirExistsWarning=auto')
requireText('Installer', iss, 'AI-Game-Manager-XiaoYu.exe')
requireText('Installer', iss, '普通用户安装器只包含完整 AGMP 的预编译运行产物')
requireText('Installer', iss, 'DestDir: "{app}\\internal\\xiaoyu"')
requireText('Installer', iss, 'function GetInstalledVersion(): String;')
requireText('Installer', iss, "{param:AGMPUPDATE|0}")
requireText('Installer', iss, '本向导将升级到版本')
requireText('Installer', iss, 'function InitializeUninstall(): Boolean;')
requireText('Installer', iss, '运行数据、日志、备份和实例信息将默认保留')
if (version) requireText('Installer', iss, `OutputBaseFilename=AI-Game-Manager-Panel-${version}-Windows-x64-Setup`)
if (iss.includes('AGMP_USE_INNO_CHINESE')) fail.push('安装器仍依赖旧的条件中文语言宏')
requireText('Wails Release', wails, 'EULA 为强制接受页')
requireText('Wails Release', wails, 'AI-Game-Manager-Panel-$script:AGMPVersion-Windows-x64-Setup.exe')
requireText('Wails Release', wails, '正式安装器')
requireText('Wails Release', wails, 'Get-FileHash')
requireText('Wails Release', wails, '.sha256')

if (fail.length) {
  console.error('AGMP Windows Installer Gate FAIL')
  for (const item of fail) console.error(` - ${item}`)
  process.exit(1)
}
console.log(`AGMP Windows Installer Gate PASS (AGMP ${version ?? 'unknown'} · zh-CN · mandatory EULA · upgrade-aware)`)
