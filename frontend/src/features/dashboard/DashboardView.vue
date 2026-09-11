<script setup lang="ts">
import AppIcon from '../../shared/components/AppIcon.vue'
import StatusPill from '../../shared/components/StatusPill.vue'
import { useAppStore } from '../../shared/store/app'

const app = useAppStore()
</script>

<template>
  <section class="page dashboard-page">
    <div class="hero panel">
      <div class="hero__glow"></div>
      <div class="hero__content">
        <span class="eyebrow">AI GAME MANAGER PLATFORM</span>
        <h1>现代化智能部署，<br><em>从一句话开始。</em></h1>
        <p>AI Game Manager Panel 正在构建现代化、智能化、AI 驱动的一键游戏服务器部署与管理平台。Desktop、Web 与未来远程 Agent 共用同一个 Go Core。</p>
        <div class="hero__actions">
          <button class="btn btn--primary" @click="app.ping"><AppIcon name="fire" />测试核心连接</button>
          <StatusPill v-if="app.backendMessage" :tone="app.backendReady ? 'success' : 'warning'" :label="app.backendMessage" />
        </div>
      </div>

      <div class="hero-status">
        <div class="hero-status__head">
          <span>本机AI游戏管理器面板</span>
          <StatusPill :tone="app.backendReady ? 'success' : 'warning'" :label="app.backendReady ? '运行中' : '检测中'" />
        </div>
        <div class="agmp-orb"><span></span><AppIcon name="fire" /></div>
        <strong>{{ app.backendReady ? '核心已经点燃' : '正在等待核心' }}</strong>
        <p>{{ app.info?.platform ?? '本机节点' }}</p>
        <div class="hero-status__line"><span>版本</span><b>v{{ app.info?.version ?? '0.2.3' }}</b></div>
        <div class="hero-status__line"><span>入口</span><b>{{ app.backendMode === 'desktop' ? (app.backendAdapter === 'electron' ? 'Electron Desktop' : 'Wails Desktop') : 'Web Browser' }}</b></div>
      </div>
    </div>

    <div class="section-heading">
      <div><span class="eyebrow">PLATFORM OVERVIEW</span><h2>平台概览</h2></div>
      <span class="section-heading__hint">0.1.67 增加 GitHub 安全门禁与源码开发许可证模式；正式 Release 仍保持许可证校验。</span>
    </div>

    <div class="overview-grid">
      <article class="overview-card panel">
        <div class="overview-card__icon orange"><AppIcon name="servers" /></div>
        <div><span>服务器实例</span><strong>0</strong><small>等待首个服务器实例</small></div>
      </article>
      <article class="overview-card panel">
        <div class="overview-card__icon amber"><AppIcon name="games" /></div>
        <div><span>已接入游戏</span><strong>1</strong><small>当前：饥荒联机版 Game Workspace</small></div>
      </article>
      <article class="overview-card panel">
        <div class="overview-card__icon green"><AppIcon name="nodes" /></div>
        <div><span>可用节点</span><strong>{{ app.backendReady ? '1' : '0' }}</strong><small>{{ app.backendReady ? '本机 Windows' : '等待本机核心' }}</small></div>
      </article>
      <article class="overview-card panel">
        <div class="overview-card__icon blue"><AppIcon name="check" /></div>
        <div><span>核心状态</span><strong class="overview-card__state">{{ app.backendReady ? '正常' : '待检测' }}</strong><small>{{ app.backendAdapter === 'wails' ? 'Wails Bridge' : (app.backendAdapter === 'electron' ? 'Electron Shell + HTTP Bridge' : 'HTTP Bridge') }} → 同一 Go Core</small></div>
      </article>
    </div>

    <div class="dashboard-grid">
      <article class="panel board-card">
        <div class="card-heading"><div><span class="eyebrow">QUICK START</span><h3>快速开始</h3></div><StatusPill tone="muted" label="规划中" /></div>
        <div class="quick-list">
          <div class="quick-item"><span class="quick-item__num">01</span><div><strong>选择游戏</strong><small>以后从 Steam、Minecraft 等游戏生态中选择服务端。</small></div></div>
          <div class="quick-item"><span class="quick-item__num">02</span><div><strong>创建实例</strong><small>AI Game Manager Panel 负责检查环境、生成原生配置并准备启动参数。</small></div></div>
          <div class="quick-item"><span class="quick-item__num">03</span><div><strong>点燃AI游戏管理器面板</strong><small>统一启动、状态、日志、备份与玩家管理入口。</small></div></div>
        </div>
      </article>

      <article class="panel board-card architecture-card">
        <div class="card-heading"><div><span class="eyebrow">ARCHITECTURE</span><h3>平台分层</h3></div></div>
        <div class="architecture-flow">
          <span>Vue UI</span><i>→</i><span>{{ app.backendAdapter === 'wails' ? 'Wails' : (app.backendAdapter === 'electron' ? 'Electron + HTTP' : 'HTTP') }} Bridge</span><i>→</i><span>Go Core</span><i>→</i><span>Game / Platform</span>
        </div>
        <p>界面不直接操作 Steam、文件和游戏进程，所有系统能力都通过 Go Core 统一执行。</p>
      </article>
    </div>
  </section>
</template>
