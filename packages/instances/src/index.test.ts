import test from 'node:test'
import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import { GameInstanceStore } from './index.ts'

test('GameInstance persists lifecycle state and remains the shared visual/agent object', async () => {
  const root = await mkdtemp(path.join(os.tmpdir(), 'agmp-instance-'))
  try {
    const store = new GameInstanceStore(root)
    const created = await store.create({ gameId: 'minecraft.java', name: 'Fabric Friends', software: 'fabric', version: '1.21.1', serverDir: 'servers/friends', port: 25565, memoryMb: 2048, state: 'planned', loader: '0.16.14', mods: [] })
    const running = await store.patch(created.id, { state: 'running', jobId: 'JOB-1', endpoint: '127.0.0.1:25565', lastVerifiedAt: new Date().toISOString() })
    assert.equal(running.state, 'running')
    assert.equal((await new GameInstanceStore(root).get(created.id)).jobId, 'JOB-1')
  } finally { await rm(root, { recursive: true, force: true }) }
})
