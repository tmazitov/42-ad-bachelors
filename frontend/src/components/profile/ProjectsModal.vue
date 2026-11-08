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
    :style="{ width: '780px', maxWidth: '95vw' }"
    @update:visible="emit('update:visible', $event)"
  >
    <template #header>
      <div class="flex flex-col sm:flex-row sm:items-center w-full gap-2 sm:gap-4 pr-2">
        <span class="font-semibold text-lg leading-tight">{{ label }}</span>

        <div class="flex items-center gap-2 sm:ml-auto flex-wrap">
          <InputText
            v-model="search"
            placeholder="Search by name…"
            size="small"
            class="w-full sm:w-44"
          />
          <span class="text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">
            {{ earned.toFixed(1) }} / {{ max }} {{ unit }}
          </span>
          <div class="flex-1 sm:w-28 h-2 bg-gray-100 dark:bg-gray-700 rounded-full overflow-hidden" style="min-width: 4rem">
            <div
              class="h-full rounded-full transition-all duration-500"
              :style="{ width: `${progressPercent}%`, background: color }"
            />
          </div>
        </div>
      </div>
    </template>

    <ProjectsTable :projects="projects" :completed-projects="completedProjects" :search="search" />
  </Dialog>
</template>
