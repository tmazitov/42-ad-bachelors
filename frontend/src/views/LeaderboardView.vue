<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Button, InputText, IconField, InputIcon } from 'primevue'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/services/api'
import StudentAvatar from '@/components/leaderboard/StudentAvatar.vue'

interface LeaderboardStudent {
  rank: number
  login: string
  display_name: string
  image: string | null
  general: number
  standard: number
  stream: number
  elective: number
  total: number
  percentage: number
  synced_at: string
}

interface LeaderboardData {
  updated_at: string | null
  students: LeaderboardStudent[]
}

type SegmentKey = 'general' | 'standard' | 'stream' | 'elective'

const TOTAL = 132
const PAGE_SIZE = 10

const SEGMENTS: { key: SegmentKey; label: string; color: string }[] = [
  { key: 'general', label: 'General', color: '#22c55e' },
  { key: 'standard', label: 'Standard', color: '#3b82f6' },
  { key: 'stream', label: 'Stream', color: '#8b5cf6' },
  { key: 'elective', label: 'Elective', color: '#f97316' },
]

const router = useRouter()
const auth = useAuthStore()

const data = ref<LeaderboardData | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const page = ref(1)
const search = ref('')

const students = computed(() => data.value?.students ?? [])
const query = computed(() => search.value.trim().toLowerCase())

const listed = computed(() => {
  if (!query.value) return students.value
  return students.value.filter(
    s => s.login.toLowerCase().includes(query.value) || s.display_name.toLowerCase().includes(query.value),
  )
})

const totalPages = computed(() => Math.ceil(listed.value.length / PAGE_SIZE))
const pageStudents = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return listed.value.slice(start, start + PAGE_SIZE)
})

watch(query, () => { page.value = 1 })

// Page buttons: all pages when there are few, otherwise the first, the last
// and a window around the current page, e.g. 1 … 4 5 6 … 20.
const pageItems = computed<(number | '…')[]>(() => {
  const total = totalPages.value
  const current = page.value
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1)

  const start = Math.max(2, Math.min(current - 1, total - 4))
  const end = Math.min(total - 1, Math.max(current + 1, 5))
  const items: (number | '…')[] = [1]
  if (start > 2) items.push('…')
  for (let p = start; p <= end; p++) items.push(p)
  if (end < total - 1) items.push('…')
  items.push(total)
  return items
})

const updatedAt = computed(() => {
  const iso = data.value?.updated_at
  return iso ? new Date(iso).toLocaleString() : null
})

const average = computed(() => {
  const list = students.value
  if (!list.length) return 0
  return list.reduce((sum, s) => sum + s.percentage, 0) / list.length
})

const me = computed(() => students.value.find(s => s.login === auth.user?.login) ?? null)

// Share of other students ranked below me.
const percentile = computed(() => {
  if (!me.value || students.value.length < 2) return null
  const below = students.value.filter(s => s.total < me.value!.total).length
  return Math.round((below / (students.value.length - 1)) * 100)
})

