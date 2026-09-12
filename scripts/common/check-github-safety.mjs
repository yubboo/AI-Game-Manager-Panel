import fs from 'node:fs'
import path from 'node:path'
import { execFileSync } from 'node:child_process'

const root = process.cwd()
const failures = []
const fallbackSkip = new Set(['.git', 'node_modules', '.tmp-test', 'dist', 'target', 'runtime', 'build', '.pnpm-store'])
const secret = /(sk-[A-Za-z0-9_-]{20,}|ghp_[A-Za-z0-9]{20,}|BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY|x-api-key\s*[:=]\s*["'][^"']{12,})/i

function normalize(relative) {
  return relative.replaceAll('\\', '/').replace(/^\.\//, '')
}

function gitCandidates() {
  if (!fs.existsSync(path.join(root, '.git'))) return null
  try {
    const output = execFileSync('git', ['ls-files', '--cached', '--others', '--exclude-standard', '-z'], { cwd: root, encoding: 'buffer', stdio: ['ignore', 'pipe', 'ignore'] })
    return output.toString('utf8').split('\0').map(normalize).filter(Boolean)
  } catch {
    return null
  }
}

function fallbackCandidates() {
  const files = []
  function walk(directory) {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      if (fallbackSkip.has(entry.name)) continue
      const absolute = path.join(directory, entry.name)
      if (entry.isDirectory()) walk(absolute)
      else if (entry.isFile()) files.push(normalize(path.relative(root, absolute)))
    }
  }
  walk(root)
  return files
}

function looksBinary(buffer) {
  const sample = buffer.subarray(0, Math.min(buffer.length, 8192))
  if (sample.includes(0)) return true
  if (!sample.length) return false
  let controls = 0
  for (const byte of sample) {
    if (byte < 0x09 || (byte > 0x0d && byte < 0x20)) controls += 1
  }
  return controls / sample.length > 0.08
}

const candidates = [...new Set(gitCandidates() ?? fallbackCandidates())]
for (const relative of candidates) {
  const absolute = path.join(root, relative)
  let stat
  try { stat = fs.statSync(absolute) } catch { continue }
  if (!stat.isFile()) continue
  if (stat.size > 25 * 1024 * 1024) failures.push(`large file ${relative}: ${stat.size}`)
  if (stat.size >= 2 * 1024 * 1024 || relative.endsWith('.test.ts')) continue
  let buffer
  try { buffer = fs.readFileSync(absolute) } catch { continue }
  if (looksBinary(buffer)) continue
  if (secret.test(buffer.toString('utf8'))) failures.push(`possible secret: ${relative}`)
}

if (failures.length) {
  console.error('AGMP GitHub Safety Gate FAIL')
  for (const failure of failures) console.error(` - ${failure}`)
  process.exit(1)
}
console.log(`AGMP GitHub Safety Gate PASS (${candidates.length} Git-trackable source files checked; ignored build/runtime artifacts skipped)`)
