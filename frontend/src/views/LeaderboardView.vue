<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Button, InputText, IconField, InputIcon } from 'primevue'
import { useAuthStore } from '@/stores/auth'
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

const SEGMENTS: { key: SegmentKey; label: string; short: string; color: string }[] = [
  { key: 'general',  label: 'General',  short: 'GEN', color: '#22c55e' },
  { key: 'standard', label: 'Standard', short: 'STD', color: '#3b82f6' },
  { key: 'stream',   label: 'Stream',   short: 'STR', color: '#8b5cf6' },
  { key: 'elective', label: 'Elective', short: 'ELE', color: '#f97316' },
]

// General credits are not trackable through the 42 API, so they are always 0.
const CHIP_SEGMENTS = SEGMENTS.filter(seg => seg.key !== 'general')

const MEDALS: Record<number, string> = { 1: '#eab308', 2: '#94a3b8', 3: '#c2703d' }

const router = useRouter()
const auth = useAuthStore()

const data = ref<LeaderboardData | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const page = ref(1)
const search = ref('')

const students = computed(() => data.value?.students ?? [])
const query = computed(() => search.value.trim().toLowerCase())

// Podium shows the top 3 unless searching; the list shows everyone else.
const podium = computed(() => (query.value ? [] : students.value.slice(0, 3)))
// Display order: 2nd, 1st, 3rd
const podiumOrdered = computed(() =>
  [podium.value[1], podium.value[0], podium.value[2]].filter((s): s is LeaderboardStudent => !!s),
)

const listed = computed(() => {
  if (query.value) {
    return students.value.filter(
      s => s.login.toLowerCase().includes(query.value) || s.display_name.toLowerCase().includes(query.value),
    )
  }
  return students.value.slice(podium.value.length)
})

const totalPages = computed(() => Math.ceil(listed.value.length / PAGE_SIZE))
const pageStudents = computed(() => {
  const start = (page.value - 1) * PAGE_SIZE
  return listed.value.slice(start, start + PAGE_SIZE)
})

