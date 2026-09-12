import type { AgentMessage, ModelTurn, ToolSpec } from '../../protocol/src/index.ts'
export interface ModelGenerateInput { system: string; messages: AgentMessage[]; tools: ToolSpec[]; signal?: AbortSignal }
export interface ModelProvider { id: string; model: string; protocol: string; generate(input: ModelGenerateInput): Promise<ModelTurn> }
