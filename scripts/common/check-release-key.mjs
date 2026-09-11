import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const allowUnconfigured = process.argv.includes('--allow-unconfigured')
const ringPath = path.join(root, 'internal/system/license/vendor_public_keys.json')
const failures = []
const warnings = []

function fingerprint(raw) {
  return `SHA256:${crypto.createHash('sha256').update(raw).digest('hex').toUpperCase()}`
}

function keyId(raw) {
  const short = crypto.createHash('sha256').update(raw).digest('hex').toUpperCase().slice(0, 32)
  return `AGMP-KID-${short.match(/.{1,8}/g).join('-')}`
}

if (!fs.existsSync(ringPath)) {
  failures.push('缺少发行公钥环：internal/system/license/vendor_public_keys.json')
} else {
  let ring
  try {
    ring = JSON.parse(fs.readFileSync(ringPath, 'utf8'))
  } catch (error) {
    failures.push(`发行公钥环 JSON 无法解析：${error instanceof Error ? error.message : String(error)}`)
  }
  if (ring) {
    if (ring.schemaVersion !== 1) failures.push(`发行公钥环 schemaVersion 必须为 1，当前=${ring.schemaVersion}`)
    if (!Array.isArray(ring.keys) || ring.keys.length === 0) failures.push('发行公钥环至少要保留一个可信公钥')
    const seenIds = new Set()
    const seenKeys = new Set()
    let activeCount = 0
    let activeMatch = null
    for (const [index, entry] of (ring.keys ?? []).entries()) {
      const label = `keys[${index}]`
      const id = String(entry?.keyId ?? '').trim()
      const publicKey = String(entry?.publicKey ?? '').trim()
      const status = String(entry?.status ?? '').trim().toLowerCase()
      if (!id) failures.push(`${label} 缺少 keyId`)
      if (!publicKey) failures.push(`${label} 缺少 publicKey`)
      if (!['active', 'retired', 'legacy'].includes(status)) failures.push(`${label} status 必须是 active/retired/legacy`)
      let raw = null
      try {
        raw = Buffer.from(publicKey, 'base64')
        if (raw.length !== 32) failures.push(`${label} 不是 32-byte Ed25519 公钥`)
      } catch {
        failures.push(`${label} publicKey 不是合法 Base64`)
      }
      if (raw?.length === 32) {
        const actualId = keyId(raw)
        const actualFingerprint = fingerprint(raw)
        if (id && id !== actualId) failures.push(`${label} keyId 与公钥不匹配：${id} != ${actualId}`)
        if (String(entry?.fingerprint ?? '').trim() !== actualFingerprint) failures.push(`${label} fingerprint 与公钥不匹配`)
      }
      if (seenIds.has(id)) failures.push(`发行公钥环存在重复 keyId：${id}`)
      if (seenKeys.has(publicKey)) failures.push(`发行公钥环存在重复 publicKey：${id}`)
      if (id) seenIds.add(id)
      if (publicKey) seenKeys.add(publicKey)
      if (status === 'active') {
        activeCount += 1
        if (id === String(ring.activeKeyId ?? '').trim()) activeMatch = entry
      }
    }

    const activeKeyId = String(ring.activeKeyId ?? '').trim()
    if (!activeKeyId) {
      const message = '尚未配置 active 发行密钥。源码开发可继续；正式 Release 前：本机已有发行密钥请运行 AGMP-License-Admin.bat -> 4 同步公钥；仅首次初始化或明确轮换时使用 -> 1。'
      if (allowUnconfigured) warnings.push(message)
      else failures.push(message)
    } else {
      if (activeCount !== 1) failures.push(`配置 activeKeyId 后必须且只能有一个 status=active，当前=${activeCount}`)
      if (!activeMatch) failures.push(`activeKeyId=${activeKeyId} 没有对应的 status=active 公钥`)
    }
  }
}

for (const item of warnings) console.warn(`[WARN] ${item}`)
if (failures.length) {
  console.error(`AGMP Release Key Gate FAIL (${failures.length})`)
  for (const item of failures) console.error(` - ${item}`)
  process.exit(1)
}
console.log(`AGMP Release Key Gate PASS${warnings.length ? ' (development-only: active issuer key not configured)' : ''}`)
