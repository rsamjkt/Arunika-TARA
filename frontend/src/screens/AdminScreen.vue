<!--
  AdminScreen — operator IT panel.

  Spec P-046:
    - PIN gate dulu (kalau cfg.admin.pin tidak kosong di backend)
    - Header: "Panel admin" kiri + tombol Keluar merah kanan
    - Stat grid 2x2: Antrian today, SEP today, Pending sync, Uptime
    - Status komponen card dengan refresh 10s
    - Pending SEP table dengan Confirm modal
    - Action grid 2x2: Reset, Lihat log, Test print, Mock info (non-Win)
-->
<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import PinGate from '../components/PinGate.vue'
import StatCard from '../components/StatCard.vue'
import StatusList from '../components/StatusList.vue'
import ActionTile from '../components/ActionTile.vue'
import AlertModal from '../components/AlertModal.vue'

import { apmService } from '../services/apm'

const router = useRouter()

// PIN gate state
const unlocked = ref(false)
function onUnlock() {
  unlocked.value = true
  refreshAll()
}

// Data
const stats = ref(null)
const sysStatus = ref(null)
const pendingSEPs = ref([])
const logs = ref([])
const showLogs = ref(false)

// Modals
const confirmModal = ref({ visible: false, title: '', message: '', action: null })
const errorModal = ref({ visible: false, message: '' })
const successModal = ref({ visible: false, message: '' })

// Loading flags per action
const loadingReset = ref(false)
const loadingTestPrint = ref(false)
const loadingConfirm = ref({}) // key: pending sep id

// Polling timer
let pollTimer = null

async function refreshStats() {
  try {
    stats.value = await apmService.getAdminStats()
  } catch {}
}
async function refreshSysStatus() {
  try {
    sysStatus.value = await apmService.getSystemStatus()
  } catch {}
}
async function refreshPending() {
  try {
    pendingSEPs.value = await apmService.getPendingSEPs() ?? []
  } catch {
    pendingSEPs.value = []
  }
}
async function refreshAll() {
  await Promise.all([refreshStats(), refreshSysStatus(), refreshPending()])
}

// Status komponen — derive dari sysStatus
const statusItems = computed(() => {
  const sys = sysStatus.value
  const hw = sys?.Hardware
  return [
    { label: 'BPJS VClaim API', status: sys?.Online ? 'online' : 'offline' },
    { label: 'BPJS Antrol', status: sys?.Online ? 'online' : 'offline' },
    { label: 'SIMRS Khanza', status: sys?.Online ? 'online' : 'offline' },
    { label: 'Sidik Wajah BPJS (Frista)', status: hw?.Frista ? 'online' : 'offline',
      detail: hw?.Frista ? 'Aplikasi siap' : 'Aplikasi tidak aktif' },
    { label: 'Sidik Jari BPJS (After.exe)', status: hw?.Fingerprint ? 'online' : 'offline',
      detail: hw?.Fingerprint ? 'Headless aktif' : 'Tidak aktif' },
    { label: 'Printer thermal', status: hw?.Printer ? 'online' : 'offline',
      detail: hw?.Printer ? 'OK' : 'Tidak terhubung' },
  ]
})

// Format uptime "7j 12m"
function formatUptime(sec) {
  if (!sec || sec < 0) return '0m'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  if (h > 0) return `${h}j ${m}m`
  return `${m}m`
}
function formatStartedAt(iso) {
  if (!iso) return ''
  try {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return ''
    const pad = (n) => String(n).padStart(2, '0')
    return `Sejak ${pad(d.getHours())}:${pad(d.getMinutes())} WIB`
  } catch { return '' }
}

// Format no kartu masked: "************XXXX"
function maskKartu(s) {
  if (!s || s.length < 8) return '***'
  return '*'.repeat(s.length - 4) + s.slice(-4)
}
function formatTs(iso) {
  if (!iso) return ''
  try {
    const d = new Date(iso)
    if (isNaN(d.getTime())) return iso
    const pad = (n) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
  } catch { return iso }
}

// Actions
function exitAdmin() {
  router.push({ name: 'home' })
}

