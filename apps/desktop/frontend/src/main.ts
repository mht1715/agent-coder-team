import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'
import './styles/tokens.scss'
import './styles/global.scss'
import App from './App.vue'
import router from './router'
import i18n from './i18n'
import { useThemeStore } from './stores/theme'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(i18n)
app.use(ElementPlus)

useThemeStore(pinia).initialize()

document.documentElement.lang = i18n.global.locale.value

app.mount('#app')