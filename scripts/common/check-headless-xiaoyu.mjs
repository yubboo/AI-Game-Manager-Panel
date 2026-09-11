import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const has = (rel, token, message) => { if (!read(rel).includes(token)) failures.push(message) }

try {
  const docker = read('distribution/docker/Dockerfile')
  const serverBuild = read('scripts/linux/actions/build-server.sh')
  const linuxBuild = read('scripts/linux/actions/build.sh')
  const deb = read('scripts/linux/actions/build-deb.sh')
  const installer = read('distribution/installer/linux/install.sh')
  const webMain = read('cmd/aigame-manager-web/main.go')
  const serverConfig = JSON.parse(read('configs/server.json'))
  const release = JSON.parse(read('configs/release.json'))
  const ci = read('.github/workflows/safety.yml')
  const headlessTest = read('internal/bridge/httpapi/server_xiaoyu_harness_dev_test.go')

  for (const token of ['FROM --platform=$TARGETPLATFORM rust:1.85-alpine AS xiaoyu-builder','cargo build --release -p xiaoyu-core','AI-Game-Manager-XiaoYu','AGMP_XIAOYU_RUNTIME']) {
    if (!docker.includes(token)) failures.push(`Docker 完整 AI Runtime 缺少：${token}`)
  }
  for (const source of [serverBuild, linuxBuild]) {
    for (const token of ['cargo','xiaoyu-core','AI-Game-Manager-XiaoYu']) if (!source.includes(token)) failures.push(`Linux 构建链禁止生成 AI-less 包，缺少：${token}`)
  }
  for (const source of [deb, installer]) if (!source.includes('AI-Game-Manager-XiaoYu')) failures.push('Linux 安装/DEB 必须携带 XiaoYu Runtime')
  if (!installer.includes('AGMP_XIAOYU_RUNTIME=')) failures.push('systemd Server 必须显式连接预编译 XiaoYu Runtime')
  if (!webMain.includes('allow-remote-web') || !webMain.includes('AllowRemote:')) failures.push('Headless Web 缺少显式远程监听安全开关')
  if (serverConfig.allowRemoteWeb !== false) failures.push('configs/server.json 必须保持 loopback/remote 安全默认关闭')
  if (release.linux?.precompiledXiaoYuIncluded !== true || release.linux?.serverEdition?.headlessAI !== true) failures.push('Release 元数据必须声明 Linux Headless XiaoYu 为必选能力')
  if (release.productArchitecture?.deliveryTargets?.linuxServerWeb?.xiaoyu !== true || release.productArchitecture?.deliveryTargets?.dockerWeb?.xiaoyu !== true) failures.push('Linux Server Web / Docker Web 必须是完整 XiaoYu 产品目标')

  if (!ci.includes('Linux Headless + Web + XiaoYu') || !ci.includes('TestLinuxHeadlessWebCanReachPrecompiledXiaoYuRuntime')) failures.push('CI 必须真实验证 Linux Web 可以连接预编译 XiaoYu Runtime')
  if (!headlessTest.includes('AGMP_HEADLESS_XIAOYU_RUNTIME') || !headlessTest.includes('/api/v1/xiaoyu/runtime')) failures.push('Headless Web 集成测试必须从 HTTP 入口验证 XiaoYu Runtime')

  const webImports = webMain.match(/github\.com\/yubboo\/AI-Game-Manager-Panel\/internal\/[^"]+/g) ?? []
  if (webImports.some(value => /bridge\/wails|desktop|electron/.test(value))) failures.push('Headless Web 主程序禁止依赖桌面 Adapter')
} catch (error) {
  failures.push(error instanceof Error ? error.message : String(error))
}

if (failures.length) {
  console.error('AGMP Headless XiaoYu Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}
console.log('AGMP Headless XiaoYu Gate PASS (Linux/Docker package AGMP Core + XiaoYu + Web, desktop not required)')
