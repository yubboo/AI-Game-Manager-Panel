import { createRouter, createWebHashHistory } from 'vue-router'
import AIWorkbenchView from '../features/xiaoyu/AIWorkbenchView.vue'
import DashboardView from '../features/dashboard/DashboardView.vue'
import DeploymentView from '../features/deployment/DeploymentView.vue'
import InstancesView from '../features/instances/InstancesView.vue'
import GamesView from '../features/games/GamesView.vue'
import LogsView from '../features/logs/LogsView.vue'
import UsersView from '../features/users/UsersView.vue'
import SettingsView from '../features/settings/SettingsView.vue'
import NodesView from '../features/nodes/NodesView.vue'
import ModulePlaceholderView from '../shared/components/ModulePlaceholderView.vue'

const placeholder = (path: string, title: string, moduleId: string, group = '本机', licenseFeature?: string) => ({
  path,
  component: ModulePlaceholderView,
  meta: { title, moduleId, group, ...(licenseFeature ? { licenseFeature } : {}) },
})

export default createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', component: AIWorkbenchView, meta: { title: '小鱼', group: '本机', licenseFeature: 'ai.workbench' } },
    { path: '/dashboard', component: DashboardView, meta: { title: '仪表盘', group: '本机' } },
    { path: '/deployment', component: DeploymentView, meta: { title: '一键部署', group: '本机' } },
    { path: '/instances', component: InstancesView, meta: { title: '实例管理', group: '本机' } },
    placeholder('/tasks', '任务中心', 'tasks'),
    placeholder('/files', '文件管理', 'files'),
    placeholder('/backups', '备份恢复', 'backups'),
    { path: '/terminal', redirect: '/?panel=terminal' },
    placeholder('/scheduler', '计划任务', 'scheduler', '本机', 'automation'),
    { path: '/environment', redirect: '/settings?section=environment' },
    placeholder('/plugins', '插件扩展', 'plugins', '本机', 'plugin.extensions'),
    placeholder('/network', '网络与组网', 'network'),
    { path: '/logs', component: LogsView, meta: { title: '日志中心', group: '本机' } },
    { path: '/nodes', component: NodesView, meta: { title: '节点管理', group: '系统', licenseFeature: 'remote.agent' } },
    { path: '/users', component: UsersView, meta: { title: '用户与权限', group: '系统' } },
    { path: '/settings', component: SettingsView, meta: { title: '设置中心', group: '系统' } },
    placeholder('/developer', '开发者工具', 'developer', '系统'),

    // 已完成或正在开发的具体游戏工作台保留为部署中心的深入入口。
    { path: '/games', component: GamesView, meta: { title: '游戏工作台', group: '一键部署' } },

    // GSM 功能矩阵对应的后续平台模块统一使用一个占位视图，避免为占位阶段复制大量页面文件。
    placeholder('/game-config', '游戏配置', 'game-config'),
    placeholder('/steamcmd', 'SteamCMD 管理', 'steamcmd'),
    placeholder('/rcon', 'RCON', 'rcon'),
    placeholder('/cloud-build', '云端构建', 'cloud-build'),
    placeholder('/external-api', '外部 API', 'external-api'),
    placeholder('/security', '安全中心', 'security', '系统'),
    placeholder('/system', '系统监控', 'system', '系统'),
    placeholder('/about', '关于 AI Game Manager Panel', 'sponsor', '系统'),
  ],
})
