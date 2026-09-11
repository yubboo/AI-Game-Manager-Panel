import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..', '..')
const failures = []

// Renames must remove the old path. These legacy names caused duplicate Go test
// declarations after a source-package sync in 0.2.6.
const forbiddenLegacyFiles = [
  'internal/bridge/httpapi/server_xiaoyu_models_test.go',
  'internal/bridge/httpapi/server_xiaoyu_models_release_test.go',
]

for (const rel of forbiddenLegacyFiles) {
  if (fs.existsSync(path.join(root, rel))) failures.push(`旧重命名文件仍存在：${rel}`)
}

function walk(dir, result = []) {
  if (!fs.existsSync(dir)) return result
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) walk(full, result)
    else result.push(full)
  }
  return result
}

// Exact duplicate Go test files in the same package are almost always a stale
// rename and can redeclare the same test functions. Scope this narrowly so
// deliberate duplicated fixtures in unrelated packages are not rejected.
const internal = path.join(root, 'internal')
const tests = walk(internal).filter((file) => file.endsWith('_test.go'))
const groups = new Map()
for (const file of tests) {
  const bytes = fs.readFileSync(file)
  const hash = crypto.createHash('sha256').update(bytes).digest('hex')
  const key = `${path.dirname(file)}\0${hash}`
  const list = groups.get(key) ?? []
  list.push(file)
  groups.set(key, list)
}

for (const files of groups.values()) {
  if (files.length < 2) continue
  failures.push(`同一 Go package 存在内容完全相同的测试文件：${files.map((f) => path.relative(root, f).replaceAll('\\', '/')).join(', ')}`)
}

if (failures.length) {
  console.error(`AGMP Duplicate Source Gate FAIL (${failures.length})`)
  for (const item of failures) console.error(` - ${item}`)
  process.exit(1)
}

console.log('AGMP Duplicate Source Gate PASS')
