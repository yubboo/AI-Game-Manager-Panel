import fs from 'node:fs'
import { pathToFileURL } from 'node:url'

const reply = (value) => process.stdout.write(JSON.stringify(value))
const fail = (message) => {
  reply({ ok: false, error: String(message || 'unknown DSH bridge error') })
  process.exitCode = 1
}

try {
  const input = fs.readFileSync(0, 'utf8')
  const request = JSON.parse(input || '{}')
  if (!request.entry) throw new Error('missing plugin entry')

  const mod = await import(pathToFileURL(request.entry).href)
  const plugin = mod.default ?? mod
  const inject = Array.isArray(plugin?.inject) ? plugin.inject : (Array.isArray(mod.inject) ? mod.inject : [])
  const unsupported = inject.filter((name) => name !== 'tools')
  if (unsupported.length) {
    throw new Error(`unsupported DSH services: ${unsupported.join(', ')}`)
  }

  const tools = []
  const cleanups = []
  const listeners = new Map()
  const toolRuntime = {
    register(tool) {
      if (!tool || typeof tool !== 'object') throw new Error('ctx.tools.register received an invalid tool')
      tools.push(tool)
      return () => {
        const index = tools.indexOf(tool)
        if (index >= 0) tools.splice(index, 1)
      }
    },
  }
  const ctx = {
    tools: toolRuntime,
    get(name) { return name === 'tools' ? toolRuntime : undefined },
    effect(effect) {
      const cleanup = effect()
      if (typeof cleanup === 'function') cleanups.push(cleanup)
      return cleanup
    },
    on(name, handler) {
      const bucket = listeners.get(name) ?? []
      bucket.push(handler)
      listeners.set(name, bucket)
      return () => {
        const next = (listeners.get(name) ?? []).filter((item) => item !== handler)
        listeners.set(name, next)
      }
    },
    emit(name, payload) {
      for (const handler of listeners.get(name) ?? []) handler(payload)
    },
  }

  const apply = typeof plugin === 'function' ? plugin : (plugin?.apply ?? mod.apply)
  if (typeof apply !== 'function') throw new Error('DSH plugin does not export apply(ctx)')
  const returned = await apply(ctx, request.config ?? {})
  if (typeof returned === 'function') cleanups.push(returned)

  const normalizeParameters = (tool) => {
    const raw = tool.parameters ?? tool.inputSchema ?? tool.input_schema ?? {}
    if (!raw || typeof raw !== 'object' || Array.isArray(raw)) return {}
    // Raw JSON-Schema ToolDefinitions (including MCP-sourced tools) already
    // carry an object schema. DeepSeek Harness defineTool() instead uses a
    // field map such as {name:{type:'string', required:true}}. Normalize both
    // into the JSON-Schema subset enforced by AGMP Host.
    if (raw.type === 'object' || raw.properties || raw.$schema) return raw
    const properties = {}
    const required = []
    for (const [name, value] of Object.entries(raw)) {
      if (!value || typeof value !== 'object' || Array.isArray(value)) continue
      const copy = { ...value }
      if (copy.required === true) required.push(name)
      delete copy.required
      properties[name] = copy
    }
    return { type: 'object', properties, required, additionalProperties: false }
  }

  const serialize = (tool) => ({
    name: String(tool.name ?? ''),
    description: String(tool.description ?? ''),
    parameters: normalizeParameters(tool),
  })

  let result
  if (request.action === 'list') {
    result = { tools: tools.map(serialize) }
  } else if (request.action === 'call') {
    const tool = tools.find((item) => item.name === request.tool)
    if (!tool) throw new Error(`DSH tool not found: ${request.tool}`)
    const execute = tool.execute ?? tool.run
    if (typeof execute !== 'function') throw new Error(`DSH tool has no execute(): ${request.tool}`)
    const value = await execute(request.arguments ?? {})
    result = { value }
  } else {
    throw new Error(`unknown bridge action: ${request.action}`)
  }

  for (const cleanup of cleanups.reverse()) await cleanup()
  reply({ ok: true, result })
} catch (error) {
  fail(error instanceof Error ? error.message : error)
}