function confirmReset() {
  confirmModal.value = {
    visible: true,
    title: 'Reset counter antrian?',
    message: 'Semua nomor antrian hari ini akan dihapus. Tindakan ini tidak bisa dibatalkan.',
    action: doReset,
  }
}
async function doReset() {
  confirmModal.value.visible = false
  loadingReset.value = true
  try {
    await apmService.resetCounters()
    successModal.value = { visible: true, message: 'Counter antrian berhasil di-reset.' }
    await refreshAll()
  } catch (e) {
    errorModal.value = { visible: true, message: e?.message ?? 'Reset gagal' }
  } finally {
    loadingReset.value = false
  }
}

async function viewLogs() {
  try {
    logs.value = await apmService.getRecentLogs(50)
    showLogs.value = true
  } catch (e) {
    errorModal.value = { visible: true, message: `Gagal load logs: ${e?.message ?? ''}` }
  }
}

async function doTestPrint() {
  loadingTestPrint.value = true
  try {
    await apmService.testPrint()
    successModal.value = { visible: true, message: 'Test print dikirim. Periksa printer.' }
  } catch (e) {
    errorModal.value = { visible: true, message: e?.message ?? 'Test print gagal' }
  } finally {
    loadingTestPrint.value = false
  }
}

function confirmPendingSync(pending) {
  confirmModal.value = {
    visible: true,
    title: 'Konfirmasi sync SEP?',
    message: `SEP untuk kartu ****${pending.no_kartu?.slice?.(-4) ?? ''} akan dikirim ke Khanza saat reconcile berikutnya. Lanjutkan?`,
    action: () => doConfirmSync(pending.id),
  }
}
async function doConfirmSync(id) {
  confirmModal.value.visible = false
  loadingConfirm.value = { ...loadingConfirm.value, [id]: true }
  try {
    await apmService.confirmSEPSync(id)
    await refreshPending()
  } catch (e) {
    errorModal.value = { visible: true, message: e?.message ?? 'Confirm gagal' }
  } finally {
    loadingConfirm.value = { ...loadingConfirm.value, [id]: false }
  }
}

const isWindows = computed(() => sysStatus.value?.Platform === 'windows')
const showMockInfo = computed(() => !isWindows.value)
function openMockInfo() {
  // Open new tab/window via standard URL — Wails webview support window.open
  // sebagai navigation, atau show inline modal.
  successModal.value = {
    visible: true,
    message: 'Mock biometrik server: http://localhost:9090\n' +
             'Endpoint: /mock/face-verify, /mock/fp-verify, /mock/fp-fail',
  }
}

