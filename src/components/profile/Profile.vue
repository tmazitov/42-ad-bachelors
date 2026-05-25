<script setup lang="ts">
import { watch } from 'vue'
import { Button, Tag } from 'primevue'
import { useAuthStore } from '@/stores/auth'
import { useProjects } from '@/composables/useProjects'
import ProgressChart from '@/components/profile/ProgressChart.vue'

const auth = useAuthStore()
const { projects, loading, error, fetchProjects } = useProjects()

watch(
  () => auth.user,
  (user) => { if (user) fetchProjects(user.id) },
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

    <ProgressChart :completed-projects="projects" />

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