watch(query, () => { page.value = 1 })

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
  if (!auth.token) {
    router.replace('/')
    return
  }
  try {
    const res = await fetch(`${import.meta.env.VITE_BACKEND_URL || ''}/api/v1/leaderboard`)
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

      <!-- Podium -->
      <div v-if="podiumOrdered.length" class="lb-podium">
        <router-link
          v-for="s in podiumOrdered"
          :id="`lb-${s.login}`"
          :key="s.login"
          :to="`/profile/${s.login}`"
          class="lb-card lb-podium__item"
          :class="[`lb-podium__item--${s.rank}`, { 'lb-is-me': s.login === me?.login }]"
          :style="{ '--medal': MEDALS[s.rank] ?? 'var(--card-border)' }"
        >
          <span class="lb-podium__medal">
            <i v-if="s.rank === 1" class="pi pi-crown" />
            <template v-else>{{ s.rank }}</template>
          </span>
          <StudentAvatar :login="s.login" :image="s.image" :size="s.rank === 1 ? 76 : 60" :ring="MEDALS[s.rank]" />
          <div class="lb-podium__name">
            <span class="font-semibold truncate">{{ s.login }}</span>
            <span class="lb-muted text-xs truncate">{{ s.display_name }}</span>
          </div>
          <span class="lb-podium__pct">{{ s.percentage.toFixed(1) }}%</span>
          <div class="lb-bar w-full">
            <div
              v-for="seg in SEGMENTS"
              :key="seg.key"
              class="lb-bar__seg"
              :style="{ width: segWidth(s, seg.key), background: seg.color }"
            />
          </div>
          <span class="lb-muted text-xs">{{ s.total }} / {{ TOTAL }} credits</span>
        </router-link>
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
        <router-link
          v-for="s in pageStudents"
          :id="`lb-${s.login}`"
          :key="s.login"
          :to="`/profile/${s.login}`"
          class="lb-row"
          :class="{ 'lb-is-me': s.login === me?.login }"
        >
          <span class="lb-rank" :style="MEDALS[s.rank] ? { color: MEDALS[s.rank] } : undefined">{{ s.rank }}</span>

          <StudentAvatar :login="s.login" :image="s.image" :size="40" />

          <div class="lb-row__name">
            <span class="font-medium truncate">
              {{ s.login }}
              <span v-if="s.login === me?.login" class="lb-you">you</span>
            </span>
            <span class="lb-muted text-xs truncate">{{ s.display_name }}</span>
          </div>

          <div class="lb-row__progress">
            <div class="lb-bar">
              <div
                v-for="seg in SEGMENTS"
                :key="seg.key"
                class="lb-bar__seg"
                :style="{ width: segWidth(s, seg.key), background: seg.color }"
              />
            </div>
            <div class="lb-chips">
              <span
                v-for="seg in CHIP_SEGMENTS"
                :key="seg.key"
                class="lb-chip"
                :class="{ 'lb-chip--zero': !s[seg.key] }"
                :title="`${seg.label}: ${s[seg.key]} credits`"
              >
                <span class="lb-legend__dot" :style="{ background: seg.color }" />{{ s[seg.key] }}
              </span>
            </div>
          </div>

          <div class="lb-row__score">
            <span class="lb-row__pct">{{ s.percentage.toFixed(1) }}%</span>
            <span class="lb-muted text-xs">{{ s.total }}/{{ TOTAL }}</span>
          </div>
        </router-link>
      </div>

      <div v-else-if="query" class="lb-card lb-empty lb-muted text-sm">
        No students match “{{ search }}”.
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="lb-pagination">
        <button class="lb-page-btn" :disabled="page === 1" @click="page--">
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

.lb-is-me {
  border-color: var(--p-primary-color, #10b981) !important;
  background: color-mix(in srgb, var(--p-primary-color, #10b981) 8%, var(--card-bg));
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

/* Podium */
.lb-podium {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  align-items: end;
  gap: 1rem;
}

.lb-podium__item {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.625rem;
  padding: 1.5rem 1rem 1.125rem;
  border-top: 4px solid var(--medal);
  text-decoration: none;
  color: inherit;
  min-width: 0;
  transition: transform 0.15s, box-shadow 0.15s;
}

.lb-podium__item:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.08);
}

.lb-podium__item--1 {
  padding-top: 2.25rem;
  background: linear-gradient(180deg, color-mix(in srgb, var(--medal) 12%, var(--card-bg)), var(--card-bg) 60%);
}

.lb-podium__medal {
  position: absolute;
  top: 0.625rem;
  left: 0.875rem;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 1.75rem;
  height: 1.75rem;
  border-radius: 999px;
  background: var(--medal);
  color: #fff;
  font-size: 0.8rem;
  font-weight: 700;
}

.lb-podium__name {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 0;
  max-width: 100%;
}

.lb-podium__pct {
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1;
}

.lb-podium__item--1 .lb-podium__pct {
  font-size: 1.875rem;
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
  padding: 0.75rem 1.25rem;
  border-bottom: 1px solid var(--card-border);
  text-decoration: none;
  color: inherit;
  transition: background 0.15s;
}

.lb-row:last-child {
  border-bottom: none;
}

.lb-row:hover {
  background: var(--color-background-mute);
}

.lb-row.lb-is-me {
  border-color: var(--card-border) !important;
  box-shadow: inset 3px 0 0 var(--p-primary-color, #10b981);
}

.lb-rank {
  width: 1.75rem;
  text-align: center;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--p-text-muted-color, #9ca3af);
  flex-shrink: 0;
  font-variant-numeric: tabular-nums;
}

.lb-row__name {
  display: flex;
  flex-direction: column;
  width: 9rem;
  min-width: 0;
  flex-shrink: 0;
}

.lb-you {
  margin-left: 0.25rem;
  padding: 0 0.375rem;
  border-radius: 999px;
  background: var(--p-primary-color, #10b981);
  color: #fff;
  font-size: 0.625rem;
  font-weight: 600;
  text-transform: uppercase;
  vertical-align: middle;
}

.lb-row__progress {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.lb-chips {
  display: flex;
  gap: 0.375rem;
  flex-wrap: wrap;
}

.lb-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0 0.4rem;
  border-radius: 999px;
  border: 1px solid var(--card-border);
  font-size: 0.68rem;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.lb-chip--zero {
  opacity: 0.45;
}

.lb-row__score {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  width: 4rem;
  flex-shrink: 0;
}

.lb-row__pct {
  font-weight: 700;
  font-variant-numeric: tabular-nums;
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

.lb-page-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

/* Mobile */
@media (max-width: 640px) {
  .lb-podium {
    gap: 0.5rem;
  }

  .lb-podium__item {
    padding: 1.25rem 0.5rem 0.875rem;
  }

  .lb-podium__item--1 {
    padding-top: 1.75rem;
  }

  .lb-podium__pct,
  .lb-podium__item--1 .lb-podium__pct {
    font-size: 1.125rem;
  }

  .lb-podium__medal {
    top: 0.375rem;
    left: 0.375rem;
    width: 1.375rem;
    height: 1.375rem;
    font-size: 0.7rem;
  }

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

  .lb-row {
    gap: 0.5rem;
    padding: 0.75rem 0.875rem;
  }

  .lb-rank {
    width: 1.25rem;
  }

  .lb-row__name {
    width: 5.5rem;
  }

  .lb-row__score {
    width: 3.25rem;
  }

  .lb-chips {
    display: none;
  }

  .lb-search {
    flex: 1 1 100%;
  }
}
</style>
