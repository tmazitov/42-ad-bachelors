<script setup lang="ts">
import { computed, ref } from 'vue'
import { Doughnut } from 'vue-chartjs'
import { Chart as ChartJS, ArcElement, Tooltip } from 'chart.js'
import type { Project } from '@/models/Project'
import type { StudentProject } from '@/models/Student'
import { useProgress, CREDITS_REQUIRED } from '@/composables/useProgress'
import {
  generalEducationModules,
  standardModules,
  streamModules,
  electiveModules,
} from '@/utils/data'
import ProjectsModal from '@/components/profile/ProjectsModal.vue'
import StreamModal from '@/components/profile/StreamModal.vue'

ChartJS.register(ArcElement, Tooltip)

const props = defineProps<{
  completedProjects: StudentProject[]
  title?: string
  loading?: boolean
  stream?: string | null // chosen stream key
  own?: boolean // the signed-in student's own progress (can pick a stream, mark modules)
  markedModules?: string[] // self-reported modules marked completed
}>()

const projectsRef = computed(() => props.completedProjects)
const streamRef = computed(() => props.stream ?? null)
const markedRef = computed(() => props.markedModules ?? [])
const progress = useProgress(projectsRef, streamRef, markedRef)

type ProgressKey = 'general' | 'standard' | 'stream' | 'elective'

interface CategoryDef {
  key: ProgressKey
  label: string
  color: string
  max: number
  unit: string
  description: string
  projects: Project[]
}

const CATEGORIES: CategoryDef[] = [
  {
    key: 'general',
    label: 'General Education',
    color: '#22c55e',
    max: CREDITS_REQUIRED.general,
    unit: 'credits',
    description: "University courses on general skills (GEN 101, 201, 301). They are not 42 projects, so they can't be tracked here.",
    projects: generalEducationModules.projects,
  },
  {
    key: 'standard',
    label: 'Standard Modules',
    color: '#3b82f6',
    max: CREDITS_REQUIRED.standard,
    unit: 'credits',
    description: 'Required for every student: the 42 common core (Libft to ft_transcendence), plus AI and data science projects (Python for Data Science to Leaffliction), maths projects (computorv1, matrix), ft_resume, the internship and the capstone.',
    projects: standardModules.projects,
  },
  {
    key: 'stream',
    label: 'Stream Modules',
    color: '#8b5cf6',
    max: CREDITS_REQUIRED.stream,
    unit: 'credits',
    description: "Your specialisation. Pick one stream; only its modules count, and credits from different streams don't add up.",
    projects: streamModules.projects,
  },
  {
    key: 'elective',
    label: 'Elective Modules',
    color: '#f97316',
    max: CREDITS_REQUIRED.elective,
    unit: 'credits',
    description: 'Free-choice projects outside the core and your stream. Any of them counts, up to 12 credits.',
    projects: electiveModules.projects,
  },
]

const selectedCat = ref<CategoryDef | null>(null)
const modalVisible = ref(false)
const streamModalVisible = ref(false)

function openModal(cat: CategoryDef) {
  if (props.loading) return
  if (cat.key === 'stream') {
    streamModalVisible.value = true
    return
  }
  selectedCat.value = cat
  modalVisible.value = true
}

function categoryLabel(cat: CategoryDef) {
  return cat.key === 'stream' ? `Stream: ${progress.value.countedStream.name}` : cat.label
}

const chartData = computed(() => {
  const p = progress.value
  const remaining = Math.max(CREDITS_REQUIRED.total - p.total, 0)
  return {
    labels: [...CATEGORIES.map((c) => c.label), 'Remaining'],
    datasets: [{
      data: [p.general, p.standard, p.stream, p.elective, remaining],
      backgroundColor: [...CATEGORIES.map((c) => c.color), '#e5e7eb'],
      borderWidth: 0,
      borderRadius: 3,
      spacing: 2,
    }],
  }
})

// External tooltip rendered as DOM element so it sits above the center overlay
const tooltipEl = ref<HTMLElement | null>(null)

function externalTooltip({ tooltip }: { chart: any; tooltip: any }) {
  const el = tooltipEl.value
  if (!el) return

  if (tooltip.opacity === 0) {
    el.style.opacity = '0'
    return
  }

  const title = tooltip.title?.[0] ?? ''
  const body = tooltip.body?.[0]?.lines?.[0] ?? ''
  const color = tooltip.labelColors?.[0]?.backgroundColor ?? '#fff'

  el.innerHTML = `
    <div class="tt-title">${title}</div>
    <div class="tt-body"><span class="tt-swatch" style="background:${color}"></span>${body}</div>
  `

  el.style.opacity = '1'
  el.style.left = tooltip.caretX + 'px'
  el.style.top = tooltip.caretY + 'px'
}

const chartOptions = {
  responsive: true,
  maintainAspectRatio: true,
  cutout: '72%',
  plugins: {
    legend: { display: false },
    tooltip: {
      enabled: false,
      external: externalTooltip,
    },
  },
}
</script>

