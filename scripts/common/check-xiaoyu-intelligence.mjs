import fs from 'node:fs'
import path from 'node:path'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8')
const requireCheck = (ok, message) => { if (!ok) failures.push(message) }

try {
  const store = read('internal/xiaoyu/host/intelligence.go')
  const runs = read('internal/xiaoyu/host/runs.go')
  const loop = read('internal/xiaoyu/host/loop.go')
  const app = read('internal/app/app.go')
  const appIntelligence = read('internal/app/app_xiaoyu_intelligence.go')
  const harness = read('internal/app/app_xiaoyu_harness.go')
  const tools = read('internal/app/app_xiaoyu_tools.go')
  const capabilityMap = read('internal/app/app_xiaoyu_capabilities.go')
  const workbench = read('frontend/src/features/xiaoyu/AIWorkbenchView.vue')
  const rightContext = read('frontend/src/features/xiaoyu/XiaoYuRightContext.vue')
  const rust = read('rust/crates/xiaoyu-core/src/lib.rs')
  const http = read('internal/bridge/httpapi/server.go')
  const wails = read('internal/bridge/wails/app.go')
  const types = read('frontend/src/shared/types/backend.ts')
  const backend = read('frontend/src/shared/api/backend.ts')
  const settings = read('frontend/src/features/settings/SettingsView.vue')
  const ui = read('frontend/src/features/settings/XiaoYuIntelligenceSection.vue')
  const appTests = read('internal/app/app_xiaoyu_intelligence_dev_test.go')
  const releaseTest = read('internal/bridge/httpapi/server_xiaoyu_intelligence_release_test.go')

  requireCheck(store.includes('VisibilityPrivate') && store.includes('VisibilityGroup') && store.includes('VisibilityOrganization'), 'Intelligence 必须显式使用 Private / Group / Organization 三层可见性')
  requireCheck(!store.includes('VisibilityGlobal') && !store.includes('cloud-global') && !store.includes('public-global'), '禁止出现跨 AGMP 实例/公网 Global Intelligence')
  requireCheck(store.includes('if org != wantedOrg') && store.includes('return false'), 'Intelligence 查询必须先按 Organization fail-closed')
  requireCheck(store.includes('item.Sensitivity == MemorySensitive'), 'Sensitive Memory 必须在进入模型 Context 前过滤')
  requireCheck(store.includes('containsSecretMaterial'), 'Memory/Skill/Expert 持久化必须拒绝明显秘密材料')
  requireCheck(store.includes('len(ctx.Memories) >= 24') && store.includes('rankMemories') && store.includes('len(ctx.Skills) >= 10') && store.includes('len(ctx.Experts) >= 6'), '模型 Intelligence Context 必须有紧凑的 Host 侧数量上限')
  requireCheck(app.includes('NewIntelligenceStore') && app.includes('xiaoyuIntelligence'), 'Application 必须拥有实例本地 Intelligence Store')
  requireCheck(appIntelligence.includes('requireXiaoYuMember') && appIntelligence.includes('canPublishXiaoYuIntelligence'), 'Intelligence 管理必须经过成员核心授权并限制共享发布权限')
  requireCheck(appIntelligence.includes('MemoryUser') && appIntelligence.includes('VisibilityPrivate'), '个人/会话/任务 Memory 必须由 Application 收紧为 Private')
  requireCheck(appIntelligence.includes('MemoryExperience') && appIntelligence.includes('request.Scope.ID = "*"'), 'Group/Organization Experience 必须使用显式 wildcard scope，不得保留调用方伪造的用户 scope')
  requireCheck(runs.includes('TaskID: id') && harness.includes('value.TaskID = ""'), 'TaskID 必须由 RunManager 服务端生成，客户端 TaskID 必须清空')
  requireCheck(loop.includes('Intelligence IntelligenceContext') && loop.includes('WithIntelligence'), '每个 Rust Agent Frame 必须能接收 Host 过滤后的 Intelligence Context')
  requireCheck(harness.includes('RequireCoreAccess(token)') && harness.includes('groupID, userID = "", ""'), 'Detached Run 每轮 Intelligence 必须重新验证成员授权，Supervisor 不得隐式读取成员 Private/Group Intelligence')
  requireCheck(appTests.includes('TestXiaoYuIntelligenceRunResolverRechecksAccessAndLimitsSupervision') && appTests.includes('TestSharedExperienceUsesExplicitWildcardScope'), '必须有 Detached Run 重新授权、Supervisor 私有隔离和共享 Experience scope 回归测试')
  requireCheck(tools.includes('Name: "memory.remember"') && tools.includes('Visibility: xiaoyuhost.VisibilityPrivate'), 'memory.remember 必须存在且只能写 Private Memory')
  requireCheck(tools.includes('server-owned task scope') && tools.includes('getXiaoYuInvocationContext'), 'memory.remember 必须使用认证后的 server-owned Run Context')
  requireCheck(store.includes('builtin.skill.agmp-autonomy') && store.includes('builtin.expert.agmp') && store.includes('ContextForGoal'), '小鱼必须始终拥有 AGMP 自主执行基础 Skill/Expert，并按目标筛选领域 Intelligence')
  requireCheck(store.includes('ModuleCatalog') && capabilityMap.includes('xiaoyuSystemModuleCatalog'), 'XiaoYu 必须获得 AGMP 全量 Module Catalog 系统认知')
  requireCheck(tools.includes('Name: "agmp.capability.search"') && capabilityMap.includes('platformConfig.Modules.Modules') && capabilityMap.includes('skeleton/planned'), '小鱼必须能检索 AGMP 产品 Module Map 与当前真实 Tool Capability')
  requireCheck(loop.includes('completionEvidenceGap') && loop.includes('run/verification-required'), 'Agent Loop 必须阻止没有执行证据或缺少执行后验证的提前完成')
  requireCheck(rust.includes('不要在每个普通步骤后反问') && rust.includes('is_procedural_wait') && rust.includes('agmp.capability.search'), 'Rust Agent Runtime 必须拒绝程序性 wait，并在能力未知时先发现 Capability')
  requireCheck(workbench.includes('xy-inline-approval') && workbench.includes('resolveInlineApproval') && !rightContext.includes('等待你的决定'), '任务审批必须显示在对话输入区上方，右侧上下文不得继续承载审批操作')
  requireCheck(rust.includes('frame.intelligence') && rust.includes('Host Tool Contract') && rust.includes('Expert / Skill > Memory'), 'Rust XiaoYu Runtime 必须明确 Intelligence 不可信优先级和 Host 最终权威')
  requireCheck(rust.includes('memory.remember') && rust.includes('不要保存秘密'), 'Rust Agent Runtime 必须约束主动记忆行为')
  requireCheck(http.includes('/api/v1/xiaoyu/intelligence') && wails.includes('GetXiaoYuIntelligenceCatalog'), 'Web/Wails 必须共用同一 Intelligence Application API')
  requireCheck(releaseTest.includes('RequiresOfficialLicenseInReleaseBuild') && releaseTest.includes('/api/v1/xiaoyu/intelligence'), 'Release build 必须有 Intelligence 无官方 License fail-closed 回归测试')
  requireCheck(types.includes('XiaoYuIntelligenceCatalog') && backend.includes('xiaoyuIntelligenceCatalog()'), 'Frontend TypeScript/Backend 必须暴露 Intelligence 契约')
  requireCheck(settings.includes('XiaoYuIntelligenceSection') && ui.includes('Organization 共享仅限当前组织成员'), '设置中心必须提供 Intelligence 管理并明确组织共享不是公网共享')
} catch (error) {
  failures.push(`XiaoYu Intelligence Gate 检查失败：${error instanceof Error ? error.message : String(error)}`)
}

if (failures.length) {
  console.error('AGMP XiaoYu Intelligence Gate FAIL')
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log('AGMP XiaoYu Intelligence Gate PASS (goal-relevant memory/skills/experts · AGMP system map · capability search · evidence gate · inline approval)')
