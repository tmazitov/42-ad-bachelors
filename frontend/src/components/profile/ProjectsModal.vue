<script setup lang="ts">
import { computed, ref } from 'vue'
import { Dialog, InputText } from 'primevue'
import type { Project } from '@/models/Project'
import type { StudentProject } from '@/models/Student'
import ProjectsTable from '@/components/profile/ProjectsTable.vue'

const props = defineProps<{
  visible: boolean
  label: string
  color: string
  earned: number
  max: number
  unit: string
  description?: string
  projects: Project[]
  completedProjects: StudentProject[]
}>()

const emit = defineEmits<{ 'update:visible': [value: boolean] }>()

const progressPercent = computed(() => Math.min((props.earned / props.max) * 100, 100))

const search = ref('')
</script>

<template>
  <Dialog
    :visible="visible"
    modal
    :draggable="false"
    :pt="{ header: { style: 'align-items: flex-start' } }"
    :style="{ width: '780px', maxWidth: '95vw' }"
    @update:visible="emit('update:visible', $event)"
  >
    <template #header>
      <div class="flex flex-col w-full gap-3 pr-2">
        <div class="flex flex-col gap-1 min-w-0">
          <span class="font-semibold text-lg leading-tight">{{ label }}</span>
          <span v-if="description" class="text-sm text-gray-500 dark:text-gray-400">{{ description }}</span>
        </div>

        <div class="flex items-center gap-2 flex-wrap">
          <InputText
            v-model="search"
            placeholder="Search by name…"
            size="small"
            class="w-full sm:w-56"
          />
          <span class="hidden sm:block flex-1" />
          <div class="flex items-center gap-2 flex-1 sm:flex-none">
            <span class="text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">
              {{ earned.toFixed(1) }} / {{ max }} {{ unit }}
            </span>
            <div class="flex-1 sm:flex-none sm:w-28 h-2 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden" style="min-width: 4rem">
              <div
                class="h-full rounded-full transition-all duration-500"
                :style="{ width: `${progressPercent}%`, background: color }"
              />
            </div>
          </div>
        </div>
      </div>
    </template>

    <ProjectsTable :projects="projects" :completed-projects="completedProjects" :search="search" />
  </Dialog>
</template>
