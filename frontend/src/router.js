// APM (T.A.R.A) - Vue Router config.
//
// Pakai memory history (BUKAN HTML5 history) karena Wails app berjalan
// di webview tanpa server, URL bar tidak relevan.
import { createRouter, createMemoryHistory } from 'vue-router'

const routes = [
  {
    // Splash = initial route. Run startup self-test → kalau ok navigate
    // replace ke /home. Kalau critical fail tetap di splash dengan
    // recovery option.
    path: '/',
    name: 'splash',
    component: () => import('./screens/SplashScreen.vue'),
  },
  {
    path: '/home',
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
    path: '/cari-pasien',
    name: 'cari-pasien',
    component: () => import('./screens/SearchPasienScreen.vue'),
  },
  {
    path: '/registrasi-umum',
    name: 'registrasi-umum',
    component: () => import('./screens/RegistrasiUmumScreen.vue'),
  },
  {
    path: '/bantu-saya',
    name: 'bantu-saya',
    component: () => import('./screens/BantuSayaScreen.vue'),
  },
  {
    path: '/result',
    name: 'result',
    component: () => import('./screens/ResultScreen.vue'),
  },
  // /antrian removed (QW1) — Antrian Loket sekarang single-tap di HomeScreen
  // langsung navigate ke /tiket. AntrianScreen file di-keep untuk reference,
  // tapi tidak ter-route.
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
