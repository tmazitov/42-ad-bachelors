import { ref, watch, type Ref } from 'vue'
import { api } from '@/services/api'
import type { ProjectSearchResult } from '@/models/Search'

// Searches 42cursus projects by name. The backend keeps the catalogue in
// memory, so a short debounce is enough.
export function useProjectSearch(query: Ref<string>) {
  const results = ref<ProjectSearchResult[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let abortController: AbortController | null = null

  watch(query, (val) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    abortController?.abort()
    abortController = null

    const trimmed = val.trim()
    if (!trimmed) {
      results.value = []
      error.value = null
      loading.value = false
      return
    }

    debounceTimer = setTimeout(() => {
      if (trimmed === query.value.trim()) search(trimmed)
    }, 250)
  })

  async function search(q: string) {
    abortController?.abort()
    abortController = new AbortController()
    const { signal } = abortController

    loading.value = true
    try {
      const data = await api<{ projects: ProjectSearchResult[] }>(
        `/projects?${new URLSearchParams({ search: q })}`,
        { signal },
      )
      results.value = data.projects
      error.value = null
    } catch (e) {
      if (signal.aborted) return
      results.value = []
      error.value = e instanceof Error ? e.message : 'Search failed'
    } finally {
      if (!signal.aborted) loading.value = false
    }
  }

  function reset() {
    if (debounceTimer) clearTimeout(debounceTimer)
    abortController?.abort()
    abortController = null
    results.value = []
    error.value = null
    loading.value = false
  }

  return { results, loading, error, reset }
}
