export type JsonPrimitive = string | number | boolean | null
export type JsonValue = JsonPrimitive | JsonValue[] | JsonObject
export interface JsonObject { [key: string]: JsonValue | undefined }

export type ToolRisk = 'read' | 'operate' | 'modify' | 'destructive' | 'system'
export type ToolConfirm = 'policy' | 'always'
export interface ToolSpec {
  name: string
  description: string
  inputSchema: Record<string, unknown>
  risk: ToolRisk
  source: string
  confirm?: ToolConfirm
}
export interface ToolCall { id: string; name: string; arguments: JsonObject }
export interface ToolAuthorization { requestHash: string }
export interface ToolObservation {
  callId: string
  name: string
  ok: boolean
  content: string
  data?: unknown
  durationMs: number
}
export interface ModelUsage { inputTokens: number; outputTokens: number; reasoningTokens: number; totalTokens: number }
export interface AgentMessage {
  role: 'system' | 'user' | 'assistant' | 'tool'
  content: string
  toolCalls?: ToolCall[]
  toolCallId?: string
  name?: string
}
export type AgentEventType = 'run-created' | 'run-started' | 'model-started' | 'model-finished' | 'tool-started' | 'tool-progress' | 'tool-finished' | 'approval-required' | 'approval-resolved' | 'run-completed' | 'run-failed' | 'run-cancelled'
export interface AgentEvent { seq: number; runId: string; type: AgentEventType; summary: string; at: string; data?: unknown }
export interface ModelTurn {
  text: string
  toolCalls: ToolCall[]
  usage: ModelUsage
  provider?: string
  model?: string
  protocol?: string
}
export interface NativeCapabilityLease { id: string; scope: string; runId: string; tool: string; expiresAt: number; maxUses: number }
