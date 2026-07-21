<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Button } from 'primevue'
import { useAuthStore } from '@/stores/auth'

interface LeaderboardStudent {
  login: string
  display_name: string
  image: string | null
  general: number
  standard: number
  stream: number
  elective: number
  total: number
  percentage: number
}

interface LeaderboardData {
  updated_at: string | null
  students: LeaderboardStudent[]
}

const TOTAL = 132
const PAGE_SIZE = 10

const SEGMENTS = [
  { key: 'general'  as const, color: '#22c55e' },
  { key: 'standard' as const, color: '#3b82f6' },
  { key: 'stream'   as const, color: '#8b5cf6' },
  { key: 'elective' as const, color: '#f97316' },
]

const router = useRouter()
const auth = useAuthStore()

const data = ref<LeaderboardData | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const page = ref(1)

const students = computed(() => data.value?.students ?? [])
const totalPages = computed(() => Math.ceil(students.value.length / PAGE_SIZE))

const pageStudents = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return students.value.slice(start, start + PAGE_SIZE)
})

const updatedAt = computed(() => {
  const iso = data.value?.updated_at
  if (!iso) return null
  return new Date(iso).toLocaleString()
})

function rankOf(login: string) {
  return students.value.findIndex(s => s.login === login) + 1
}

function segWidth(student: LeaderboardStudent, key: typeof SEGMENTS[number]['key']) {
  return `${(student[key] / TOTAL) * 100}%`
}

function remainWidth(student: LeaderboardStudent) {
  return `${Math.max(0, (TOTAL - student.total) / TOTAL * 100)}%`
}

onMounted(async () => {
  if (!auth.token) {
    router.replace('/')
    return
  }
  try {
    const base = import.meta.env.VITE_WORKER_URL || import.meta.env.VITE_42_API_BASE || ''
    const res = await fetch(`${base}/leaderboard`)
    if (!res.ok) throw new Error(`${res.status}`)
    data.value = await res.json()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load leaderboard'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="flex flex-col gap-4">
    <div class="flex items-center gap-3">
      <Button
        icon="pi pi-arrow-left"
        rounded
        outlined
        severity="secondary"
        @click="router.back()"
      />
      <span class="font-semibold text-lg">Leaderboard</span>
      <span v-if="updatedAt" class="lb-updated">Updated {{ updatedAt }}</span>
    </div>

    <div v-if="loading" class="flex justify-center py-16">
      <i class="pi pi-spinner pi-spin text-3xl text-gray-400" />
    </div>

    <div v-else-if="error" class="text-center py-16 text-red-400 text-sm">
      Could not load leaderboard: {{ error }}
    </div>

    <div v-else-if="!students.length" class="text-center py-16 text-gray-400 text-sm">
      Leaderboard is being built — check back in a few minutes.
    </div>

    <template v-else>
      <div class="lb-card">
        <div
          v-for="student in pageStudents"
          :key="student.login"
          class="lb-row"
        >
          <!-- Rank -->
          <span class="lb-rank">{{ rankOf(student.login) }}</span>

          <!-- Avatar -->
          <div class="lb-avatar">
            <span class="lb-avatar__initial">{{ student.login[0]?.toUpperCase() }}</span>
            <img
              v-if="student.image"
              :src="student.image"
              class="lb-avatar__img"
              alt=""
              loading="lazy"
              @load="(e) => (e.target as HTMLElement).classList.add('is-loaded')"
            />
          </div>

          <!-- Login -->
          <router-link :to="`/profile/${student.login}`" class="lb-login">
            {{ student.login }}
          </router-link>

          <!-- Segmented progress bar -->
          <div class="lb-bar-wrap">
            <div class="lb-bar">
              <div
                v-for="seg in SEGMENTS"
                :key="seg.key"
                class="lb-bar__seg"
                :style="{ width: segWidth(student, seg.key), background: seg.color }"
              />
              <div
                class="lb-bar__seg lb-bar__seg--remaining"
                :style="{ width: remainWidth(student) }"
              />
            </div>
          </div>

          <!-- Percentage -->
          <span class="lb-pct">{{ student.percentage.toFixed(1) }}%</span>
        </div>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="lb-pagination">
        <button
          class="lb-page-btn"
          :disabled="page === 1"
          @click="page--"
        >
          <i class="pi pi-chevron-left" />
        </button>

        <button
          v-for="p in totalPages"
          :key="p"
          class="lb-page-btn"
          :class="{ 'lb-page-btn--active': p === page }"
          @click="page = p"
        >
          {{ p }}
        </button>

        <button
          class="lb-page-btn"
          :disabled="page === totalPages"
          @click="page++"
        >
          <i class="pi pi-chevron-right" />
        </button>
      </div>
    </template>
  </main>
</template>

<style scoped>
.lb-updated {
  font-size: 0.75rem;
  color: var(--p-text-muted-color, #9ca3af);
}

/* Card */
.lb-card {
  border: 1px solid var(--card-border);
  border-radius: 1.5rem;
  background: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
  overflow: hidden;
}

/* Row */
.lb-row {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  padding: 0.75rem 1.25rem;
  border-bottom: 1px solid var(--card-border);
}

.lb-row:last-child {
  border-bottom: none;
}

.lb-row:hover {
  background: var(--color-background-mute);
}

/* Rank */
.lb-rank {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--p-text-muted-color, #9ca3af);
  width: 1.5rem;
  text-align: right;
  flex-shrink: 0;
}

/* Avatar */
.lb-avatar {
  position: relative;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  overflow: hidden;
  background: var(--p-primary-color, #3b82f6);
  flex-shrink: 0;
}

.lb-avatar__initial {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 0.875rem;
  color: #fff;
  user-select: none;
}

.lb-avatar__img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.2s;
}

.lb-avatar__img.is-loaded {
  opacity: 1;
}

/* Login */
.lb-login {
  font-size: 0.875rem;
  font-weight: 500;
  text-decoration: none;
  color: inherit;
  width: 7rem;
  flex-shrink: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.lb-login:hover {
  color: var(--p-primary-color, #3b82f6);
  text-decoration: underline;
}

/* Progress bar */
.lb-bar-wrap {
  flex: 1;
  min-width: 0;
}

.lb-bar {
  display: flex;
  height: 8px;
  border-radius: 999px;
  overflow: hidden;
  background: transparent;
}

.lb-bar__seg {
  height: 100%;
  transition: width 0.4s ease;
}

.lb-bar__seg--remaining {
  background: var(--color-background-mute);
  flex: 1;
}

/* Percentage */
.lb-pct {
  font-size: 0.8rem;
  font-weight: 600;
  width: 3.5rem;
  text-align: right;
  flex-shrink: 0;
  color: var(--p-text-muted-color, #9ca3af);
}

/* Pagination */
.lb-pagination {
  display: flex;
  justify-content: center;
  gap: 0.375rem;
}

.lb-page-btn {
  min-width: 2rem;
  height: 2rem;
  padding: 0 0.5rem;
  border-radius: 0.5rem;
  border: 1px solid var(--card-border);
  background: var(--card-bg);
  color: inherit;
  font-size: 0.8rem;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.lb-page-btn:hover:not(:disabled) {
  background: var(--color-background-mute);
}

.lb-page-btn--active {
  background: var(--p-primary-color, #3b82f6) !important;
  border-color: var(--p-primary-color, #3b82f6) !important;
  color: #fff;
}

.lb-page-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
</style>
