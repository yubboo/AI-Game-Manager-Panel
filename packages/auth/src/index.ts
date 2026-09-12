import { createHash, randomBytes, randomUUID, timingSafeEqual } from 'node:crypto'
import { mkdir, readFile, writeFile } from 'node:fs/promises'
import path from 'node:path'
export interface AuthUser { id: string; username: string; displayName: string; role: 'owner' | 'manager' | 'member' }
type StoredUser = AuthUser & { passwordHash: string; salt: string }
export class LocalAuthService {
  #file: string; #sessions = new Map<string, { user: AuthUser; expiresAt: number }>()
  constructor(root: string) { this.#file = path.join(root, 'auth', 'users.json') }
  async bootstrapStatus() { const users = await this.#users(); return { initialized: users.length > 0 } }
  async createOwner(username: string, password: string, displayName = 'Owner') { const users = await this.#users(); if (users.length) throw new Error('owner already exists'); const user = this.#make(username, password, displayName, 'owner'); await this.#save([user]); return this.#public(user) }
  async login(username: string, password: string) { const users = await this.#users(); const user = users.find(item => item.username === username.trim()); if (!user || !this.#verify(password, user)) throw new Error('用户名或密码错误'); const token = randomBytes(32).toString('hex'); this.#sessions.set(token, { user: this.#public(user), expiresAt: Date.now() + 7 * 86400000 }); return { token, user: this.#public(user) } }
  session(token: string): AuthUser | null { const item = this.#sessions.get(token); if (!item || item.expiresAt < Date.now()) { if (item) this.#sessions.delete(token); return null } return structuredClone(item.user) }
  #make(username: string, password: string, displayName: string, role: StoredUser['role']): StoredUser { if (password.length < 8) throw new Error('密码至少 8 位'); const salt = randomBytes(16).toString('hex'); return { id: randomUUID(), username: username.trim(), displayName: displayName.trim() || username.trim(), role, salt, passwordHash: this.#hash(password, salt) } }
  #hash(password: string, salt: string) { return createHash('sha256').update(`${salt}\n${password}`).digest('hex') }
  #verify(password: string, user: StoredUser) { const a = Buffer.from(this.#hash(password, user.salt), 'hex'); const b = Buffer.from(user.passwordHash, 'hex'); return a.length === b.length && timingSafeEqual(a, b) }
  #public(user: StoredUser): AuthUser { const { passwordHash: _p, salt: _s, ...publicUser } = user; return publicUser }
  async #users(): Promise<StoredUser[]> { try { return JSON.parse(await readFile(this.#file, 'utf8')) as StoredUser[] } catch { return [] } }
  async #save(users: StoredUser[]) { await mkdir(path.dirname(this.#file), { recursive: true }); await writeFile(this.#file, JSON.stringify(users, null, 2), 'utf8') }
}
