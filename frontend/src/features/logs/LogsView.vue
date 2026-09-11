<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppIcon from '../../shared/components/AppIcon.vue'
import { useAppStore } from '../../shared/store/app'
import type { GlobalLogFile } from '../../shared/types/backend'
import { formatBytes, formatLogTime, useLogHub } from './useLogHub'

const app = useAppStore()
const route = useRoute()
const logging = computed(() => app.platformConfig?.logging)
const hub = useLogHub({
  catalogPageSize: () => logging.value?.catalogPageSize || 100,
  readPageSize: () => logging.value?.readPageSize || 600,
  autoRefreshSeconds: () => logging.value?.autoRefreshSeconds || 5,
})

const confirmAction = ref<'' | 'delete-selected' | 'delete-filtered' | 'clear-history'>('')
const confirmTitle = computed(() => {
  if (confirmAction.value === 'delete-selected') return '删除这个历史日志？'
  if (confirmAction.value === 'delete-filtered') return '删除当前筛选结果？'
  if (confirmAction.value === 'clear-history') return '清空全部历史日志？'
  return ''
})
const confirmDetail = computed(() => {
  if (confirmAction.value === 'delete-selected') return '日志文件会从磁盘真实删除。正在写入的活动日志受保护，无法删除。'
  if (confirmAction.value === 'delete-filtered') return `将删除当前筛选命中的 ${hub.catalog.value.total} 个日志中的非活动文件，运行中日志会自动跳过。`
  if (confirmAction.value === 'clear-history') return '将清理当前配置日志目录中全部可删除历史日志；AI Game Manager Panel Core、当前操作审计和运行中游戏日志会保留。'
  return ''
})

const gameOptions = computed(() => app.platformConfig?.games.templates ?? [])
const selectedLabel = computed(() => {
  const item = hub.selected.value
  if (!item) return '尚未选择日志'
  const parts = [item.sourceLabel, item.instanceName, item.shard].filter(Boolean)
  return parts.join(' · ') || item.name
})

function logMeta(item: GlobalLogFile) {
  const parts = [item.instanceName, item.shard, item.status].filter(Boolean)
  return parts.join(' · ') || item.kind
}

function levelText(level: string) {
  const labels: Record<string, string> = { info: 'INFO', warning: 'WARN', error: 'ERROR', debug: 'DEBUG' }
  return labels[level] ?? level.toUpperCase()
}

function categoryText(category: string) {
  const labels: Record<string, string> = {
    general: '常规', system: '系统', operation: '操作', deployment: '部署', steam: 'Steam', network: '网络', auth: '认证', security: '安全', player: '玩家', world: '世界', mod: 'Mod', backup: '备份', ai: 'AI', task: '任务', file: '文件', audit: '审计',
  }
  return labels[category] ?? category
}

function toggleAutoRefresh(event: Event) {
  const target = event.target as HTMLInputElement | null
  hub.setAutoRefresh(Boolean(target?.checked))
}

async function runConfirmed() {
  const action = confirmAction.value
  confirmAction.value = ''
  if (action === 'delete-selected') await hub.deleteSelected()
  else if (action === 'delete-filtered') await hub.deleteFiltered()
  else if (action === 'clear-history') await hub.clearHistory()
}

onMounted(async () => {
  if (!app.platformConfig) await app.loadPlatformConfig()
  if (typeof route.query.source === 'string' && route.query.source) hub.filters.source = route.query.source
  if (typeof route.query.game === 'string' && route.query.game) hub.filters.gameId = route.query.game
  if (typeof route.query.instance === 'string' && route.query.instance) hub.filters.instanceId = route.query.instance
  if (typeof route.query.shard === 'string' && route.query.shard) hub.filters.shard = route.query.shard
  await hub.refreshCatalog(true)
})
</script>

