<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Dialog, InputText, Button, Tag } from 'primevue'
import type { StudentProject } from '@/models/Student'
import { STREAMS, type StreamDef } from '@/utils/data'
import { CREDITS_REQUIRED, countedStream } from '@/composables/useProgress'
import { useAuthStore } from '@/stores/auth'
import ProjectsTable from '@/components/profile/ProjectsTable.vue'

// Stream modules: a student picks one stream, and only its modules count.
// On their own profile, a student without a stream is asked to pick one first
// (and can change it later); other profiles are read-only.
const props = defineProps<{
  visible: boolean
  own: boolean
  chosen: string | null
  counted: StreamDef // the stream whose credits count (chosen, or best)
  streamCredits: Record<string, number>
  completedProjects: StudentProject[]
  color: string
  description?: string
}>()

const emit = defineEmits<{ 'update:visible': [value: boolean] }>()

const auth = useAuthStore()
const MAX = CREDITS_REQUIRED.stream

const changing = ref(false)
const saving = ref<string | null>(null)
const error = ref<string | null>(null)
const search = ref('')

watch(() => props.visible, (open) => {
  if (!open) {
    changing.value = false
    error.value = null
    search.value = ''
  }
})

const picking = computed(() => props.own && (!props.chosen || changing.value))
const recommended = computed(() => countedStream(null, props.streamCredits).key)

const earned = computed(() => Math.min(props.streamCredits[props.counted.key] ?? 0, MAX))
const percent = computed(() => (earned.value / MAX) * 100)

function creditsIn(key: string) {
  return Math.min(props.streamCredits[key] ?? 0, MAX)
}

async function choose(key: string) {
  if (key === props.chosen) {
    changing.value = false
    return
  }
  saving.value = key
  error.value = null
  try {
    await auth.setStream(key)
    changing.value = false
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Could not save the stream'
  } finally {
    saving.value = null
  }
}
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
    <!-- Picker -->
    <template v-if="picking" #header>
      <div class="flex flex-col gap-1 pr-2">
        <span class="font-semibold text-lg leading-tight">Choose your stream</span>
        <span class="text-sm text-gray-500 dark:text-gray-400">
          Only the modules of your stream count toward the {{ MAX }} stream credits. You can change it later.
        </span>
      </div>
    </template>

    <!-- Stream table -->
    <template v-else #header>
      <div class="flex flex-col w-full gap-3 pr-2">
        <div class="flex flex-col gap-1 min-w-0">
          <span class="font-semibold text-lg leading-tight">{{ counted.name }}</span>
          <span v-if="description" class="text-sm text-gray-500 dark:text-gray-400">{{ description }}</span>
          <span v-if="!chosen" class="text-xs text-gray-500 dark:text-gray-400">
            No stream chosen yet — counting the stream with the most credits.
          </span>
        </div>

        <div class="flex items-center gap-2 flex-wrap">
          <InputText v-model="search" placeholder="Search by name…" size="small" class="w-full sm:w-56" />
          <Button
            v-if="own"
            label="Change"
            icon="pi pi-sync"
            size="small"
            text
            @click="changing = true"
          />
          <span class="hidden sm:block flex-1" />
          <div class="flex items-center gap-2 flex-1 sm:flex-none">
            <span class="text-sm text-gray-500 dark:text-gray-400 whitespace-nowrap">{{ earned }} / {{ MAX }} credits</span>
            <div class="stream-bar flex-1 sm:flex-none sm:w-28" style="min-width: 4rem">
              <div class="stream-bar__fill" :style="{ width: `${percent}%`, background: color }" />
            </div>
          </div>
        </div>
      </div>
    </template>

    <div v-if="picking" class="flex flex-col gap-3">
      <div class="stream-grid">
        <div
          v-for="s in STREAMS"
          :key="s.key"
          class="stream-card"
          :class="{ 'stream-card--current': s.key === chosen }"
        >
          <div class="stream-card__title">
            <span class="font-semibold">{{ s.name }}</span>
            <Tag v-if="s.key === chosen" value="Current" severity="success" />
            <Tag v-else-if="s.key === recommended && creditsIn(s.key) > 0" value="Most credits" severity="secondary" />
          </div>

          <div class="stream-card__codes">
            <span v-for="p in s.projects" :key="p.code" class="stream-card__code">{{ p.code }}</span>
          </div>

          <div class="flex items-center gap-2">
            <div class="stream-bar flex-1">
              <div class="stream-bar__fill" :style="{ width: `${(creditsIn(s.key) / MAX) * 100}%`, background: color }" />
            </div>
            <span class="text-xs text-gray-500 dark:text-gray-400 whitespace-nowrap">
              {{ creditsIn(s.key) }} / {{ MAX }} already
            </span>
          </div>

          <Button
            :label="s.key === chosen ? 'Keep this stream' : 'Choose'"
            size="small"
            :outlined="s.key !== chosen"
            :loading="saving === s.key"
            :disabled="saving !== null && saving !== s.key"
            @click="choose(s.key)"
          />
        </div>
      </div>

      <p v-if="error" class="text-sm text-red-500">{{ error }}</p>
      <Button
        v-if="chosen"
        label="Cancel"
        text
        size="small"
        class="self-start"
        @click="changing = false"
      />
    </div>

    <ProjectsTable
      v-else
      :projects="counted.projects"
      :completed-projects="completedProjects"
      :search="search"
    />
  </Dialog>
</template>

<style scoped>
.stream-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 0.75rem;
}

.stream-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border: 1px solid var(--card-border);
  border-radius: 1rem;
  background: var(--card-bg);
}

.stream-card__title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-height: 1.75rem;
}

.stream-card__title :deep(.p-tag) {
  padding: 0.1rem 0.45rem;
  font-size: 0.7rem;
}

.stream-card--current {
  border-color: var(--p-primary-color, #10b981);
}

.stream-card__codes {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
}

.stream-card__code {
  padding: 0.05rem 0.45rem;
  border-radius: 999px;
  border: 1px solid var(--card-border);
  font-family: monospace;
  font-size: 0.72rem;
  color: var(--p-text-muted-color, #9ca3af);
}

.stream-bar {
  height: 0.5rem;
  border-radius: 999px;
  overflow: hidden;
  background: var(--card-border);
}

.stream-bar__fill {
  height: 100%;
  border-radius: 999px;
  transition: width 0.4s ease;
}
</style>
