<script setup lang="ts">
import { watch, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import type { FtUser } from '@/stores/auth'
import { useProjects } from '@/composables/useProjects'
import { useMyProgressStore } from '@/stores/myProgress'
import ProgressChart from '@/components/profile/ProgressChart.vue'

const props = defineProps<{ user?: FtUser }>()
const auth = useAuthStore()
const { projects, loading, fetchProjects } = useProjects()
const myProgress = useMyProgressStore()

const effectiveUser = computed(() => props.user ?? auth.user)
const isOwnProfile = computed(() => !props.user || props.user.id === auth.user?.id)

// The signed-in student's projects come from the shared store (also used by
// the header menu); anyone else's are fetched here.
const shownProjects = computed(() => (isOwnProfile.value ? myProgress.projects : projects.value))
// Also loading while the signed-in user itself is still being fetched.
const isLoading = computed(() => {
  const user = effectiveUser.value
  if (!user) return true
  if (isOwnProfile.value) return myProgress.loadedFor !== user.id && !myProgress.error
  return loading.value
})

watch(
  effectiveUser,
  (user) => {
    if (!user) return
    if (isOwnProfile.value) myProgress.load(user.id)
    else fetchProjects(user.id)
  },
  { immediate: true },
)

function markColor(mark: number | null): string {
  if (mark === null) return 'text-gray-400'
  if (mark >= 100) return 'text-purple-500'
  if (mark >= 75) return 'text-green-500'
  if (mark >= 50) return 'text-yellow-500'
  return 'text-red-500'
}

function formatDate(iso: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })
}
</script>

<template>
  <section class="flex flex-col gap-4">

    <ProgressChart
      :completed-projects="shownProjects"
      :loading="isLoading"
      :title="isOwnProfile ? 'Your Progress' : `${effectiveUser?.login}'s Progress`"
    />

  </section>
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

.project-card {
  display: flex;
  flex-direction: column;
  padding: 0.75rem;
  border: 1px solid var(--p-gray-200);
  border-radius: 1rem;
  background: var(--p-surface-card);
  min-height: 5.5rem;
}
</style>