<template>
  <section class="page loghub-page">
    <header class="loghub-heading">
      <div>
        <span class="eyebrow">GLOBAL LOG CENTER</span>
        <h1>全局日志中心</h1>
        <p>统一管理 AI Game Manager Panel Core、Steam、游戏服务器、操作审计、AI 和未来远程节点日志。所有数据均来自当前配置日志目录（0.1.65 新默认 <code>runtime/log/</code>）的真实文件。</p>
      </div>
      <div class="loghub-heading__actions">
        <button class="btn btn--secondary" :disabled="hub.loadingCatalog.value" @click="hub.refreshCatalog(false)"><AppIcon name="logs" />刷新</button>
        <button class="btn btn--secondary" @click="hub.openFolder"><AppIcon name="files" />打开日志目录</button>
        <button class="btn btn--primary" @click="hub.exportFiltered">导出筛选结果</button>
      </div>
    </header>

    <div v-if="hub.error.value" class="notice notice--error">{{ hub.error.value }}</div>
    <div v-if="hub.notice.value" class="notice notice--success">{{ hub.notice.value }}</div>

    <div class="loghub-stats">
      <article><span>日志文件</span><strong>{{ hub.catalog.value.summary.files.toLocaleString() }}</strong><small>真实文件数量</small></article>
      <article><span>真实记录</span><strong>{{ hub.catalog.value.summary.lines.toLocaleString() }}</strong><small>按文件真实行数统计</small></article>
      <article><span>磁盘占用</span><strong>{{ formatBytes(hub.catalog.value.summary.bytes) }}</strong><small>当前筛选范围</small></article>
      <article><span>运行中</span><strong>{{ hub.catalog.value.summary.activeFiles }}</strong><small>受删除保护的日志</small></article>
    </div>

    <section class="panel loghub-filter-panel">
      <div class="loghub-filter-grid">
        <label class="log-field log-field--wide"><span>搜索文件 / 游戏 / 实例</span><input v-model="hub.filters.query" placeholder="例如 Cluster_1、Steam、Master…" @keyup.enter="hub.applyFilters" /></label>
        <label class="log-field"><span>来源</span><select v-model="hub.filters.source"><option value="all">全部来源</option><option value="agmp">AI Game Manager Panel Core</option><option value="operations">操作记录</option><option value="audit">安全审计</option><option value="ai">小鱼</option><option value="steam">Steam</option><option value="dst">饥荒联机版</option><option value="nodes">节点</option></select></label>
        <label class="log-field"><span>类型</span><select v-model="hub.filters.kind"><option value="all">全部类型</option><option value="system">系统日志</option><option value="operation">操作记录</option><option value="audit">安全审计</option><option value="steam">Steam</option><option value="game">游戏日志</option><option value="ai">小鱼日志</option><option value="node">节点日志</option></select></label>
        <label class="log-field"><span>游戏</span><select v-model="hub.filters.gameId"><option value="all">全部游戏</option><option v-for="game in gameOptions" :key="game.id" :value="game.id">{{ game.nameZh }}</option></select></label>
        <label class="log-field"><span>Shard</span><select v-model="hub.filters.shard"><option value="all">全部 Shard</option><option value="Master">Master 地面</option><option value="Caves">Caves 洞穴</option></select></label>
        <label class="log-field"><span>状态</span><select v-model="hub.filters.status"><option value="all">全部状态</option><option value="running">运行中</option><option value="stopped">已停止</option><option value="history">历史</option><option value="completed">已完成</option><option value="failed">失败</option></select></label>
        <label class="log-field"><span>实例 / Cluster</span><input v-model="hub.filters.instanceId" placeholder="可选" /></label>
        <label class="log-field"><span>开始日期</span><input v-model="hub.filters.dateFrom" type="date" /></label>
        <label class="log-field"><span>结束日期</span><input v-model="hub.filters.dateTo" type="date" /></label>
      </div>
      <div class="loghub-filter-actions">
        <label class="auto-refresh"><input type="checkbox" :checked="hub.autoRefresh.value" @change="toggleAutoRefresh" /><span>自动刷新</span><small>{{ logging?.autoRefreshSeconds || 5 }} 秒</small></label>
        <span class="filter-spacer"></span>
        <button class="btn btn--secondary" @click="hub.clearFilters">重置筛选</button>
        <button class="btn btn--primary" @click="hub.applyFilters">应用筛选</button>
        <button class="btn btn--danger-soft" :disabled="hub.catalog.value.total === 0" @click="confirmAction = 'delete-filtered'">删除筛选结果</button>
        <button class="btn btn--danger-soft" @click="confirmAction = 'clear-history'">清空历史日志</button>
      </div>
    </section>

    <section class="loghub-workspace">
      <aside class="panel log-catalog">
        <div class="log-catalog__heading">
          <div><span class="eyebrow">LOG FILES</span><strong>运行记录</strong></div>
          <span>{{ hub.catalog.value.total }} 个</span>
        </div>
        <div class="log-catalog__list">
          <button v-for="item in hub.catalog.value.items" :key="item.id" class="log-file" :class="{ active: hub.selected.value?.id === item.id }" @click="hub.selectLog(item)">
            <span class="log-file__state" :class="{ running: item.active }"></span>
            <span class="log-file__body"><b>{{ item.sourceLabel || item.name }}</b><small>{{ logMeta(item) }}</small><em>{{ formatLogTime(item.modifiedAt) }}</em></span>
            <span class="log-file__numbers"><b>{{ item.lineCount.toLocaleString() }}</b><small>行</small><em>{{ formatBytes(item.byteSize) }}</em></span>
          </button>
          <div v-if="hub.catalog.value.items.length === 0" class="loghub-empty">当前筛选没有日志。手动从文件夹删除日志后，刷新即可同步真实数量。</div>
        </div>
        <footer class="log-catalog__pager">
          <button :disabled="!hub.hasPreviousPage.value" @click="hub.previousPage">上一页</button>
          <span>{{ hub.pageIndex.value }} / {{ hub.pageCount.value }}</span>
          <button :disabled="!hub.hasNextPage.value" @click="hub.nextPage">下一页</button>
        </footer>
      </aside>

      <article class="panel log-viewer">
        <div class="log-viewer__header">
          <div class="log-viewer__identity">
            <span class="eyebrow">LOG VIEWER</span>
            <h2>{{ selectedLabel }}</h2>
            <p v-if="hub.selected.value">{{ hub.selected.value.relativePath }}</p>
          </div>
          <div class="log-viewer__actions">
            <button class="btn btn--secondary" :disabled="!hub.selected.value" @click="hub.exportSelected">下载日志</button>
            <button class="btn btn--danger-soft" :disabled="!hub.selected.value || hub.selected.value.active" @click="confirmAction = 'delete-selected'">删除当前</button>
          </div>
        </div>

        <div v-if="hub.selected.value" class="log-viewer__meta">
          <span><b>{{ hub.selected.value.lineCount.toLocaleString() }}</b>真实记录</span>
          <span><b>{{ hub.lines.value.length.toLocaleString() }}</b>当前显示</span>
          <span><b>{{ hub.matched.value.toLocaleString() }}</b>匹配记录</span>
          <span><b>{{ hub.scanned.value.toLocaleString() }}</b>本次扫描</span>
          <span><b>{{ hub.readDurationMs.value }}</b>ms</span>
          <span v-if="hub.selected.value.active" class="live-chip">● 正在写入</span>
        </div>

        <div class="log-content-toolbar">
          <div class="direction-tabs"><button :class="{ active: hub.direction.value === 'head' }" @click="hub.setDirection('head')">从头看</button><button :class="{ active: hub.direction.value === 'tail' }" @click="hub.setDirection('tail')">从尾部看</button></div>
          <input v-model="hub.contentFilters.query" placeholder="搜索当前日志正文…" @keyup.enter="hub.searchContent" />
          <select v-model="hub.contentFilters.level"><option value="all">全部级别</option><option value="info">INFO</option><option value="warning">WARNING</option><option value="error">ERROR</option><option value="debug">DEBUG</option></select>
          <select v-model="hub.contentFilters.category"><option value="all">全部分类</option><option value="general">常规</option><option value="system">系统</option><option value="operation">操作</option><option value="deployment">部署</option><option value="steam">Steam</option><option value="network">网络</option><option value="auth">认证</option><option value="security">安全</option><option value="audit">审计</option><option value="player">玩家</option><option value="world">世界</option><option value="mod">Mod</option><option value="backup">备份</option><option value="ai">AI</option><option value="task">任务</option><option value="file">文件</option></select>
          <button class="btn btn--secondary" :disabled="!hub.selected.value" @click="hub.searchContent">搜索</button>
        </div>

        <div class="log-terminal">
          <div v-if="!hub.selected.value" class="loghub-empty loghub-empty--terminal">从左侧选择一份日志。</div>
          <div v-else-if="hub.lines.value.length === 0" class="loghub-empty loghub-empty--terminal">没有匹配的日志记录。</div>
          <div v-for="line in hub.lines.value" v-else :key="`${line.lineNumber}-${line.text}`" class="log-line" :class="`log-line--${line.level}`">
            <span class="log-line__number">{{ line.lineNumber }}</span>
            <span class="log-line__time">{{ line.timestamp || '—' }}</span>
            <span class="log-badge" :class="`log-badge--${line.level}`">{{ levelText(line.level) }}</span>
            <span class="log-category">{{ categoryText(line.category) }}</span>
            <code>{{ line.text }}</code>
          </div>
        </div>
        <div class="log-viewer__footer">
          <span>日志正文按需由 Go 分页读取，不会一次把整个大文件送入前端。</span>
          <button v-if="hub.direction.value === 'head' && !hub.eof.value" class="btn btn--secondary" :disabled="hub.loadingContent.value" @click="hub.loadSelected('next')">继续读取</button>
        </div>
      </article>
    </section>

    <div v-if="confirmAction" class="confirm-layer" @click.self="confirmAction = ''">
      <section class="confirm-dialog" role="dialog" aria-modal="true">
        <span class="eyebrow">DESTRUCTIVE ACTION</span>
        <h3>{{ confirmTitle }}</h3>
        <p>{{ confirmDetail }}</p>
        <div class="confirm-dialog__actions"><button class="btn btn--secondary" @click="confirmAction = ''">取消</button><button class="btn btn--danger" @click="runConfirmed">确认删除</button></div>
      </section>
    </div>
  </section>
