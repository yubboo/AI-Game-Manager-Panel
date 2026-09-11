import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const here = path.dirname(fileURLToPath(import.meta.url))
const root = path.resolve(here, '..', '..')
const failures = []

const requiredFiles = [
  '.github/workflows/safety.yml',
  'go.mod',
  'go.sum',
  'main.go',
  'wails.json',
  'frontend/package.json',
  'frontend/src/app/router.ts',
  'frontend/src/features/xiaoyu/AIWorkbenchView.vue',
  'internal/xiaoyu/host/loop.go',
  'internal/xiaoyu/host/brain_model.go',
  'internal/app/app_xiaoyu_tools.go',
  'runtime/README.md',
  'rust/Cargo.toml',
  'rust/crates/xiaoyu-core/Cargo.toml',
  'rust/crates/xiaoyu-core/src/lib.rs',
  'rust/crates/xiaoyu-protocol/Cargo.toml',
  'scripts/common/check-github-safety.mjs',
  'scripts/common/check-project-layout.mjs',
  'scripts/common/check-xiaoyu-harness.mjs',
  'scripts/common/check-xiaoyu-agent-runtime.mjs',
  'scripts/windows/AIGameManagerPanel.ps1',
  'scripts/windows/tasks/Tasks.ps1',
  'docs/NAMING-CONVENTIONS.md',
  'AGMP-GitHub.bat',
  'AI-Game-Manager-Panel.bat',
  'push-agmp.ps1',
  'sync-agmp.ps1',
  'AGMP-Sync.bat',
]

for (const rel of requiredFiles) {
  if (!fs.existsSync(path.join(root, rel))) failures.push(`缺少关键源码：${rel}`)
}

const requiredTrees = [
  ['scripts/common', 15],
  ['scripts/windows', 8],
  ['rust/crates', 5],
  ['internal/xiaoyu', 20],
  ['frontend/src', 25],
]

function countFiles(dir) {
  let count = 0
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) count += countFiles(full)
    else count++
  }
  return count
}

for (const [rel, minimum] of requiredTrees) {
  const full = path.join(root, rel)
  if (!fs.existsSync(full) || !fs.statSync(full).isDirectory()) {
    failures.push(`缺少关键源码目录：${rel}`)
    continue
  }
  const count = countFiles(full)
  if (count < minimum) failures.push(`关键源码目录文件数异常：${rel} (${count} < ${minimum})`)
}

if (failures.length) {
  console.error(`AGMP Source Tree Gate FAIL (${failures.length})`)
  for (const item of failures) console.error(` - ${item}`)
  process.exit(1)
}

console.log('AGMP Source Tree Gate PASS')
