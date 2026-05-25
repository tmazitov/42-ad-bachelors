<script setup lang="ts">
import { computed, ref } from 'vue'
import { Dialog, DataTable, Column, Tag } from 'primevue'
import type { Project } from '@/models/Project'
import type { ProjectUser } from '@/composables/useProjects'
import { isCompleted } from '@/composables/useProgress'

const props = defineProps<{
  visible: boolean
  label: string
  color: string
  earned: number
  max: number
  unit: string
  projects: Project[]
  completedProjects: ProjectUser[]
}>()

const emit = defineEmits<{ 'update:visible': [value: boolean] }>()

const completedIds = computed(() => new Set(props.completedProjects.map((p) => p.project.id)))

const rows = computed(() =>
  props.projects.map((p) => ({
    code: p.code,
    name: p.name,
    credit: p.credit,
    done: isCompleted(p, completedIds.value),
    trackable: p.id !== null,
  })),
)

const progressPercent = computed(() => Math.min((props.earned / props.max) * 100, 100))

const sortField = ref('name')
const sortOrder = ref<1 | -1>(1)
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

        <div class="flex items-center gap-2 sm:ml-auto">
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

    <DataTable
      :value="rows"
      :sort-field="sortField"
      :sort-order="sortOrder"
      scrollable
      scroll-height="flex"
      size="small"
      @sort="(e) => { sortField = e.sortField as string; sortOrder = e.sortOrder as 1 | -1 }"
    >
      <Column field="code" header="Code" class="hidden sm:table-cell" style="width: 7.5rem; font-family: monospace" />
      <Column field="name" header="Name" :sortable="true" style="min-width: 8rem" />
      <Column field="credit" header="Cr." :sortable="true" style="width: 4.5rem">
        <template #body="{ data }">
          <span :class="data.credit > 0 ? 'font-medium' : 'text-gray-300'">
            {{ data.credit > 0 ? data.credit : '—' }}
          </span>
        </template>
      </Column>
      <Column field="done" header="Status" :sortable="true" style="width: 5.5rem">
        <template #body="{ data }">
          <Tag v-if="data.done" value="Done" severity="success" />
          <span v-else-if="!data.trackable" class="text-gray-300 text-xs">N/A</span>
        </template>
      </Column>
    </DataTable>
  </Dialog>
</template>
