// APM (T.A.R.A) - Vue Router config.
//
// Pakai memory history (BUKAN HTML5 history) karena Wails app berjalan
// di webview tanpa server, URL bar tidak relevan.
import { createRouter, createMemoryHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'home',
    component: () => import('./screens/HomeScreen.vue'),
  },
  {
    path: '/input',
    name: 'input',
    component: () => import('./screens/InputScreen.vue'),
  },
  {
    path: '/detect',
    name: 'detect',
    component: () => import('./screens/DetectScreen.vue'),
  },
  {
    path: '/result',
    name: 'result',
    component: () => import('./screens/ResultScreen.vue'),
  },
  {
    path: '/antrian',
    name: 'antrian',
    component: () => import('./screens/AntrianScreen.vue'),
  },
  {
    path: '/tiket',
    name: 'tiket',
    component: () => import('./screens/TicketScreen.vue'),
  },
  {
    path: '/admin',
    name: 'admin',
    component: () => import('./screens/AdminScreen.vue'),
  },
  // Catch-all 404 - balik ke home untuk kiosk safety
  {
    path: '/:pathMatch(.*)*',
    redirect: { name: 'home' },
  },
]

export const router = createRouter({
  history: createMemoryHistory(),
  routes,
})
