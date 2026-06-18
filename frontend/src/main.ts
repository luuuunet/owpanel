import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

import { bootstrapTheme } from './stores/theme'
import App from './App.vue'
import router from './router'
import i18n from './locales'
import './styles/theme-cloudflare.css'
import './styles/theme-apple.css'
import './styles/theme-dark.css'
import './styles/main.css'

bootstrapTheme()

const app = createApp(App)

for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(createPinia())
app.use(i18n)
app.use(router)
app.use(ElementPlus)
app.mount('#app')