onMounted(async () => {
  // Kalau backend cfg.admin.pin kosong, VerifyAdminPIN("") return true.
  // Gate akan auto-unlock kalau PIN dimasukkan kosong, tapi UX-nya
  // user tetap harus kasih input. Sederhana: skip gate kalau verify("")
  // return true (mode dev tanpa PIN).
  try {
    const ok = await apmService.verifyAdminPIN('')
    if (ok) {
      unlocked.value = true
      refreshAll()
      pollTimer = setInterval(() => {
        refreshSysStatus()
        refreshStats()
        refreshPending()
      }, 10000)
    }
  } catch {}
})
onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <PinGate v-if="!unlocked" @unlock="onUnlock" />

    <template v-else>
      <!-- Header -->
      <header
        class="bg-surface border-b border-border flex items-center justify-between
               px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.8vw,16px)]"
      >
        <h1 class="text-[clamp(15px,2.2vw,18px)] font-medium text-text-primary">
          Panel admin
        </h1>
        <button
          type="button"
          class="text-rose-700 hover:text-rose-800 px-3 py-1 rounded-btn
                 text-[clamp(12px,1.6vw,14px)] font-medium
                 hover:bg-rose-50 active:bg-rose-100"
          @click="exitAdmin"
        >
          Keluar
        </button>
      </header>

      <!-- Body -->
      <section class="flex-1 p-[clamp(12px,2.5vw,20px)] max-w-[960px] mx-auto w-full
                      flex flex-col gap-[clamp(14px,2.5vw,20px)]">

        <!-- Stat grid 2x2 -->
        <div class="grid grid-cols-2 gap-[clamp(8px,1.5vw,12px)]">
          <StatCard
            label="Antrian hari ini"
            :value="stats?.antrian_hari_ini ?? 0"
            sub="Reset pukul 00:01"
          />
          <StatCard
            label="SEP berhasil"
            :value="stats?.sep_hari_ini ?? 0"
            sub="Hari ini"
          />
          <StatCard
            label="Pending rekonsiliasi"
            :value="stats?.pending_sync ?? 0"
            :variant="(stats?.pending_sync ?? 0) > 0 ? 'warn' : 'default'"
            sub="Butuh konfirmasi"
          />
          <StatCard
            label="Uptime"
            :value="formatUptime(stats?.uptime_sec)"
            :sub="formatStartedAt(stats?.started_at)"
          />
        </div>

        <!-- Status komponen -->
        <div>
          <h2 class="text-[clamp(11px,1.5vw,13px)] uppercase tracking-wide
                     text-text-muted font-medium mb-2">
            Status komponen
          </h2>
          <StatusList :items="statusItems" />
        </div>

        <!-- Pending SEP table -->
        <div v-if="pendingSEPs.length > 0">
          <h2 class="text-[clamp(11px,1.5vw,13px)] uppercase tracking-wide
                     text-text-muted font-medium mb-2">
            Pending SEP ({{ pendingSEPs.length }})
          </h2>
          <div class="bg-surface border border-border rounded-card overflow-hidden">
            <table class="w-full text-[clamp(11px,1.5vw,13px)]">
              <thead class="bg-bg text-text-muted">
                <tr>
                  <th class="text-left px-3 py-2 font-medium">No Kartu</th>
                  <th class="text-left px-3 py-2 font-medium">Kategori</th>
                  <th class="text-left px-3 py-2 font-medium">Waktu</th>
                  <th class="text-right px-3 py-2 font-medium">Aksi</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr v-for="p in pendingSEPs" :key="p.id">
                  <td class="px-3 py-2 font-mono text-text-secondary">
                    {{ maskKartu(p.no_kartu) }}
                  </td>
                  <td class="px-3 py-2">{{ p.kategori }}</td>
                  <td class="px-3 py-2 text-text-muted">
                    {{ formatTs(p.created_at?.Time ?? p.created_at) }}
                  </td>
                  <td class="px-3 py-2 text-right">
                    <button
                      type="button"
                      :disabled="loadingConfirm[p.id]"
                      class="text-blue text-[clamp(11px,1.4vw,12px)] font-medium
                             hover:underline disabled:opacity-50"
                      @click="confirmPendingSync(p)"
                    >
                      {{ loadingConfirm[p.id] ? 'Memproses...' : 'Konfirmasi' }}
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <!-- Action grid -->
        <div>
          <h2 class="text-[clamp(11px,1.5vw,13px)] uppercase tracking-wide
                     text-text-muted font-medium mb-2">
            Tindakan
          </h2>
          <div class="grid grid-cols-2 gap-[clamp(8px,1.5vw,12px)]">
            <ActionTile
              title="Reset counter antrian"
              subtitle="Manual reset (selain cron 00:01)"
              variant="danger"
              :loading="loadingReset"
              @click="confirmReset"
            >
              <template #icon>
                <svg
                  xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round"
                  stroke-linejoin="round" class="w-5 h-5"
                >
                  <path d="M3 12a9 9 0 1 0 9-9" />
                  <polyline points="3 4 3 12 11 12" />
                </svg>
              </template>
            </ActionTile>

            <ActionTile
              title="Lihat log rekonsiliasi"
              subtitle="50 entry terakhir"
              @click="viewLogs"
            >
              <template #icon>
                <svg
                  xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round"
                  stroke-linejoin="round" class="w-5 h-5"
                >
                  <line x1="8" y1="6" x2="21" y2="6" />
                  <line x1="8" y1="12" x2="21" y2="12" />
                  <line x1="8" y1="18" x2="21" y2="18" />
                  <line x1="3" y1="6" x2="3.01" y2="6" />
                  <line x1="3" y1="12" x2="3.01" y2="12" />
                  <line x1="3" y1="18" x2="3.01" y2="18" />
                </svg>
              </template>
            </ActionTile>

            <ActionTile
              title="Test cetak printer"
              subtitle="Cetak halaman uji"
              variant="success"
              :loading="loadingTestPrint"
              @click="doTestPrint"
            >
              <template #icon>
                <svg
                  xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round"
                  stroke-linejoin="round" class="w-5 h-5"
                >
                  <polyline points="6 9 6 2 18 2 18 9" />
                  <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2" />
                  <rect x="6" y="14" width="12" height="8" />
                </svg>
              </template>
            </ActionTile>

            <ActionTile
              v-if="showMockInfo"
              title="Info mock server"
              subtitle="Endpoint dev di port 9090"
              @click="openMockInfo"
            >
              <template #icon>
                <svg
                  xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                  stroke="currentColor" stroke-width="2" stroke-linecap="round"
                  stroke-linejoin="round" class="w-5 h-5"
                >
                  <circle cx="12" cy="12" r="10" />
                  <line x1="12" y1="16" x2="12" y2="12" />
                  <line x1="12" y1="8" x2="12.01" y2="8" />
                </svg>
              </template>
            </ActionTile>
          </div>
        </div>
      </section>
    </template>

    <!-- Confirm modal -->
    <AlertModal
      :visible="confirmModal.visible"
      variant="warning"
      :title="confirmModal.title"
      :message="confirmModal.message"
      primary-label="Lanjutkan"
      close-label="Batal"
      @primary="confirmModal.action && confirmModal.action()"
      @close="confirmModal.visible = false"
    />

    <!-- Error modal -->
    <AlertModal
      :visible="errorModal.visible"
      variant="error"
      title="Terjadi kesalahan"
      :message="errorModal.message"
      primary-label="Tutup"
      close-label="OK"
      @primary="errorModal.visible = false"
      @close="errorModal.visible = false"
    />

    <!-- Success modal -->
    <AlertModal
      :visible="successModal.visible"
      variant="success"
      title="Berhasil"
      :message="successModal.message"
      primary-label="OK"
      close-label="Tutup"
      @primary="successModal.visible = false"
      @close="successModal.visible = false"
    />

    <!-- Logs modal (separate karena tabel) -->
    <Transition
      enter-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0" enter-to-class="opacity-100"
    >
      <div
        v-if="showLogs"
        class="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm
               flex items-center justify-center p-4"
        @click.self="showLogs = false"
      >
        <div class="bg-surface rounded-card max-w-[720px] w-full max-h-[80vh] flex flex-col">
          <div class="flex items-center justify-between p-4 border-b border-border">
            <h3 class="text-[clamp(13px,1.8vw,16px)] font-medium">
              Log rekonsiliasi (50 terakhir)
            </h3>
            <button
              type="button"
              class="text-text-muted hover:text-text-primary"
              @click="showLogs = false"
            >
              ✕
            </button>
          </div>
          <div class="overflow-y-auto p-4">
            <table v-if="logs.length > 0" class="w-full text-[clamp(10px,1.3vw,12px)]">
              <thead class="bg-bg text-text-muted sticky top-0">
                <tr>
                  <th class="text-left px-2 py-1 font-medium">Waktu</th>
                  <th class="text-left px-2 py-1 font-medium">Tabel</th>
                  <th class="text-left px-2 py-1 font-medium">Aksi</th>
                  <th class="text-left px-2 py-1 font-medium">Hasil</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr v-for="l in logs" :key="l.id">
                  <td class="px-2 py-1 font-mono text-text-muted">{{ formatTs(l.timestamp) }}</td>
                  <td class="px-2 py-1">{{ l.table_name }}</td>
                  <td class="px-2 py-1">{{ l.action }}</td>
                  <td class="px-2 py-1 text-text-secondary">{{ l.result || '—' }}</td>
                </tr>
              </tbody>
            </table>
            <p v-else class="text-text-muted text-center py-8">Belum ada log.</p>
          </div>
        </div>
      </div>
    </Transition>
  </main>
</template>
