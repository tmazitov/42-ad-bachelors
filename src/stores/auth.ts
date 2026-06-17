import { ref, computed } from 'vue'
import { defineStore } from 'pinia'

export interface FtUser {
  id: number
  login: string
  displayname: string
  'staff?': boolean
  'alumni?': boolean
  location: string | null
  image: {
    link: string | null
    versions: {
      large: string | null
      medium: string | null
      small: string | null
      micro: string | null
    }
  }
  cursus_users: Array<{
    cursus: { id: number; name: string }
    level: number
    grade: string | null
  }>
}

const TOKEN_KEY = '42_access_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const user = ref<FtUser | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isLoggedIn = computed(() => !!token.value)

  function setToken(t: string) {
    token.value = t
    localStorage.setItem(TOKEN_KEY, t)
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
  }

  async function fetchUser() {
    if (!token.value) return
    loading.value = true
    error.value = null
    try {
      const res = await fetch(`${import.meta.env.VITE_42_API_BASE || ''}/v2/me`, {
        headers: { Authorization: `Bearer ${token.value}` },
      })
      if (!res.ok) {
        logout()
        return
      }
      user.value = await res.json()
    } catch {
      logout()
    } finally {
      loading.value = false
    }
  }

  return { token, user, loading, error, isLoggedIn, setToken, logout, fetchUser }
})
