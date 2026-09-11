import type { DSTProcessLogLine } from '../../../../shared/types/backend'

export type DSTConsoleCategory = 'system' | 'steam' | 'auth' | 'world' | 'player' | 'mod' | 'save' | 'warning' | 'error'

export interface DSTConsoleDisplayLine extends DSTProcessLogLine {
  category: DSTConsoleCategory
  label: string
  title: string
  detail: string
  clock: string
  important: boolean
}

const categoryLabels: Record<DSTConsoleCategory, string> = {
  system: '系统',
  steam: 'Steam',
  auth: '认证',
  world: '世界',
  player: '玩家',
  mod: 'Mod',
  save: '存档',
  warning: '警告',
  error: '错误',
}

export function classifyDSTConsoleLine(line: DSTProcessLogLine): DSTConsoleDisplayLine {
  const raw = line.text ?? ''
  const stripped = stripPrefix(raw)
  const lower = stripped.toLowerCase()
  let category: DSTConsoleCategory = 'system'
  let title = stripped || '空日志行'
  let detail = ''
  let important = false

  if (lower.includes('could not confirm port') && lower.includes('firewall')) {
    // DST emits this even when the UDP socket binds successfully moments later.
    // Keep it as a neutral hint so a healthy server is not painted as failed.
    category = 'system'
  } else if (containsAny(lower, ['onloadpermissionlist:', 'onloaduseridlist:']) && lower.includes('(failure)')) {
    category = 'system'
  } else if (containsAny(lower, ['[error]', 'e_invalid_token', 'your server will not start', 'attempt to call a nil value', 'fatal'])) {
    category = 'error'
    important = true
  } else if (containsAny(lower, ['[warning]', '[warn]', 'warning:', ' failed ', 'failure)'])) {
    category = 'warning'
    important = true
  } else if (containsAny(lower, ['join announcement', 'client authenticated:', 'client connected from', 'new incoming connection', 'spawn request:', 'client disconnected', 'player disconnected', 'lost connection'])) {
    category = 'player'
    important = true
  } else if (containsAny(lower, ['token retrieved from', 'account communication success', 'received (ku_', 'tokenpurpose', 'server registered via geo dns'])) {
    category = 'auth'
    important = true
  } else if (containsAny(lower, ['[steam]', 'steamgameserver_init', 'authenticated host'])) {
    category = 'steam'
    important = true
  } else if (containsAny(lower, ['world generated on build', 'loading world:', 'begin session:', 'online server started on port:', '[shard] starting', '[shard] shard server started', 'sim paused', 'sim unpaused', 'about to start a server', 'about to start a shard', 'secondary shard is now ready', 'secondary caves', ' is now connected'])) {
    category = 'world'
    important = true
  } else if (containsAny(lower, ['modindex:', 'registering mods', 'loaded modoverrides.lua', 'workshop'])) {
    category = 'mod'
  } else if (containsAny(lower, ['saving to ', 'serializing user:', 'save file is at version', 'uploads added to server'])) {
    category = 'save'
  }

  const friendly = friendlyMessage(stripped, category)
  if (friendly) {
    title = friendly.title
    detail = friendly.detail
  }

  return {
    ...line,
    category,
    label: categoryLabels[category],
    title,
    detail,
    clock: extractClock(raw),
    important,
  }
}

function friendlyMessage(text: string, category: DSTConsoleCategory): { title: string; detail: string } | null {
  let match: RegExpMatchArray | null
  if (/SteamGameServer_Init success/i.test(text)) return { title: 'Steam 服务器接口初始化成功', detail: text }
  if (/Could not confirm port .*firewall/i.test(text)) return { title: 'Windows 防火墙状态未确认', detail: '这是 DST 的探测提示，不等于端口绑定失败；若后续出现“在线服务器已启动/Shard 已就绪”，AGMP 会自动视为已解决。' }
  if (/Token retrieved from:/i.test(text)) return { title: 'Klei 服务器令牌已读取', detail: 'Token 正文不会在 AGMP 中显示。' }
  if (/Account Communication Success/i.test(text)) return { title: 'Klei 身份认证成功', detail: text }
  if ((match = text.match(/Online Server Started on port:\s*(\d+)/i))) return { title: '在线服务器已启动', detail: `游戏端口 ${match[1]}` }
  if ((match = text.match(/Shard server started on port:\s*(\d+)/i))) return { title: 'Shard 服务已启动', detail: `Shard 端口 ${match[1]}` }
  if ((match = text.match(/Server registered via geo DNS in\s+(.+)/i))) return { title: '服务器已完成区域注册', detail: match[1]!.trim() }
  if ((match = text.match(/Client authenticated:\s*\([^)]*\)\s*(.+)$/i))) return { title: '玩家认证成功', detail: match[1]!.trim() }
  if ((match = text.match(/\[Join Announcement\]\s*(.+)$/i))) return { title: '玩家进入服务器', detail: match[1]!.trim() }
  if ((match = text.match(/Spawn request:\s*([^\s]+)\s+from\s+(.+)$/i))) return { title: '玩家角色生成', detail: `${match[2]!.trim()} · ${match[1]}` }
  if (/secondary shard LUA is now ready!/i.test(text)) return { title: '洞穴 Shard 已就绪', detail: text }
  if (/Secondary (?:shard )?Caves.*ready!/i.test(text) || /secondary shard is now ready!/i.test(text)) return { title: '洞穴 Shard 已就绪', detail: text }
  if (/World .*\(Caves\) is now connected/i.test(text) || /World 1\(Master\) is now connected/i.test(text)) return { title: '地面与洞穴联动成功', detail: text }
  if (/Sim paused/i.test(text)) return { title: '世界已暂停', detail: '通常表示当前没有活跃玩家或世界正在等待。' }
  if (/Sim unpaused/i.test(text)) return { title: '世界已恢复运行', detail: text }
  if ((match = text.match(/Loading world:\s*(.+)$/i))) return { title: '正在加载世界存档', detail: match[1]!.trim() }
  if (/World generated on build/i.test(text)) return { title: '世界数据加载完成', detail: text }
  if (/No auth token could be found/i.test(text)) return { title: '缺少 Klei 服务器令牌', detail: text }
  if (/E_INVALID_TOKEN/i.test(text)) return { title: 'Klei 服务器令牌无效', detail: text }
  if (/Server failed to start!/i.test(text)) return { title: '服务器网络启动失败', detail: '启动已进入网络绑定阶段但未成功；优先检查 AGMP 的端口状态、残留 Dedicated Server 进程与 Master/Caves 端口配置。' }
  if (category === 'warning') return { title: '服务器警告', detail: text }
  if (category === 'error') return { title: '服务器错误', detail: text }
  return null
}

function stripPrefix(value: string) {
  return value.replace(/^\s*\[\d{2}:\d{2}:\d{2}\]:\s*/, '').trimEnd()
}

function extractClock(value: string) {
  const match = value.match(/^\s*\[(\d{2}:\d{2}:\d{2})\]:/)
  return match?.[1] ?? ''
}

function containsAny(value: string, needles: string[]) {
  return needles.some(needle => value.includes(needle))
}
