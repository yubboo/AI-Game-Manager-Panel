import type { AgentEvent, AgentEventType } from '../../protocol/src/index.ts'
export class EventBus {
  #sequence = 0
  #events: AgentEvent[] = []
  #subscribers = new Set<(event: AgentEvent) => void>()
  publish(input: Omit<AgentEvent, 'seq' | 'at'> & { at?: string }): AgentEvent {
    const event: AgentEvent = { ...input, seq: ++this.#sequence, at: input.at ?? new Date().toISOString() }
    this.#events.push(event)
    if (this.#events.length > 5000) this.#events.splice(0, this.#events.length - 5000)
    for (const subscriber of this.#subscribers) subscriber(event)
    return event
  }
  after(sequence: number, limit = 100): AgentEvent[] { return this.#events.filter(item => item.seq > sequence).slice(0, Math.max(1, Math.min(limit, 1000))).map(item => structuredClone(item)) }
  subscribe(subscriber: (event: AgentEvent) => void): () => void { this.#subscribers.add(subscriber); return () => this.#subscribers.delete(subscriber) }
  emit(runId: string, type: AgentEventType, summary: string, data?: unknown) { return this.publish({ runId, type, summary, ...(data === undefined ? {} : { data }) }) }
}
