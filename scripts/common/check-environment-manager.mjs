import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const req = (ok, message) => { if (!ok) failures.push(message) }
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8')

try {
  const service = read('internal/deploy/environment/service.go')
  const registry = read('internal/deploy/environment/registry.go')
  const java = read('internal/deploy/environment/java.go')
  const profiles = read('internal/deploy/environment/profiles.go')
  const tests = read('internal/deploy/environment/service_test.go') + '\n' + read('internal/deploy/environment/runtime_v2_test.go')
  const app = read('internal/app/app_environment.go')
  const http = read('internal/bridge/httpapi/server.go')
  const wails = read('internal/bridge/wails/app.go')
  const backend = read('frontend/src/shared/api/backend.ts')
  const types = read('frontend/src/shared/types/backend.ts')
  const view = read('frontend/src/features/settings/EnvironmentStorageSection.vue')
  const tools = read('internal/app/app_xiaoyu_tools.go')

  req(!service.includes('os/exec') && !registry.includes('os/exec') && !java.includes('os/exec') && !profiles.includes('os/exec'), 'Environment Manager 禁止直接使用 os/exec，所有进程验证/系统安装必须经过 platform/runtime')
  req(java.includes('platformruntime.Run'), 'Java -version 验证必须走共享 Process Runtime')
  req(profiles.includes('platformruntime.Run'), 'Linux 系统依赖安装必须走共享 Process Runtime')
  req(registry.includes('RuntimeJava') && registry.includes('RuntimeSteamCMD'), 'Runtime Registry 必须同时支持 Java 与 SteamCMD')
  for (const major of ['8','17','21','25']) req(registry.includes(major), `Runtime Catalog 缺少 Java ${major}`)
  req(service.includes('steamCMDLinuxURL') && service.includes('steamcmd_linux.tar.gz'), 'SteamCMD 必须支持 Linux 官方安装包')
  req(service.includes('steamCMDWindowsURL'), 'SteamCMD 必须保留 Windows 官方安装')
  req(service.includes('managedSteamCMDExecutableName') && service.includes('steamcmd.sh'), 'Linux managed SteamCMD 必须识别 steamcmd.sh')
  req(tests.includes('TestInitializationDoesNotRequireSteamCMD'), '基础 Runtime Manager 初始化必须有“不强制 SteamCMD”回归测试')
  req(!tests.includes('TestInitializationWithoutSteamCMDFailsClosed'), '禁止恢复“无 SteamCMD 初始化失败”的旧测试')
  req(tests.includes('TestGameProfilesDeclareOnlyRelevantDependencies'), '缺少按游戏声明 Runtime 依赖测试')
  req(tests.includes('TestSecureJoinRejectsArchiveTraversal'), '缺少压缩包路径逃逸测试')
  req(tests.includes('TestExternalRuntimeRemovalNeverDeletesExternalFile'), '缺少外部 Runtime 删除边界测试')
  req(tests.includes('TestRuntimeRegistryPersistsDefaultSelection') && tests.includes('TestRuntimeRepairPreservesExistingDefault'), '缺少 Runtime Registry 持久化/默认版本回归测试')
  req(tests.includes('TestJavaVersionParserRecognizesLegacyAndModernVersions'), '缺少 Java 8/17/21/25 版本解析回归测试')
  req(tests.includes('TestExtractTarGZRejectsSymlink') && tests.includes('TestExtractZIPRejectsSymlink'), '缺少 ZIP/TAR 特殊文件/符号链接攻击测试')
  req(tests.includes('TestJavaInstallIsSerializedByInstallLock'), '缺少 Runtime 并发安装串行化测试')
  req(profiles.includes('GameID: "minecraft"') && profiles.includes('GameID: "dst"'), 'Game Runtime Profile 必须包含 Minecraft 与 DST')
  req(profiles.includes('steamcmd-linux-32bit-loader') && profiles.includes('steamcmd-linux-32bit-cpp'), 'Linux SteamCMD 缺少 32-bit 系统依赖诊断')
  req(app.includes('EnvironmentRuntimeCatalog') && app.includes('InstallJavaRuntime') && app.includes('EnvironmentGameRuntimeProfiles'), 'Application 未完整接入 Runtime Manager v2')
  for (const route of ['/api/v1/environment/runtime/catalog','/api/v1/environment/runtime/java/install','/api/v1/environment/games/profiles','/api/v1/environment/system-prerequisites/install']) req(http.includes(route), `HTTP 缺少 ${route}`)
  req(wails.includes('GetEnvironmentRuntimeCatalog') && wails.includes('InstallJavaRuntime') && wails.includes('GetEnvironmentGameRuntimeProfiles'), 'Wails 未完整接入 Runtime Manager v2')
  req(backend.includes('environmentRuntimeCatalog') && backend.includes('installJavaRuntime') && backend.includes('environmentGameRuntimeProfiles'), 'TypeScript Backend 未完整接入 Runtime Manager v2')
  req(types.includes('EnvironmentRuntimeCatalog') && types.includes('EnvironmentGameRuntimeProfile'), 'TypeScript 类型缺少 Runtime Manager v2')
  req(view.includes('Java 多版本 Runtime') && view.includes('游戏运行环境检查'), '系统设置 UI 缺少 Java 多版本 / Game Runtime Profile')
  req(view.includes('serverPlatformLabel') && view.includes('catalog.value?.platform') && !view.includes('navigator.platform'), 'Web UI 必须使用 AGMP 服务端平台，不得使用浏览器 navigator.platform 判断服务器 Runtime')
  req(tools.includes('environment.catalog') && tools.includes('environment.game_profile') && tools.includes('environment.install_java') && tools.includes('environment.install_steamcmd') && tools.includes('environment.install_system_prerequisite'), 'XiaoYu 未复用同一 Environment Runtime Manager Tools')
  req(tools.includes('Name: "environment.install_system_prerequisite"') && tools.includes('Risk: xiaoyucontract.RiskSystem'), 'Linux 系统前置依赖 Tool 必须标记 RiskSystem，走统一高风险审批/二验链')
} catch (error) {
  failures.push(`Environment Manager Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP Environment Manager v2 Gate FAIL')
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log('AGMP Environment Manager v2 Gate PASS (server platform · Java multi-version · Windows/Linux SteamCMD · game profiles · shared Process Runtime)')
