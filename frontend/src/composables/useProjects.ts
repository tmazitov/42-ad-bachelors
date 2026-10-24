import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { apiFetch } from '@/services/apiFetch'

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

// Fetches a user's finished projects (first 100, most recent first).
export async function fetchFinishedProjects(userId: number): Promise<ProjectUser[]> {
  const res = await apiFetch(
    `${import.meta.env.VITE_42_API_BASE || ''}/v2/users/${userId}/projects_users?filter[status]=finished&sort=-marked_at&per_page=100`,
  )
  if (!res.ok) throw new Error(`${res.status}`)
  return res.json()
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
      projects.value = await fetchFinishedProjects(userId)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load projects'
    } finally {
      loading.value = false
    }
  }

  return { projects, loading, error, fetchProjects }
}
