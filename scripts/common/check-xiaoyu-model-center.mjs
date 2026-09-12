import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const req = (text, token, msg) => { if (!text.includes(token)) failures.push(msg) }
try {
  for (const rel of ['internal/xiaoyu/host/models.go','internal/xiaoyu/host/models_store.go','internal/xiaoyu/host/models_http.go','internal/xiaoyu/host/brain_model.go','frontend/src/features/settings/ModelManagementSection.vue','internal/platform/security/vault.go']) {
    if (!fs.existsSync(path.join(root, rel))) failures.push(`模型中心缺少：${rel}`)
  }
  const models = read('internal/xiaoyu/host/models.go')
  for (const provider of ['openai','openai-codex','deepseek','minimax','qwen','volcengine','zhipu','siliconflow','openrouter','gemini','claude','xai','ollama','lmstudio','custom']) req(models, `ID: "${provider}"`, `模型中心缺少 Provider：${provider}`)
  for (const token of ['ProtocolOpenAIResponses','ProtocolDeepSeek','ProtocolCodexAppServer','ModelAuthSubscription','ModelProviderAgent','ModelCapabilities','ResolveModelCapabilities','ReasoningEffort']) req(models, token, `模型中心缺少 Native Harness 能力：${token}`)
  const brain = read('internal/xiaoyu/host/brain_model.go')
  for (const token of ['reasoning.encrypted_content','reasoning_content','thoughtSignature','thinking','AttachmentIDs','openAIResponsesUserContent','anthropicUserContent','geminiUserParts']) req(brain, token, `Native Model Adapter 缺少：${token}`)
  const store = read('internal/xiaoyu/host/models_store.go')
  req(models, 'SecretRef       string         `json:"-"`', 'Model Profile SecretRef 必须禁止序列化')
  req(store, 'validateModelExtra', '模型额外参数必须经过敏感字段校验')
  for (const token of ['AuthMode: request.AuthMode','ModelAuthNeedsSecret','ModelBrainEligible','ModelProfileReady']) req(store, token, `模型 Provider Contract 缺少：${token}`)
  const types = read('frontend/src/shared/types/backend.ts')
  const saveModelRequest = types.match(/export interface XiaoYuSaveModelRequest \{([\s\S]*?)\n\}/)?.[1] ?? ''
  req(saveModelRequest, 'reasoningEffort:', 'XiaoYuSaveModelRequest 缺少 reasoningEffort，模型表单会在 vue-tsc 阶段失败')
  const frontend = read('frontend/src/features/settings/ModelManagementSection.vue')
  for (const token of ['模型管理','API Key','套餐 / 订阅','本地','测试连接','小鱼','form.reasoningEffort','form.authMode','brainEligible']) req(frontend, token, `模型管理 UI 缺少：${token}`)
  const app = read('internal/app/app_xiaoyu_models.go')
  for (const token of ['XiaoYuModelCatalog','SaveXiaoYuModel','TestXiaoYuModel','DiscoverXiaoYuModels','testCodexSubscription','"login", "status"']) req(app, token, `模型中心 Application API 缺少：${token}`)
} catch (error) { failures.push(error instanceof Error ? error.message : String(error)) }
if (failures.length) {
  console.error('AGMP XiaoYu Model Center Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}
console.log('AGMP XiaoYu Model Center Gate PASS (API · subscription/CLI · local · native providers · secret isolation · Brain eligibility)')