</template>

<style scoped>
.loghub-page{max-width:1760px;margin:0 auto;padding:28px 30px 42px}.loghub-heading{display:flex;justify-content:space-between;gap:28px;align-items:flex-end;margin-bottom:18px}.loghub-heading h1{margin:7px 0 5px;font-size:32px;letter-spacing:-.035em}.loghub-heading p{max-width:860px;margin:0;color:var(--muted);font-size:12px;line-height:1.75}.loghub-heading code{color:var(--text-2)}.loghub-heading__actions{display:flex;gap:8px;flex-wrap:wrap;justify-content:flex-end}.loghub-stats{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px;margin-bottom:12px}.loghub-stats article{padding:16px 18px;border:1px solid var(--border);border-radius:14px;background:var(--surface)}.loghub-stats span,.loghub-stats small{display:block;color:var(--muted);font-size:10px}.loghub-stats strong{display:block;margin:7px 0 5px;font-size:23px}.loghub-filter-panel{padding:14px;margin-bottom:12px;box-shadow:none}.loghub-filter-grid{display:grid;grid-template-columns:minmax(220px,2fr) repeat(8,minmax(120px,1fr));gap:9px}.log-field{display:grid;gap:5px;min-width:0}.log-field span{font-size:9px;color:var(--muted)}.log-field input,.log-field select,.log-content-toolbar input,.log-content-toolbar select{width:100%;height:35px;padding:0 10px;border:1px solid var(--border);border-radius:9px;background:var(--surface-2);color:var(--text);outline:none;font:inherit;font-size:10px}.log-field input:focus,.log-field select:focus,.log-content-toolbar input:focus,.log-content-toolbar select:focus{border-color:rgba(242,145,51,.58)}.loghub-filter-actions{display:flex;gap:8px;align-items:center;margin-top:12px;padding-top:11px;border-top:1px solid var(--border-soft)}.filter-spacer{flex:1}.auto-refresh{display:flex;align-items:center;gap:7px;color:var(--text-2);font-size:10px}.auto-refresh small{color:var(--muted)}.loghub-workspace{display:grid;grid-template-columns:320px minmax(0,1fr);gap:12px;min-height:670px}.log-catalog,.log-viewer{box-shadow:none;min-width:0;overflow:hidden}.log-catalog{display:flex;flex-direction:column}.log-catalog__heading{display:flex;align-items:center;justify-content:space-between;padding:15px 16px;border-bottom:1px solid var(--border-soft)}.log-catalog__heading div{display:grid;gap:4px}.log-catalog__heading strong{font-size:15px}.log-catalog__heading>span{font-size:10px;color:var(--muted)}.log-catalog__list{flex:1;overflow:auto;min-height:0}.log-file{width:100%;display:grid;grid-template-columns:8px minmax(0,1fr) 64px;gap:10px;align-items:start;padding:12px 14px;border:0;border-bottom:1px solid var(--border-soft);background:transparent;color:var(--text);text-align:left;cursor:pointer}.log-file:hover{background:var(--surface-2)}.log-file.active{background:rgba(242,145,51,.08);box-shadow:inset 2px 0 var(--accent-2)}.log-file__state{width:7px;height:7px;margin-top:4px;border-radius:50%;background:#686868}.log-file__state.running{background:#52b56f;box-shadow:0 0 0 3px rgba(82,181,111,.1)}.log-file__body,.log-file__numbers{display:grid;gap:4px;min-width:0}.log-file__body b{font-size:11px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.log-file__body small,.log-file__body em,.log-file__numbers small,.log-file__numbers em{font-style:normal;color:var(--muted);font-size:8px}.log-file__numbers{text-align:right}.log-file__numbers b{font-size:11px}.log-catalog__pager{display:flex;align-items:center;justify-content:space-between;padding:10px 12px;border-top:1px solid var(--border-soft);font-size:9px;color:var(--muted)}.log-catalog__pager button,.direction-tabs button{border:1px solid var(--border);border-radius:8px;background:var(--surface-2);color:var(--text-2);padding:6px 9px;font-size:9px;cursor:pointer}.log-catalog__pager button:disabled{opacity:.35;cursor:default}.log-viewer{display:flex;flex-direction:column}.log-viewer__header{display:flex;align-items:flex-start;justify-content:space-between;gap:18px;padding:15px 17px 13px;border-bottom:1px solid var(--border-soft)}.log-viewer__identity{min-width:0}.log-viewer__identity h2{margin:5px 0 4px;font-size:16px}.log-viewer__identity p{margin:0;color:var(--muted);font:9px/1.5 ui-monospace,SFMono-Regular,Consolas,monospace;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.log-viewer__actions{display:flex;gap:7px}.log-viewer__meta{display:flex;gap:17px;align-items:center;padding:10px 16px;border-bottom:1px solid var(--border-soft);background:var(--surface-2);font-size:9px;color:var(--muted);flex-wrap:wrap}.log-viewer__meta b{margin-right:4px;color:var(--text-2)}.live-chip{color:#65bd7a!important}.log-content-toolbar{display:grid;grid-template-columns:auto minmax(180px,1fr) 115px 125px auto;gap:8px;padding:10px 12px;border-bottom:1px solid var(--border-soft);background:#101111}.direction-tabs{display:flex;gap:4px}.direction-tabs button.active{border-color:rgba(242,145,51,.55);color:#f3a04d;background:rgba(242,145,51,.08)}.log-terminal{flex:1;min-height:500px;max-height:680px;overflow:auto;background:#080909;padding:8px 0;font-family:ui-monospace,SFMono-Regular,Consolas,"Liberation Mono",monospace}.log-line{display:grid;grid-template-columns:54px 72px 52px 56px minmax(0,1fr);gap:8px;align-items:start;padding:4px 11px;border-left:2px solid transparent;font-size:9px;line-height:1.55}.log-line:hover{background:#111313}.log-line--error{border-left-color:#c95050;background:rgba(201,80,80,.045)}.log-line--warning{border-left-color:#c7933b}.log-line__number,.log-line__time{color:#626767;text-align:right}.log-line code{white-space:pre-wrap;overflow-wrap:anywhere;color:#c8cccc;font:inherit}.log-badge{padding:1px 4px;border-radius:4px;text-align:center;font-size:7px;font-weight:700;color:#9ca1a1;background:#202323}.log-badge--error{color:#ffaaaa;background:#431f1f}.log-badge--warning{color:#ffd48b;background:#49371d}.log-badge--info{color:#9bc9f6;background:#1c3041}.log-badge--debug{color:#b5b5b5}.log-category{color:#858b8b;font-size:8px}.log-viewer__footer{min-height:45px;display:flex;align-items:center;justify-content:space-between;gap:12px;padding:9px 13px;border-top:1px solid var(--border-soft);font-size:8px;color:var(--muted)}.loghub-empty{padding:28px 18px;color:var(--muted);font-size:10px;line-height:1.7;text-align:center}.loghub-empty--terminal{min-height:400px;display:grid;place-items:center}.btn--danger-soft{border-color:rgba(204,79,79,.25)!important;color:#d98686!important;background:rgba(204,79,79,.06)!important}.btn--danger{border-color:#a33!important;color:white!important;background:#a33!important}.confirm-layer{position:fixed;inset:0;z-index:1000;display:grid;place-items:center;padding:20px;background:rgba(0,0,0,.62);backdrop-filter:blur(5px)}.confirm-dialog{width:min(450px,100%);padding:22px;border:1px solid #414141;border-radius:16px;background:#1b1c1c;box-shadow:0 24px 70px rgba(0,0,0,.5)}.confirm-dialog h3{margin:8px 0;font-size:20px}.confirm-dialog p{margin:0;color:#a4a7a7;font-size:11px;line-height:1.7}.confirm-dialog__actions{display:flex;justify-content:flex-end;gap:8px;margin-top:20px}@media(max-width:1450px){.loghub-filter-grid{grid-template-columns:repeat(4,minmax(130px,1fr))}.log-field--wide{grid-column:span 2}.loghub-workspace{grid-template-columns:285px minmax(0,1fr)}}@media(max-width:1050px){.loghub-heading{align-items:flex-start;flex-direction:column}.loghub-stats{grid-template-columns:repeat(2,1fr)}.loghub-workspace{grid-template-columns:1fr}.log-catalog{max-height:430px}.log-content-toolbar{grid-template-columns:1fr 1fr}.direction-tabs{grid-column:1/-1}.log-line{grid-template-columns:46px 60px 48px minmax(0,1fr)}.log-category{display:none}}@media(max-width:700px){.loghub-page{padding:20px 14px 30px}.loghub-stats{grid-template-columns:1fr 1fr}.loghub-filter-grid{grid-template-columns:1fr 1fr}.log-field--wide{grid-column:1/-1}.loghub-filter-actions{align-items:stretch;flex-wrap:wrap}.filter-spacer{display:none}.log-content-toolbar{grid-template-columns:1fr}.log-line{grid-template-columns:42px 46px minmax(0,1fr)}.log-line__time,.log-category{display:none}}
</style>
