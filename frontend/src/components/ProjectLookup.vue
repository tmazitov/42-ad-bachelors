<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { apiFetch } from '@/services/apiFetch'

interface ApiProject {
  id: number
  name: string
  slug: string
}

const auth = useAuthStore()
const rows = ref<ApiProject[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

async function fetchPage(page: number): Promise<ApiProject[]> {
  const res = await apiFetch(
    `${import.meta.env.VITE_42_API_BASE || ''}/v2/cursus/21/projects?per_page=100&page=${page}&sort=name`,
  )
  if (!res.ok) throw new Error(`${res.status}`)
  return res.json()
}

onMounted(async () => {
  if (!auth.token) return
  loading.value = true
  error.value = null
  try {
    let page = 1
    while (true) {
      const batch = await fetchPage(page)
      if (batch.length === 0) break
      rows.value.push(...batch)
      if (batch.length < 100) break
      page++
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed'
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="custom-card flex-col items-start gap-3">
    <h2 class="text-xl font-semibold">42 Project ID Lookup</h2>

    <div v-if="loading" class="flex items-center gap-2 text-gray-500 dark:text-gray-400">
      <i class="pi pi-spin pi-spinner" />
      <span>Fetching… ({{ rows.length }} loaded)</span>
    </div>

    <div v-else-if="error" class="text-red-500 text-sm">{{ error }}</div>

    <div v-else class="w-full overflow-auto max-h-[60vh]">
      <table class="w-full text-sm border-collapse">
        <thead class="sticky top-0 bg-white dark:bg-gray-800">
          <tr class="border-b border-gray-200 dark:border-gray-700 text-left">
            <th class="py-1 pr-4 font-semibold w-16">ID</th>
            <th class="py-1 pr-4 font-semibold">Name</th>
            <th class="py-1 font-semibold text-gray-400 dark:text-gray-500">Slug</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="p in rows"
            :key="p.id"
            class="border-b border-gray-100 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800"
          >
            <td class="py-1 pr-4 font-mono text-gray-500 dark:text-gray-400">{{ p.id }}</td>
            <td class="py-1 pr-4">{{ p.name }}</td>
            <td class="py-1 text-gray-400 dark:text-gray-500 font-mono text-xs">{{ p.slug }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <span v-if="!loading" class="text-xs text-gray-400">{{ rows.length }} projects total</span>
  </div>
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
</style>
