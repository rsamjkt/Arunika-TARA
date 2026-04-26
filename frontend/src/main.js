// APM (T.A.R.A) — Vue app entry point.
//
// Wire Pinia (state management) + Vue Router (navigation), lalu mount.
import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'
import { router } from './router'
import './style.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
