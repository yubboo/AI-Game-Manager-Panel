<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppIcon from '../../shared/components/AppIcon.vue'
import { backend } from '../../shared/api/backend'
import { useAppStore } from '../../shared/store/app'

const app = useAppStore()
const installing = ref(false)
const actionMessage = ref('')

const status = computed(() => app.update)
const desktopInstallSupported = computed(() => app.backendAdapter === 'wails')

function formatBytes(value: number) {
  if (!value || value < 0) return '未知'
  const mb = value / 1024 / 1024
  return `${mb.toFixed(mb >= 100 ? 0 : 1)} MB`
}

function formatTime(value: string) {
  if (!value) return '—'
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? value : d.toLocaleString('zh-CN')
}

async function check(force = true) {
  actionMessage.value = ''
  const ok = await app.checkForUpdates(force)
  if (!ok && app.updateError) actionMessage.value = app.updateError
}

async function install() {
  if (!status.value?.updateAvailable) return
  const confirmed = window.confirm('更新会退出 AI游戏管理器面板，并按正常关闭流程停止当前 Core 管理的本机游戏进程。请先保存游戏进度、完成备份并停服。\n\n确认现在下载并进入升级安装吗？')
  if (!confirmed) return
  installing.value = true
  actionMessage.value = '正在下载并校验 Windows 安装器。完成后会打开中文升级向导，并自动退出当前程序。'
  try {
    await backend.installLatestUpdate()
    actionMessage.value = '更新安装器已启动，当前程序即将退出。'
  } catch (error) {
    actionMessage.value = error instanceof Error ? error.message : String(error)
    installing.value = false
  }
}

onMounted(() => {
  if (!app.update) void check(false)
})
</script>

<template>
  <section class="settings-section">
    <div class="settings-section__intro">
      <div class="settings-section__icon"><AppIcon name="check" /></div>
      <div>
        <strong>更新与升级</strong>
        <span>Windows 正式版从 GitHub Releases 检查稳定更新。下载安装器前必须通过 SHA256 校验，升级只替换程序文件，不删除 runtime 用户数据。</span>
      </div>
    </div>

    <div class="feature-grid feature-grid--two">
      <div class="feature-card">
        <span class="feature-title">当前版本</span>
        <p class="feature-description">正在运行的 AI游戏管理器面板版本。</p>
        <strong>v{{ status?.currentVersion || app.info?.version || '0.2.15' }}</strong>
      </div>
      <div class="feature-card">
        <span class="feature-title">最新稳定版</span>
        <p class="feature-description">更新通道：{{ status?.channel || app.platformConfig?.update?.channel || 'stable' }}</p>
        <strong>{{ status?.latestVersion ? `v${status.latestVersion}` : '尚未检查' }}</strong>
      </div>
    </div>

    <div v-if="status?.updateAvailable" class="feature-card update-release-card">
      <div class="update-release-card__head">
        <div>
          <span class="feature-title">发现新版本 v{{ status.latestVersion }}</span>
          <p class="feature-description">{{ status.releaseName || `AI Game Manager Panel ${status.latestVersion}` }}</p>
        </div>
        <span class="status-pill status-pill--success">可更新</span>
      </div>
      <div class="update-meta-grid">
        <span>安装包：<strong>{{ formatBytes(status.installerSize) }}</strong></span>
        <span>发布时间：<strong>{{ formatTime(status.publishedAt) }}</strong></span>
        <span>完整性：<strong>{{ status.installerSha256 ? 'SHA256 已提供' : '缺少校验' }}</strong></span>
      </div>
      <div class="update-release-notes">
        <strong>更新说明</strong>
        <pre>{{ status.releaseNotes || '该 Release 暂未填写更新说明。' }}</pre>
      </div>
      <div class="settings-footer">
        <span class="save-message active">{{ actionMessage || '点击后会下载安装器、校验 SHA256、打开升级向导，然后退出当前程序。更新前请先停服并完成备份。' }}</span>
        <button class="btn btn--primary" type="button" :disabled="installing || !status.canInstall || !desktopInstallSupported" @click="install">
          {{ installing ? '正在准备更新…' : '下载并安装更新' }}
        </button>
      </div>
      <p v-if="!desktopInstallSupported" class="feature-description">当前不是 Wails Windows 桌面端。Web / Electron 请从正式 Release 下载 Wails Setup 升级。</p>
    </div>

    <div v-else class="feature-card">
      <span class="feature-title">更新状态</span>
      <p class="feature-description">{{ status?.message || app.updateError || '尚未检查更新。' }}</p>
      <strong>{{ status?.latestVersion && !status.updateAvailable ? '已是最新版本' : '等待检查' }}</strong>
    </div>

    <div class="feature-card">
      <span class="feature-title">数据保留策略</span>
      <p class="feature-description">升级使用固定 Windows AppId 识别同一应用，并复用原安装目录。安装器仅覆盖程序 EXE、静态配置和发行文件；runtime/data、runtime/backups、runtime/instances、许可证激活、管理员账号和用户设置不会主动删除。</p>
      <strong>程序更新 ≠ 清空用户数据</strong>
    </div>

    <div class="settings-footer">
      <span class="save-message" :class="{ active: actionMessage || app.updateError }">{{ actionMessage || app.updateError || `上次检查：${status?.checkedAt ? new Date(status.checkedAt * 1000).toLocaleString('zh-CN') : '尚未检查'}` }}</span>
      <button class="btn btn--secondary" type="button" :disabled="app.updateLoading || installing" @click="check(true)">
        <AppIcon name="check" />{{ app.updateLoading ? '检查中…' : '检查更新' }}
      </button>
    </div>
  </section>
</template>
