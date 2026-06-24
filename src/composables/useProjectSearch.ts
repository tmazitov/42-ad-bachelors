import { ref, watch, type Ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { apiFetch } from '@/services/apiFetch'

export interface FtProject {
  id: number
  name: string
  slug: string
  exam: boolean
}

const FETCH_SIZE = 15
const DISPLAY_SIZE = 5

export function useProjectSearch(query: Ref<string>) {
  const auth = useAuthStore()
  const results = ref<FtProject[]>([])
  const loading = ref(false)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let abortController: AbortController | null = null

  watch(query, (val) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    if (abortController) {
      abortController.abort()
      abortController = null
    }

    const trimmed = val.trim()
    if (!trimmed) {
      results.value = []
      loading.value = false
      return
    }

    debounceTimer = setTimeout(() => {
      if (trimmed === query.value.trim()) search(trimmed)
    }, 500)
  })

  async function search(q: string) {
    if (!auth.token) return

    if (abortController) abortController.abort()
    abortController = new AbortController()
    const { signal } = abortController

    loading.value = true
    try {
      const res = await apiFetch(
        `${import.meta.env.VITE_42_API_BASE || ''}/v2/cursus/21/projects?search[name]=${encodeURIComponent(q)}&per_page=${FETCH_SIZE}`,
        { signal },
      )
      if (!res.ok) throw new Error(`${res.status}`)
      const data: FtProject[] = await res.json()
      results.value = data.filter(p => !p.exam).slice(0, DISPLAY_SIZE)
    } catch {
      if (!signal.aborted) results.value = []
    } finally {
      if (!signal.aborted) loading.value = false
    }
  }

  function reset() {
    if (debounceTimer) clearTimeout(debounceTimer)
    if (abortController) {
      abortController.abort()
      abortController = null
    }
    results.value = []
    loading.value = false
  }

  return { results, loading, reset }
}
