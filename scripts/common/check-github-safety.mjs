import fs from 'node:fs'
import path from 'node:path'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..', '..')
const failures = []
const warnings = []
const gitignorePath = path.join(root, '.gitignore')

const requiredIgnoreRules = [
  '/build/', 'runtime/*', '!runtime/README.md', 'data/', 'log/', 'backups/', 'instances/',
  '.env', '.env.*', '*.key', '*.priv', '*.seed', '*.pem', '*.p12', '*.pfx', '*.cdk', '*.license',
  '**/activation.json', '**/accounts.json', '**/bootstrap.lock', '**/install.id', '**/device.id',
  '**/cluster_token.txt', '**/credentials.json', '**/secrets.json', '**/agmp-release-private.*',
]

if (!fs.existsSync(gitignorePath)) {
  failures.push('缺少根目录 .gitignore')
} else {
  const ignore = fs.readFileSync(gitignorePath, 'utf8')
  for (const rule of requiredIgnoreRules) if (!ignore.includes(rule)) failures.push(`.gitignore 缺少安全规则：${rule}`)
}

const normalize = value => value.replaceAll('\\', '/').replace(/^\.\//, '')
const base = value => path.posix.basename(normalize(value)).toLowerCase()

function forbiddenName(rel) {
  const p = normalize(rel)
  const lower = p.toLowerCase()
  const name = base(p)
  if (lower === '.env.example') return null
  if (name === '.env' || name.startsWith('.env.')) return '环境变量文件'
  if (/\.(key|priv|seed|pem|p12|pfx|jks|keystore)$/i.test(name)) return '密钥/私钥容器'
  if (/\.(cdk|lic|license|bflc)$/i.test(name)) return '真实授权材料'
  if (['activation.json','accounts.json','bootstrap.lock','install.id','device.id','credentials.json','secrets.json'].includes(name)) return '本机认证/授权状态'
  if (['cluster_token.txt','server_token.txt','adminlist.txt','whitelist.txt','blocklist.txt'].includes(name)) return '游戏服务器私密状态'
  if (/security[-_ ]?key.*\.txt$/i.test(name)) return '用户安全密钥导出'
  if (lower.startsWith('runtime/') && lower !== 'runtime/readme.md') return '运行时数据'
  for (const dir of ['data/','log/','logs/','backups/','instances/','temp/','exports/','plugins/','cache/','build/']) {
    if (lower.startsWith(dir)) return '运行时/构建数据'
  }
  return null
}

const skipDirs = new Set(['.git','node_modules','dist','build'])
function walk(dir, prefix = '') {
  const out = []
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const rel = normalize(path.posix.join(prefix, entry.name))
    if (entry.isDirectory()) {
      if (skipDirs.has(entry.name)) continue
      if (rel === 'runtime') { out.push('runtime/README.md'); continue }
      out.push(...walk(path.join(dir, entry.name), rel))
    } else out.push(rel)
  }
  return out
}

let candidates = []
if (fs.existsSync(path.join(root, '.git'))) {
  try {
    const raw = execFileSync('git', ['ls-files', '-co', '--exclude-standard', '-z'], { cwd: root, encoding: 'utf8' })
    candidates = raw.split('\0').filter(Boolean).map(normalize)
  } catch (error) {
    warnings.push(`无法调用 git ls-files，回退源码树扫描：${error instanceof Error ? error.message : String(error)}`)
    candidates = walk(root)
  }
} else {
  candidates = walk(root)
}

const unique = [...new Set(candidates)]
for (const rel of unique) {
  const reason = forbiddenName(rel)
  if (reason) failures.push(`禁止进入 Git：${rel}（${reason}）`)
}

const secretPatterns = [
  { label: 'PEM 私钥', regex: /-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----/ },
  { label: 'GitHub Token', regex: /\b(?:gh[pousr]_[A-Za-z0-9_]{30,}|github_pat_[A-Za-z0-9_]{40,})\b/ },
  { label: 'AWS Access Key', regex: /\bAKIA[0-9A-Z]{16}\b/ },
  { label: '疑似 OpenAI/API Key', regex: /\bsk-[A-Za-z0-9_-]{32,}\b/ },
]
const textExt = new Set(['.go','.ts','.tsx','.js','.mjs','.cjs','.vue','.json','.yaml','.yml','.toml','.ini','.conf','.ps1','.sh','.bat','.md','.txt','.iss'])
for (const rel of unique) {
  if (rel === 'scripts/common/check-github-safety.mjs') continue
  const full = path.join(root, rel)
  if (!fs.existsSync(full) || !fs.statSync(full).isFile()) continue
  if (!textExt.has(path.extname(rel).toLowerCase())) continue
  if (fs.statSync(full).size > 2 * 1024 * 1024) continue
  const text = fs.readFileSync(full, 'utf8')
  for (const { label, regex } of secretPatterns) if (regex.test(text)) failures.push(`疑似 ${label} 出现在：${rel}`)
}

if (warnings.length) for (const item of warnings) console.warn(`[WARN] ${item}`)
if (failures.length) {
  console.error(`AGMP GitHub Safety Gate FAIL (${failures.length})`)
  for (const item of failures) console.error(` - ${item}`)
  process.exit(1)
}
console.log(`AGMP GitHub Safety Gate PASS (${unique.length} candidate files checked)`)
