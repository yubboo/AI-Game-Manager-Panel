import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const p = rel => path.join(root, rel)
const read = rel => fs.readFileSync(p(rel), 'utf8').replace(/^\uFEFF/, '')
const need = rel => { if (!fs.existsSync(p(rel))) failures.push(`缺少语言职责文件：${rel}`) }
const has = (rel, token) => {
  if (!read(rel).includes(token)) failures.push(`${rel} 缺少语言职责规则：${token}`)
}

for (const rel of [
  'docs/architecture/LANGUAGE-OWNERSHIP.md',
  'docs/PROJECT-ARCHITECTURE.md',
  'docs/architecture/XIAOYU-CORE.md',
  'AGENTS.md',
  'rust/crates/xiaoyu-core/src/lib.rs',
  'rust/crates/xiaoyu-core/src/tool_search.rs',
  'rust/crates/xiaoyu-protocol/src/lib.rs',
  'internal/xiaoyu/runtime/service.go',
]) need(rel)

try {
  for (const token of ['Rust = XiaoYu Agent Runtime', 'Go = AGMP Product Host / Game Domain Services', 'Vue / TypeScript']) {
    has('docs/architecture/LANGUAGE-OWNERSHIP.md', token)
  }
  for (const token of ['Rust-first Agent Runtime', '0.2.9', 'Wails']) has('docs/PROJECT-ARCHITECTURE.md', token)
  for (const token of ['rust-agent-runtime-boundary', 'tool-search-v1', 'native-runtime-migration', 'search_tools']) {
    has('rust/crates/xiaoyu-core/src/lib.rs', token)
  }
  for (const token of ['ToolSearchRequest', 'ToolSearchHit', 'ToolSearchResponse']) {
    has('rust/crates/xiaoyu-protocol/src/lib.rs', token)
  }
  has('rust/crates/xiaoyu-core/src/tool_search.rs', 'Approval, RBAC, sandbox')
  has('internal/xiaoyu/runtime/service.go', 'SearchTools')
  has('AGENTS.md', 'LANGUAGE-OWNERSHIP.md')

  const active = [
    read('AGENTS.md'),
    read('docs/PROJECT-ARCHITECTURE.md'),
    read('docs/architecture/XIAOYU-CORE.md'),
    read('docs/development/PROJECT-RULES.md'),
    read('docs/development/MODULES.md'),
  ].join('\n')
  if (/Rust `xiaoyu-core` 是 \*\*Brain-only\*\*/.test(active)) {
    failures.push('现役规范仍把 Rust xiaoyu-core 定义为 Brain-only')
  }
} catch (error) {
  failures.push(error instanceof Error ? error.message : String(error))
}

if (failures.length) {
  console.error('AGMP Language Ownership Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}

console.log('AGMP Language Ownership Gate PASS (Rust Agent Runtime · Go Domain Host · Vue UI)')
