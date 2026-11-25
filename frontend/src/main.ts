// Self-hosted variable families. One file per family carries the full weight
// range, so no static weight imports are needed.
import '@fontsource-variable/plus-jakarta-sans'
import '@fontsource-variable/jetbrains-mono'

import '@/styles/tokens.css'
import '@/styles/base.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import { useFailureStore } from './stores/failure'

const app = createApp(App)

app.use(createPinia())
app.use(router)

// A crash inside a view must not leave a blank screen.
app.config.errorHandler = (error, _instance, info) => {
  console.error(`[PromisedLand] ${info}`, error)
  useFailureStore().markRenderFailed()
}

app.mount('#app')
