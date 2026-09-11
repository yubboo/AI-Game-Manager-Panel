import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8')
const appGo = read('internal/app/app.go')
const version = appGo.match(/Version\s*=\s*"([^"]+)"/)?.[1] || ''
const cfg = JSON.parse(read('configs/update.json'))
const service = read('internal/deploy/updater/service.go')
const test = read('internal/deploy/updater/service_test.go')
const installer = read('distribution/installer/windows/AIGameManagerPanel.iss')
const wails = read('scripts/windows/lib/Wails.ps1')
const settings = read('frontend/src/features/settings/UpdateSection.vue')

function req(ok, message) { if (!ok) failures.push(message) }
req(cfg.enabled === true, 'update.json 必须 enabled=true')
req(cfg.provider === 'github-releases', 'update provider 必须 github-releases')
req(cfg.repository === 'yubboo/AI-Game-Manager-Panel', 'update repository 必须为官方仓库')
req(cfg.channel === 'stable', 'update channel 必须 stable')
req(cfg.requireSha256 === true, '自动安装必须要求 SHA256')
req(cfg.preserveRuntimeData === true, '更新策略必须保留 runtime 数据')
req(cfg.assetPattern === 'AI-Game-Manager-Panel-{version}-Windows-x64-Setup.exe', 'Setup asset 命名规则不一致')
req(service.includes('/releases/latest'), 'Go updater 缺少 GitHub latest release API')
req(service.includes('compareVersions'), 'Go updater 缺少数字版本比较')
req(service.includes('PrepareLatest'), 'Go updater 缺少下载准备')
req(service.includes('LaunchInstaller'), 'Go updater 缺少安装器启动')
req(service.includes('安装器 SHA256 校验失败'), 'Go updater 必须 fail-closed 校验 SHA256')
req(test.includes('0.1.100') && test.includes('0.2.0'), 'Updater 单测必须覆盖 0.1.100 -> 0.2.0 版本规则')
req(installer.includes('DirExistsTitle=文件夹已存在'), 'Installer Folder Exists 标题未中文化')
req(installer.includes('DirDoesntExistTitle=文件夹不存在'), 'Installer Folder Does Not Exist 未中文化')
req(installer.includes('{param:AGMPUPDATE|0}'), 'Installer 缺少自动更新启动参数识别')
req(installer.includes('GetInstalledVersion'), 'Installer 缺少已安装版本检测')
req(wails.includes('Get-FileHash') && wails.includes('.sha256'), 'Wails Release 缺少安装器 SHA256 输出')
req(settings.includes('下载并安装更新') && settings.includes('检查更新'), '设置中心更新 UI 未完成')
req(/^\d+\.\d+\.\d+$/.test(version), `当前版本格式无效：${version}`)

if (failures.length) {
  console.error('AGMP Updater Gate FAIL')
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log(`AGMP Updater Gate PASS (v${version} · GitHub stable · SHA256 · interactive Windows upgrade)`)
