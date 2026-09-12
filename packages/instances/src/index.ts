import { randomUUID } from 'node:crypto'
import { mkdir, readFile, rename, writeFile } from 'node:fs/promises'
import path from 'node:path'

export type GameInstanceState = 'planned' | 'installing' | 'configured' | 'starting' | 'running' | 'stopped' | 'failed'
export interface GameInstanceMod { source: string; slug: string; versionId: string; fileName: string; sha1?: string; sha512?: string }
export interface GameInstance {
  id: string
  gameId: string
  name: string
  software: string
  version: string
  serverDir: string
  port: number
  memoryMb: number
  state: GameInstanceState
  javaMajor?: number
  javaExecutable?: string
  jobId?: string
  endpoint?: string
  loader?: string
  build?: number
  mods?: GameInstanceMod[]
  lastVerifiedAt?: string
  lastError?: string
  createdAt: string
  updatedAt: string
}

export class GameInstanceStore {
  #file: string
  constructor(root: string) { this.#file = path.join(root, 'instances', 'instances.json') }
  async list(): Promise<GameInstance[]> { return this.#read() }
  async get(id: string): Promise<GameInstance> {
    const item = (await this.#read()).find(value => value.id === id)
    if (!item) throw new Error('GameInstance 不存在')
    return item
  }
  async create(input: Omit<GameInstance, 'id' | 'createdAt' | 'updatedAt'>): Promise<GameInstance> {
    const now = new Date().toISOString()
    const item: GameInstance = { ...structuredClone(input), id: `inst_${randomUUID()}`, createdAt: now, updatedAt: now }
    const all = await this.#read(); all.push(item); await this.#save(all); return item
  }
  async patch(id: string, patch: Partial<Omit<GameInstance, 'id' | 'createdAt'>>): Promise<GameInstance> {
    const all = await this.#read(); const index = all.findIndex(item => item.id === id)
    if (index < 0) throw new Error('GameInstance 不存在')
    const current = all[index]!
    const next: GameInstance = { ...current, ...structuredClone(patch), id: current.id, createdAt: current.createdAt, updatedAt: new Date().toISOString() }
    all[index] = next; await this.#save(all); return next
  }
  async #read(): Promise<GameInstance[]> {
    try { const value = JSON.parse(await readFile(this.#file, 'utf8')); return Array.isArray(value) ? value as GameInstance[] : [] } catch { return [] }
  }
  async #save(all: GameInstance[]): Promise<void> {
    await mkdir(path.dirname(this.#file), { recursive: true })
    const temp = `${this.#file}.tmp`
    await writeFile(temp, JSON.stringify(all, null, 2), 'utf8')
    await rename(temp, this.#file)
  }
}
