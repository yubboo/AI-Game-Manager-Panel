import { appendFile, mkdir } from 'node:fs/promises'
import path from 'node:path'
import type { AgentEvent, AgentMessage } from '../../protocol/src/index.ts'
export class JsonlSessionStore {
  constructor(private readonly root: string) {}
  async appendMessage(runId: string, message: AgentMessage): Promise<void> { await this.#append(runId, { kind: 'message', at: new Date().toISOString(), message }) }
  async appendEvent(runId: string, event: AgentEvent): Promise<void> { await this.#append(runId, { kind: 'event', at: event.at, event }) }
  async #append(runId: string, value: unknown) { await mkdir(this.root, { recursive: true }); await appendFile(path.join(this.root, `${runId}.jsonl`), `${JSON.stringify(value)}\n`, 'utf8') }
}
