import '@shared/assets/main.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from '@core/App.vue'
import router from '@core/router'

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')
