<script setup lang="ts">
import { computed, ref } from 'vue'
import { Doughnut } from 'vue-chartjs'
import { Chart as ChartJS, ArcElement, Tooltip } from 'chart.js'
import type { Project } from '@/models/Project'
import type { ProjectUser } from '@/composables/useProjects'
import { useProgress, CREDITS_REQUIRED } from '@/composables/useProgress'
import {
  generalEducationModules,
  standardModules,
  streamModules,
  electiveModules,
} from '@/utils/data'
import ProjectsModal from '@/components/profile/ProjectsModal.vue'

ChartJS.register(ArcElement, Tooltip)

const props = defineProps<{ completedProjects: ProjectUser[] }>()

const projectsRef = computed(() => props.completedProjects)
const progress = useProgress(projectsRef)

type ProgressKey = 'general' | 'standard' | 'stream' | 'elective' | 'capstone' | 'internship'

interface CategoryDef {
  key: ProgressKey
  label: string
  color: string
  max: number
  unit: string
  projects: Project[]
}

const CATEGORIES: CategoryDef[] = [
  {
    key: 'general',
    label: 'General Education',
    color: '#22c55e',
    max: CREDITS_REQUIRED.general,
    unit: 'credits',
    projects: generalEducationModules.projects,
  },
  {
    key: 'standard',
    label: 'Standard Modules',
    color: '#3b82f6',
    max: CREDITS_REQUIRED.standard,
    unit: 'credits',
    projects: standardModules.projects.filter((p) => p.code !== 'BAI 498' && p.code !== 'BAI 499'),
  },
  {
    key: 'stream',
    label: 'Stream Modules',
    color: '#8b5cf6',
    max: CREDITS_REQUIRED.stream,
    unit: 'credits',
    projects: streamModules.projects,
  },
  {
    key: 'elective',
    label: 'Elective Modules',
    color: '#f97316',
    max: CREDITS_REQUIRED.elective,
    unit: 'credits',
    projects: electiveModules.projects,
  },
  {
    key: 'capstone',
    label: 'Capstone Project',
    color: '#eab308',
    max: CREDITS_REQUIRED.capstone,
    unit: 'credits',
    projects: standardModules.projects.filter((p) => p.code === 'BAI 498'),
  },
  {
    key: 'internship',
    label: 'Internship',
    color: '#14b8a6',
    max: CREDITS_REQUIRED.internship,
    unit: 'credits',
    projects: standardModules.projects.filter((p) => p.code === 'BAI 499'),
  },
]

const selectedCat = ref<CategoryDef | null>(null)
const modalVisible = ref(false)

function openModal(cat: CategoryDef) {
  selectedCat.value = cat
  modalVisible.value = true
}

const chartData = computed(() => {
  const p = progress.value
  const remaining = Math.max(CREDITS_REQUIRED.total - p.total, 0)
  return {
    labels: [...CATEGORIES.map((c) => c.label), 'Remaining'],
    datasets: [{
      data: [p.general, p.standard, p.stream, p.elective, p.capstone, p.internship, remaining],
      backgroundColor: [...CATEGORIES.map((c) => c.color), '#e5e7eb'],
      borderWidth: 0,
      borderRadius: 3,
      spacing: 2,
    }],
  }
})

const chartOptions = {
  responsive: true,
  maintainAspectRatio: true,
  cutout: '72%',
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (ctx: { dataIndex: number; parsed: number }) => {
          const cat = CATEGORIES[ctx.dataIndex]
          const max = cat ? cat.max : CREDITS_REQUIRED.total
          return `  ${ctx.parsed.toFixed(1)} / ${max} credits`
        },
      },
    },
  },
}
</script>

<template>
  <div class="custom-card flex-col items-start gap-4">
    <h2 class="text-xl font-semibold ">Your Bachelor's Progress</h2>

    <div class="flex flex-col sm:flex-row items-center gap-6 w-full">

      <!-- Doughnut -->
      <div class="relative w-44 h-44 shrink-0">
        <Doughnut :data="chartData" :options="chartOptions" />
        <div class="absolute inset-0 flex flex-col items-center justify-center pointer-events-none">
          <span class="text-3xl font-bold leading-none">{{ Math.round(progress.percentage) }}%</span>
          <span class="text-xs text-gray-400 mt-1">{{ Math.round(progress.total) }} / {{ CREDITS_REQUIRED.total }}</span>
          <span class="text-xs text-gray-400">credits</span>
        </div>
      </div>

      <!-- Legend / breakdown -->
      <div class="flex flex-col gap-2 w-full text-sm">
        <div
          v-for="cat in CATEGORIES"
          :key="cat.key"
          class="flex items-center gap-2 cursor-pointer rounded-lg px-2 py-1 -mx-2 hover:bg-gray-100 transition-colors"
          @click="openModal(cat)"
        >
          <span class="w-2.5 h-2.5 rounded-full shrink-0" :style="{ background: cat.color }" />
          <span class="w-36 text-gray-600 shrink-0">{{ cat.label }}</span>
          <div class="flex-1 h-1.5 rounded-full bg-gray-100 overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500"
              :style="{
                width: `${Math.min((progress[cat.key] / cat.max) * 100, 100)}%`,
                background: cat.color,
              }"
            />
          </div>
          <span class="text-gray-400 text-xs w-16 text-right shrink-0">
            {{ progress[cat.key].toFixed(1) }} / {{ cat.max }}
          </span>
        </div>
      </div>

    </div>
  </div>

  <ProjectsModal
    v-if="selectedCat"
    v-model:visible="modalVisible"
    :label="selectedCat.label"
    :color="selectedCat.color"
    :earned="progress[selectedCat.key]"
    :max="selectedCat.max"
    :unit="selectedCat.unit"
    :projects="selectedCat.projects"
    :completed-projects="completedProjects"
  />
</template>

<style scoped>
.custom-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 1.25rem;
  border: 1px solid var(--p-gray-200);
  border-radius: 1.5rem;
  background: var(--p-gray-50);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
}
</style>
