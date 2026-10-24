<script setup lang="ts">
import { computed, ref } from 'vue'
import { Dialog, DataTable, Column, InputText } from 'primevue'
import type { Project, Team } from '@/models/Project'
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

const completedIds = computed(() => new Set(props.completedProjects.filter((p) => p['validated?']).map((p) => p.project.id)))

const slugMap = computed(() => {
  const map = new Map<number, string>()
  for (const pu of props.completedProjects) {
    if (pu.project?.slug) map.set(pu.project.id, pu.project.slug)
  }
  return map
})

const rows = computed(() =>
  props.projects.map((p) => {
    const ids = Array.isArray(p.id) ? p.id : p.id !== null ? [p.id] : []
    const slug = ids.reduce<string | null>((found, id) => found ?? slugMap.value.get(id) ?? null, null)
    const primaryId = ids[0] ?? null
    const url = primaryId !== null
      ? `https://projects.intra.42.fr/projects/${slug ?? primaryId}`
      : null
    return {
      code: p.code,
      name: p.name,
      credit: p.credit,
      team: p.team,
      done: isCompleted(p, completedIds.value),
      trackable: p.id !== null,
      url,
    }
  }),
)

const TEAM_LABELS: Record<Team, { label: string; icon: string }> = {
  solo: { label: 'Solo', icon: 'pi pi-user' },
  group: { label: 'Group', icon: 'pi pi-users' },
  mixed: { label: 'Solo / Group', icon: 'pi pi-user-plus' },
}

const progressPercent = computed(() => Math.min((props.earned / props.max) * 100, 100))

const search = ref('')

const filteredRows = computed(() => {
  const q = search.value.trim().toLowerCase()
  return q ? rows.value.filter((r) => r.name.toLowerCase().includes(q) || r.code.toLowerCase().includes(q)) : rows.value
})

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

    <DataTable
      :value="filteredRows"
      :sort-field="sortField"
      :sort-order="sortOrder"
      scrollable
      scroll-height="flex"
      size="small"
      @sort="(e) => { sortField = e.sortField as string; sortOrder = e.sortOrder as 1 | -1 }"
    >
      <Column field="code" header="Code" class="hidden sm:table-cell" style="width: 7.5rem; font-family: monospace" />
      <Column field="name" header="Name" :sortable="true" style="min-width: 8rem">
        <template #body="{ data }">
          <a
            v-if="data.url"
            :href="data.url"
            target="_blank"
            rel="noopener noreferrer"
            class="project-link"
          >{{ data.name }}</a>
          <span v-else>{{ data.name }}</span>
        </template>
      </Column>
      <Column field="team" header="Type" :sortable="true" style="width: 8rem">
        <template #body="{ data }">
          <span v-if="data.team" class="team-badge" :class="`team-badge--${data.team}`">
            <i :class="TEAM_LABELS[data.team as Team].icon" />
            {{ TEAM_LABELS[data.team as Team].label }}
          </span>
          <span v-else class="text-gray-300 text-xs">—</span>
        </template>
      </Column>
      <Column field="credit" header="Cr." :sortable="true" style="width: 4.5rem">
        <template #body="{ data }">
          <span :class="data.credit > 0 ? 'font-medium' : 'text-gray-300'">
            {{ data.credit > 0 ? data.credit : '—' }}
          </span>
        </template>
      </Column>
      <Column field="done" header="Status" :sortable="true" style="width: 5.5rem">
        <template #body="{ data }">
          <span v-if="data.done">✅</span>
          <span v-else-if="!data.trackable" class="text-gray-300 text-xs">N/A</span>
        </template>
      </Column>
    </DataTable>
  </Dialog>
</template>

<style scoped>
.team-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 500;
  white-space: nowrap;
  border: 1px solid var(--card-border);
}

.team-badge i {
  font-size: 0.7rem;
}

.team-badge--solo {
  color: #0ea5e9;
  border-color: color-mix(in srgb, #0ea5e9 35%, transparent);
  background: color-mix(in srgb, #0ea5e9 8%, transparent);
}

.team-badge--group {
  color: #a855f7;
  border-color: color-mix(in srgb, #a855f7 35%, transparent);
  background: color-mix(in srgb, #a855f7 8%, transparent);
}

.team-badge--mixed {
  color: var(--p-text-muted-color, #9ca3af);
}

.project-link {
  color: inherit;
  text-decoration: none;
}

.project-link:hover {
  text-decoration: underline;
  opacity: 0.8;
}
</style>
