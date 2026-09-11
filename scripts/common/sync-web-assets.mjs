import { cp, mkdir, rm, stat } from 'node:fs/promises'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..', '..')
const source = resolve(root, 'frontend', 'dist')
const target = resolve(root, 'cmd', 'aigame-manager-web', 'web')

try {
  const info = await stat(source)
  if (!info.isDirectory()) throw new Error('frontend/dist is not a directory')
} catch {
  console.error('[ERROR] frontend/dist does not exist. Build the frontend first.')
  process.exit(1)
}

await rm(target, { recursive: true, force: true })
await mkdir(target, { recursive: true })
await cp(source, target, { recursive: true })
console.log(`[OK] Web assets synced: ${target}`)
