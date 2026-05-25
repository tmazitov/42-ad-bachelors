import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

export interface ProjectUser {
  id: number
  final_mark: number | null
  status: string
  'validated?': boolean
  marked_at: string | null
  project: {
    id: number
    name: string
    slug: string
  }
  cursus_ids: number[]
}

export function useProjects() {
  const auth = useAuthStore()
  const projects = ref<ProjectUser[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchProjects(userId: number) {
    if (!auth.token) return
    loading.value = true
    error.value = null
    try {
      const res = await fetch(
        `/api/42/v2/users/${userId}/projects_users?filter[status]=finished&sort=-marked_at&per_page=100`,
        { headers: { Authorization: `Bearer ${auth.token}` } },
      )
      if (!res.ok) throw new Error(`${res.status}`)
      projects.value = await res.json()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load projects'
    } finally {
      loading.value = false
    }
  }

  return { projects, loading, error, fetchProjects }
}
