import { createApp } from 'vue'
import App from './App.vue'
import './style.css'

// 禁用 Wails v3 在原生边框窗口下的前端 DOM 模拟缩放，避免标题栏下方触发缩放手柄
window._wails = window._wails || {}
if (typeof window._wails.setResizable === 'function') {
  window._wails.setResizable(false)
}
Object.defineProperty(window._wails, 'setResizable', {
  configurable: true,
  enumerable: true,
  get() {
    return () => {}
  },
  set(fn) {
    if (typeof fn === 'function') {
      fn(false)
    }
  }
})

createApp(App).mount('#app')
