import { ref, watch, type Ref } from 'vue'
import { api } from '@/services/api'
import type { StudentSummary } from '@/models/Student'

interface SearchResponse {
  students: StudentSummary[]
  total: number
  page: number
  page_size: number
  intra: StudentSummary[] // campus students on 42 who never signed in (page 1 only)
}

// Searches students registered in the app by login or display name, and on
// the first page 42 Abu Dhabi students on intra by login.
export function useStudentSearch(query: Ref<string>) {
  const results = ref<StudentSummary[]>([])
  const intra = ref<StudentSummary[]>([])
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
      intra.value = []
      page.value = 1
      totalPages.value = 0
      loading.value = false
      return
    }

    // The first page also searches 42 (cached on the backend), so wait for
    // the user to pause typing.
    debounceTimer = setTimeout(() => {
      if (trimmed === query.value.trim()) search(trimmed, 1)
    }, 400)
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
      // Keep the intra list while paging through the app's students.
      if (data.page === 1) intra.value = data.intra ?? []
      page.value = data.page
      totalPages.value = Math.ceil(data.total / data.page_size)
    } catch {
      if (signal.aborted) return
      results.value = []
      intra.value = []
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
    intra.value = []
    page.value = 1
    totalPages.value = 0
    loading.value = false
  }

  return { results, intra, loading, page, totalPages, goToPage, reset }
}
