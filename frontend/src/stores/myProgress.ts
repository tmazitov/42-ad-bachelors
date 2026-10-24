import { ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchFinishedProjects, type ProjectUser } from '@/composables/useProjects'
import { useProgress } from '@/composables/useProgress'

// The signed-in student's projects and progress, shared by the header menu
// and the home page so they are fetched once.
export const useMyProgressStore = defineStore('myProgress', () => {
  const projects = ref<ProjectUser[]>([])
  const loadedFor = ref<number | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const progress = useProgress(projects)

  let inFlight: Promise<void> | null = null
  let inFlightFor: number | null = null

  // Loads the user's projects unless they are already loaded (or loading).
  function load(userId: number): Promise<void> {
    if (loadedFor.value === userId) return Promise.resolve()
    if (inFlight && inFlightFor === userId) return inFlight

    inFlightFor = userId
    loading.value = true
    error.value = null
    inFlight = fetchFinishedProjects(userId)
      .then((data) => {
        if (inFlightFor !== userId) return
        projects.value = data
        loadedFor.value = userId
      })
      .catch((e) => {
        if (inFlightFor !== userId) return
        error.value = e instanceof Error ? e.message : 'Failed to load projects'
      })
      .finally(() => {
        if (inFlightFor !== userId) return
        loading.value = false
        inFlight = null
        inFlightFor = null
      })
    return inFlight
  }

  function reset() {
    projects.value = []
    loadedFor.value = null
    loading.value = false
    error.value = null
    inFlight = null
    inFlightFor = null
  }

  return { projects, loadedFor, loading, error, progress, load, reset }
})
