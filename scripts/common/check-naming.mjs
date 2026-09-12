import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..', '..')
const failures = []
const warnings = []
const skipDirs = new Set(['.git', 'node_modules', 'target', 'build', 'dist', '.pnpm-store'])
const localRootDirs = new Set(['bin', '.vscode', '.idea', 'data', 'log', 'logs', 'backups', 'instances', 'temp', 'exports', 'plugins', 'cache'])

function shouldSkipDir(rel, name) {
  if (skipDirs.has(name)) return true
  if (localRootDirs.has(rel)) return true
  // runtime/README.md belongs to source; every runtime subdirectory is local/generated state.
  if (rel.startsWith('runtime/')) return true
  return false
}

function walk(dir, prefix = '') {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const rel = path.posix.join(prefix, entry.name)
    if (entry.isDirectory()) {
      if (shouldSkipDir(rel, entry.name)) continue
      walk(path.join(dir, entry.name), rel)
      continue
    }

    const filenameLimit = /_test\.[^.]+$/i.test(entry.name) ? 48 : 40
    if (entry.name.length > filenameLimit) {
      failures.push(`文件名过长 (${entry.name.length} > ${filenameLimit})：${rel}`)
    }
    if (rel.length > 220) {
      failures.push(`相对路径过长 (${rel.length} > 220)：${rel}`)
    } else if (rel.length > 180) {
      warnings.push(`相对路径偏长 (${rel.length} > 180)：${rel}`)
    }
  }
}

walk(root)
for (const item of warnings) console.warn(`[WARN] ${item}`)
if (failures.length) {
  console.error(`AGMP Naming Gate FAIL (${failures.length})`)
  for (const item of failures) console.error(` - ${item}`)
  process.exit(1)
}
console.log(`AGMP Naming Gate PASS${warnings.length ? ` (${warnings.length} warning)` : ''}`)
