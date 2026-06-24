import { ref, watch, type Ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import type { FtUser } from '@/stores/auth'
import { apiFetch } from '@/services/apiFetch'

const PER_PAGE = 10

export function useStudentSearch(query: Ref<string>) {
  const auth = useAuthStore()
  const results = ref<FtUser[]>([])
  const loading = ref(false)
  const page = ref(1)
  const totalPages = ref(0)
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
      page.value = 1
      totalPages.value = 0
      loading.value = false
      return
    }

    debounceTimer = setTimeout(() => {
      if (trimmed === query.value.trim()) search(trimmed, 1)
    }, 500)
  })

  async function search(q: string, p: number) {
    if (!auth.token) return

    if (abortController) abortController.abort()
    abortController = new AbortController()
    const { signal } = abortController

    loading.value = true
    try {
      const res = await apiFetch(
        `${import.meta.env.VITE_42_API_BASE || ''}/v2/users?search[login]=${encodeURIComponent(q)}&per_page=${PER_PAGE}&page=${p}`,
        { signal },
      )
      if (!res.ok) throw new Error(`${res.status}`)

      const data: FtUser[] = await res.json()
      results.value = data
      page.value = p

      const total = parseInt(res.headers.get('X-Total') ?? '0', 10)
      totalPages.value = total > 0
        ? Math.ceil(total / PER_PAGE)
        : (data.length === PER_PAGE ? p + 1 : p)
    } catch {
      if (signal.aborted) return
      results.value = []
      totalPages.value = 0
    } finally {
      if (!signal.aborted) loading.value = false
    }
  }

  function goToPage(p: number) {
    const trimmed = query.value.trim()
    if (!trimmed || loading.value) return
    search(trimmed, p)
  }

  function reset() {
    if (debounceTimer) clearTimeout(debounceTimer)
    if (abortController) {
      abortController.abort()
      abortController = null
    }
    results.value = []
    page.value = 1
    totalPages.value = 0
    loading.value = false
  }

  return { results, loading, page, totalPages, goToPage, reset }
}