function goToMe() {
  if (!me.value) return
  search.value = ''
  const index = listed.value.indexOf(me.value)
  if (index >= 0) page.value = Math.floor(index / PAGE_SIZE) + 1
  requestAnimationFrame(() => {
    document.getElementById(`lb-${me.value!.login}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  })
}

function segWidth(student: LeaderboardStudent, key: SegmentKey) {
  return `${(student[key] / TOTAL) * 100}%`
}

onMounted(async () => {
  try {
    data.value = await api<LeaderboardData>('/leaderboard')
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load leaderboard'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <main class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex items-center gap-3">
      <Button icon="pi pi-arrow-left" rounded outlined severity="secondary" @click="router.back()" />
      <div class="flex flex-col">
        <span class="font-semibold text-lg leading-tight">Leaderboard</span>
        <span v-if="updatedAt" class="lb-muted text-xs">Updated {{ updatedAt }}</span>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center py-16">
      <i class="pi pi-spinner pi-spin text-3xl text-gray-400" />
    </div>

    <div v-else-if="error" class="lb-card lb-empty text-red-400">
      <i class="pi pi-exclamation-triangle text-2xl" />
      Could not load leaderboard: {{ error }}
    </div>

    <div v-else-if="!students.length" class="lb-card lb-empty">
      <i class="pi pi-trophy text-3xl" />
      <span class="font-medium">No students yet</span>
      <span class="lb-muted text-sm">Students appear here after they log in.</span>
    </div>

    <template v-else>
      <!-- Stats -->
      <div class="lb-stats">
        <div class="lb-card lb-stat">
          <span class="lb-stat__label"><i class="pi pi-users" /> Students</span>
          <span class="lb-stat__value">{{ students.length }}</span>
        </div>
        <div class="lb-card lb-stat">
          <span class="lb-stat__label"><i class="pi pi-chart-line" /> Average progress</span>
          <span class="lb-stat__value">{{ average.toFixed(1) }}%</span>
        </div>
        <button v-if="me" class="lb-card lb-stat lb-stat--me" title="Show my row" @click="goToMe">
          <span class="lb-stat__label"><i class="pi pi-user" /> Your rank</span>
          <span class="lb-stat__value">
            #{{ me.rank }}
            <span class="lb-stat__sub">
              {{ me.total }}/{{ TOTAL }} credits<template v-if="percentile !== null"> · ahead of {{ percentile }}%</template>
            </span>
          </span>
        </button>
      </div>

      <!-- Toolbar -->
      <div class="lb-toolbar">
        <IconField class="lb-search">
          <InputIcon class="pi pi-search" />
          <InputText v-model="search" placeholder="Search students…" size="small" class="w-full" />
        </IconField>
        <div class="lb-legend">
          <span v-for="seg in SEGMENTS" :key="seg.key" class="lb-legend__item">
            <span class="lb-legend__dot" :style="{ background: seg.color }" />
            {{ seg.label }}
          </span>
        </div>
      </div>

      <!-- List -->
      <div v-if="pageStudents.length" class="lb-card lb-list">
        <div
          v-for="s in pageStudents"
          :id="`lb-${s.login}`"
          :key="s.login"
          class="lb-row"
        >
          <!-- Rank -->
          <span class="lb-rank">{{ s.rank }}</span>

          <!-- Avatar -->
          <StudentAvatar :login="s.login" :image="s.image" :size="36" />

          <!-- Login -->
          <router-link :to="`/profile/${s.login}`" class="lb-login">
            {{ s.login }}
          </router-link>

          <!-- Segmented progress bar -->
          <div class="lb-bar-wrap">
            <div class="lb-bar">
              <div
                v-for="seg in SEGMENTS"
                :key="seg.key"
                class="lb-bar__seg"
                :style="{ width: segWidth(s, seg.key), background: seg.color }"
              />
            </div>
          </div>

          <!-- Percentage -->
          <span class="lb-pct">{{ s.percentage.toFixed(1) }}%</span>
        </div>
      </div>

      <div v-else-if="query" class="lb-card lb-empty lb-muted text-sm">
        No students match “{{ search }}”.
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="lb-pagination">
        <button class="lb-page-btn" :disabled="page === 1" @click="page--">
          <i class="pi pi-chevron-left" />
        </button>
        <template v-for="(item, i) in pageItems" :key="`${item}-${i}`">
          <span v-if="item === '…'" class="lb-page-gap">…</span>
          <button
            v-else
            class="lb-page-btn"
            :class="{ 'lb-page-btn--active': item === page }"
            @click="page = item"
          >
            {{ item }}
          </button>
        </template>
        <button class="lb-page-btn" :disabled="page === totalPages" @click="page++">
          <i class="pi pi-chevron-right" />
        </button>
      </div>
    </template>
  </main>
</template>

<style scoped>
.lb-muted {
  color: var(--p-text-muted-color, #9ca3af);
}

.lb-card {
  border: 1px solid var(--card-border);
  border-radius: 1.5rem;
  background: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.lb-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 3rem 1rem;
  text-align: center;
  color: var(--p-text-muted-color, #9ca3af);
}

/* Stats */
.lb-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 1rem;
}

.lb-stat {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.875rem 1.25rem;
  text-align: left;
  color: inherit;
  font: inherit;
}

.lb-stat__label {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--p-text-muted-color, #9ca3af);
}

.lb-stat__value {
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1.2;
  display: flex;
  align-items: baseline;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.lb-stat__sub {
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--p-text-muted-color, #9ca3af);
}

.lb-stat--me {
  cursor: pointer;
  border-color: color-mix(in srgb, var(--p-primary-color, #10b981) 50%, var(--card-border));
  transition: border-color 0.15s, transform 0.15s;
}

.lb-stat--me:hover {
  border-color: var(--p-primary-color, #10b981);
  transform: translateY(-1px);
}

/* Toolbar */
.lb-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.lb-search {
  flex: 0 1 260px;
}

.lb-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  font-size: 0.75rem;
  color: var(--p-text-muted-color, #9ca3af);
}

.lb-legend__item {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
}

.lb-legend__dot {
  display: inline-block;
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 999px;
  flex-shrink: 0;
}

/* List */
.lb-list {
  overflow: hidden;
}

.lb-row {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  padding: 0.375rem 1rem;
  border-bottom: 1px solid var(--card-border);
}

.lb-row:last-child {
  border-bottom: none;
}

.lb-row:hover {
  background: var(--color-background-mute);
}

.lb-rank {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--p-text-muted-color, #9ca3af);
  width: 1.5rem;
  text-align: right;
  flex-shrink: 0;
}

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

.lb-bar-wrap {
  flex: 1;
  min-width: 0;
}

.lb-pct {
  font-size: 0.8rem;
  font-weight: 600;
  width: 3.5rem;
  text-align: right;
  flex-shrink: 0;
  color: var(--p-text-muted-color, #9ca3af);
}

/* Progress bar */
.lb-bar {
  display: flex;
  height: 8px;
  border-radius: 999px;
  overflow: hidden;
  background: var(--color-background-mute);
}

.lb-bar__seg {
  height: 100%;
  transition: width 0.4s ease;
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
  background: var(--p-primary-color, #10b981) !important;
  border-color: var(--p-primary-color, #10b981) !important;
  color: #fff;
}

.lb-page-gap {
  min-width: 1.5rem;
  line-height: 2rem;
  text-align: center;
  font-size: 0.8rem;
  color: var(--p-text-muted-color, #9ca3af);
}

.lb-page-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

/* Mobile */
@media (max-width: 640px) {
  .lb-stats {
    grid-template-columns: repeat(2, 1fr);
    gap: 0.5rem;
  }

  .lb-stat {
    padding: 0.75rem 1rem;
  }

  .lb-stat--me {
    grid-column: 1 / -1;
  }

  .lb-stat__value {
    font-size: 1.25rem;
  }

  .lb-search {
    flex: 1 1 100%;
  }
}
</style>
