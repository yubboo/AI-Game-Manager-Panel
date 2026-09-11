import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './app/App.vue'
import router from './app/router'
import './shared/styles/base.css'

createApp(App).use(createPinia()).use(router).mount('#app')

// 首帧稳定后才恢复过渡动画，避免刷新时组件初始状态产生亮暗/位移闪烁。
window.requestAnimationFrame(() => {
  window.requestAnimationFrame(() => document.documentElement.classList.remove('agmp-preload'))
})
