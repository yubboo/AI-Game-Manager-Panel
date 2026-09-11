import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'

const root = process.cwd()
const failures = []
const read = rel => fs.readFileSync(path.join(root, rel), 'utf8').replace(/^\uFEFF/, '')
const need = rel => {
  const full = path.join(root, rel)
  if (!fs.existsSync(full)) failures.push(`缺少依赖锁文件：${rel}`)
  return full
}

const files = [
  'go.mod',
  'go.sum',
  'rust/Cargo.lock',
  'frontend/pnpm-lock.yaml',
  'desktop/electron/pnpm-lock.yaml',
]
files.forEach(need)

try {
  const goMod = read('go.mod')
  const goSum = read('go.sum')
  const cargoToml = read('rust/Cargo.toml')
  const cargoLock = read('rust/Cargo.lock')
  const frontPkg = JSON.parse(read('frontend/package.json'))
  const frontLock = read('frontend/pnpm-lock.yaml')
  const electronPkg = JSON.parse(read('desktop/electron/package.json'))
  const electronLock = read('desktop/electron/pnpm-lock.yaml')
  const workflow = read('.github/workflows/safety.yml')
  const gitignore = read('.gitignore')

  if (!goMod.includes('go 1.25.0')) failures.push('go.mod 必须冻结 Go 1.25.0 基线')
  if (goSum.length < 5000 || !goSum.includes('github.com/leaanthony/slicer') || !goSum.includes('golang.org/x/sys')) {
    failures.push('go.sum 看起来不是 GitHub Runner 生成的完整依赖图')
  }

  const version = cargoToml.match(/^version\s*=\s*"([^"]+)"/m)?.[1]
  if (!version) failures.push('无法读取 Rust workspace version')
  if (version && !cargoLock.includes(`name = "xiaoyu-core"\nversion = "${version}"`)) failures.push('Cargo.lock 中 xiaoyu-core 版本与 workspace 不一致')
  if (version && !cargoLock.includes(`name = "xiaoyu-protocol"\nversion = "${version}"`)) failures.push('Cargo.lock 中 xiaoyu-protocol 版本与 workspace 不一致')

  for (const [name, spec] of Object.entries({...frontPkg.dependencies, ...frontPkg.devDependencies})) {
    if (!frontLock.includes(`specifier: ${spec}`)) failures.push(`Frontend lock 缺少 ${name}@${spec}`)
  }
  for (const [name, spec] of Object.entries(electronPkg.devDependencies ?? {})) {
    if (!electronLock.includes(`specifier: ${spec}`)) failures.push(`Electron lock 缺少 ${name}@${spec}`)
  }
  if (!frontLock.includes("lockfileVersion: '9.0'")) failures.push('Frontend pnpm lockfileVersion 必须为 9.0')
  if (!electronLock.includes("lockfileVersion: '9.0'")) failures.push('Electron pnpm lockfileVersion 必须为 9.0')

  for (const token of ['cargo check --manifest-path rust/Cargo.toml --workspace --locked', 'cargo test --manifest-path rust/Cargo.toml --workspace --locked', 'pnpm install --frozen-lockfile', 'go mod verify']) {
    if (!workflow.includes(token)) failures.push(`GitHub Actions 尚未冻结依赖：${token}`)
  }
  if (workflow.includes('--no-frozen-lockfile')) failures.push('GitHub Actions 禁止继续使用 --no-frozen-lockfile')
  if (/^Cargo\.lock$/m.test(gitignore) || /pnpm-lock\.yaml/.test(gitignore)) failures.push('.gitignore 不得忽略项目锁文件')
} catch (error) {
  failures.push(error instanceof Error ? error.message : String(error))
}

if (failures.length) {
  console.error('AGMP Dependency Lock Gate FAIL')
  failures.forEach(item => console.error(` - ${item}`))
  process.exit(1)
}

console.log('AGMP Dependency Lock Gate PASS (Go modules · Cargo --locked · pnpm frozen)')