<template>
  <div class="custom-card flex-col items-start gap-4">
    <h2 class="text-xl font-semibold">{{ title ?? 'Your Progress' }}</h2>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col sm:flex-row items-center gap-6 w-full" aria-busy="true">
      <div class="relative w-44 h-44 shrink-0 flex items-center justify-center">
        <div class="chart-loader" />
        <span class="absolute text-xs text-gray-400 dark:text-gray-500">Loading…</span>
      </div>
      <div class="flex flex-col gap-2 w-full text-sm">
        <div v-for="cat in CATEGORIES" :key="cat.key" class="flex items-center gap-2 px-2 py-1 -mx-2">
          <span class="w-2.5 h-2.5 rounded-full shrink-0 opacity-40" :style="{ background: cat.color }" />
          <span class="w-36 text-gray-400 dark:text-gray-500 shrink-0">{{ cat.label }}</span>
          <div class="flex-1 h-1.5 rounded-full skeleton" />
          <span class="w-16 flex justify-end shrink-0"><span class="h-3 w-10 rounded skeleton" /></span>
        </div>
      </div>
    </div>


    <div v-else class="flex flex-col sm:flex-row items-center gap-6 w-full">

      <!-- Doughnut -->
      <div class="relative w-44 h-44 shrink-0">
        <Doughnut :data="chartData" :options="chartOptions" />
        <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
          <span class="text-3xl font-bold leading-none">{{ Math.round(progress.percentage) }}%</span>
          <span class="text-xs text-gray-400 dark:text-gray-500 mt-1">{{ Math.round(progress.total) }} / {{ CREDITS_REQUIRED.total }}</span>
          <span class="text-xs text-gray-400 dark:text-gray-500">credits</span>
        </div>
        <div ref="tooltipEl" class="chart-tooltip" />
      </div>

      <!-- Legend / breakdown -->
      <div class="flex flex-col gap-2 w-full text-sm">
        <div
          v-for="cat in CATEGORIES"
          :key="cat.key"
          class="flex items-center gap-2 cursor-pointer rounded-lg px-2 py-1 -mx-2 hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
          @click="openModal(cat)"
        >
          <span class="w-2.5 h-2.5 rounded-full shrink-0" :style="{ background: cat.color }" />
          <span class="w-36 text-gray-600 dark:text-gray-300 shrink-0 truncate" :title="categoryLabel(cat)">{{ categoryLabel(cat) }}</span>
          <div class="flex-1 h-1.5 rounded-full bg-gray-100 dark:bg-gray-700 overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500"
              :style="{
                width: `${Math.min((progress[cat.key] / cat.max) * 100, 100)}%`,
                background: cat.color,
              }"
            />
          </div>
          <span class="text-gray-400 dark:text-gray-500 text-xs w-16 text-right shrink-0">
            {{ progress[cat.key].toFixed(1) }} / {{ cat.max }}
          </span>
        </div>
      </div>

    </div>
  </div>

  <StreamModal
    v-model:visible="streamModalVisible"
    :own="own ?? false"
    :chosen="stream ?? null"
    :counted="progress.countedStream"
    :stream-credits="progress.streamCredits"
    :completed-projects="completedProjects"
    color="#8b5cf6"
    :description="CATEGORIES.find((c) => c.key === 'stream')?.description"
  />

  <ProjectsModal
    v-if="selectedCat"
    v-model:visible="modalVisible"
    :label="selectedCat.label"
    :color="selectedCat.color"
    :earned="progress[selectedCat.key]"
    :max="selectedCat.max"
    :unit="selectedCat.unit"
    :description="selectedCat.description"
    :projects="selectedCat.projects"
    :completed-projects="completedProjects"
    :marked-modules="markedModules ?? []"
    :own="own ?? false"
  />
</template>

<style scoped>
.custom-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border: 1px solid var(--card-border);
  border-radius: 1.5rem;
  background: var(--card-bg);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}

.chart-loader {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  border: 12px solid var(--card-border);
  border-top-color: var(--p-primary-color, #10b981);
  animation: chart-spin 0.9s linear infinite;
}

@keyframes chart-spin {
  to { transform: rotate(360deg); }
}

.skeleton {
  background: var(--card-border);
  animation: skeleton-pulse 1.2s ease-in-out infinite;
}

@keyframes skeleton-pulse {
  50% { opacity: 0.45; }
}

.chart-tooltip {
  position: absolute;
  z-index: 10;
  background: rgba(0, 0, 0, 0.8);
  color: #fff;
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  pointer-events: none;
  opacity: 0;
  transition: opacity 0.15s;
  transform: translate(-50%, calc(-100% - 8px));
  white-space: nowrap;
}

.chart-tooltip :deep(.tt-title) {
  font-weight: 600;
  margin-bottom: 2px;
}

.chart-tooltip :deep(.tt-body) {
  display: flex;
  align-items: center;
  gap: 6px;
}

.chart-tooltip :deep(.tt-swatch) {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}
</style>
