import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { api, ApiError } from '@/services/api'
import type { Student } from '@/models/Student'

// The signed-in student. The session lives in an httpOnly cookie set by the
// backend, so the app never handles tokens: it asks /me who is signed in.
export const useAuthStore = defineStore('auth', () => {
  const user = ref<Student | null>(null)
  const loading = ref(false)
  // True once the first /me answered (signed in or not).
  const checked = ref(false)
  // True after "Log In" while the browser navigates to 42.
  const redirecting = ref(false)
  let pending: Promise<void> | null = null

  const isLoggedIn = computed(() => !!user.value)

  // Loads the signed-in student (once, unless force). A 401 means signed out.
  function fetchMe(force = false): Promise<void> {
    if (pending) return pending
    if (checked.value && !force) return Promise.resolve()

    loading.value = true
    pending = api<Student>('/me')
      .then((me) => { user.value = me })
      .catch((e) => {
        if (!(e instanceof ApiError && e.status === 401)) console.error('Loading the signed-in student failed:', e)
        user.value = null
      })
      .finally(() => {
        checked.value = true
        loading.value = false
        pending = null
      })
    return pending
  }

  function login() {
    redirecting.value = true
    window.location.href = '/api/v1/auth/login'
  }

  // Coming back from 42 with the browser's Back button can restore this page
  // from the back/forward cache with the overlay still up.
  window.addEventListener('pageshow', (e) => {
    if (e.persisted) redirecting.value = false
  })

  // Saves the signed-in student's stream; the backend returns the updated student.
  async function setStream(key: string) {
    user.value = await api<Student>('/me/stream', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ stream: key }),
    })
  }

  // Marks a self-reported module completed (or not); returns the updated student.
  async function markModule(code: string, completed: boolean) {
    user.value = await api<Student>('/me/modules', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ code, completed }),
    })
  }

  async function logout() {
    try {
      await api('/auth/logout', { method: 'POST' })
    } finally {
      user.value = null
    }
  }

  return { user, loading, checked, redirecting, isLoggedIn, fetchMe, login, logout, setStream, markModule }
})
