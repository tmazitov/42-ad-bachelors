import { ref, watch, type Ref } from 'vue'
import { api } from '@/services/api'
import type { StudentSummary } from '@/models/Student'

interface SearchResponse {
  students: StudentSummary[]
  total: number
  page: number
  page_size: number
}

// Searches students registered in the app by login or display name.
export function useStudentSearch(query: Ref<string>) {
  const results = ref<StudentSummary[]>([])
  const loading = ref(false)
  const page = ref(1)
  const totalPages = ref(0)
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let abortController: AbortController | null = null

  watch(query, (val) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    abortController?.abort()
    abortController = null

    const trimmed = val.trim()
    if (!trimmed) {
      results.value = []
      page.value = 1
      totalPages.value = 0
      loading.value = false
      return
    }

    // Searching the local database is cheap, so a short debounce is enough.
    debounceTimer = setTimeout(() => {
      if (trimmed === query.value.trim()) search(trimmed, 1)
    }, 250)
  })

  async function search(q: string, p: number) {
    abortController?.abort()
    abortController = new AbortController()
    const { signal } = abortController

    loading.value = true
    try {
      const params = new URLSearchParams({ search: q, page: String(p) })
      const data = await api<SearchResponse>(`/students?${params}`, { signal })
      results.value = data.students
      page.value = data.page
      totalPages.value = Math.ceil(data.total / data.page_size)
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
    abortController?.abort()
    abortController = null
    results.value = []
    page.value = 1
    totalPages.value = 0
    loading.value = false
  }

  return { results, loading, page, totalPages, goToPage, reset }
}
