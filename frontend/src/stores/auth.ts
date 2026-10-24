import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { refreshAccessToken, type TokenData } from '@/services/oauth'

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
const REFRESH_TOKEN_KEY = '42_refresh_token'
const EXPIRES_AT_KEY = '42_expires_at'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const storedRefreshToken = ref<string | null>(localStorage.getItem(REFRESH_TOKEN_KEY))
  const expiresAt = ref<number>(parseInt(localStorage.getItem(EXPIRES_AT_KEY) || '0'))
  const user = ref<FtUser | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isLoggedIn = computed(() => !!token.value)

  let refreshTimer: ReturnType<typeof setTimeout> | null = null

  function scheduleRefresh() {
    if (refreshTimer) clearTimeout(refreshTimer)
    if (!expiresAt.value || !storedRefreshToken.value) return
    const delay = expiresAt.value - Date.now() - 5 * 60 * 1000
    if (delay > 0) {
      refreshTimer = setTimeout(() => tryRefresh(), delay)
    }
  }

  function setTokens(data: TokenData) {
    token.value = data.access_token
    storedRefreshToken.value = data.refresh_token
    expiresAt.value = (data.created_at + data.expires_in) * 1000
    localStorage.setItem(TOKEN_KEY, data.access_token)
    localStorage.setItem(REFRESH_TOKEN_KEY, data.refresh_token)
    localStorage.setItem(EXPIRES_AT_KEY, String(expiresAt.value))
    scheduleRefresh()
  }

  function logout() {
    if (refreshTimer) clearTimeout(refreshTimer)
    token.value = null
    storedRefreshToken.value = null
    expiresAt.value = 0
    user.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    localStorage.removeItem(EXPIRES_AT_KEY)
  }

  async function tryRefresh(): Promise<boolean> {
    if (!storedRefreshToken.value) return false
    try {
      const data = await refreshAccessToken(storedRefreshToken.value)
      setTokens(data)
      return true
    } catch {
      logout()
      return false
    }
  }

  async function fetchUser() {
    if (!token.value) return

    // Refresh if token is expired or expiring within 60 seconds
    if (expiresAt.value && Date.now() >= expiresAt.value - 60 * 1000) {
      const refreshed = await tryRefresh()
      if (!refreshed) return
    }

    loading.value = true
    error.value = null
    try {
      const res = await fetch(`${import.meta.env.VITE_42_API_BASE || ''}/v2/me`, {
        headers: { Authorization: `Bearer ${token.value}` },
      })
      if (!res.ok) {
        if (res.status === 401) {
          const refreshed = await tryRefresh()
          if (!refreshed) return
          // retry once with new token
          const retry = await fetch(`${import.meta.env.VITE_42_API_BASE || ''}/v2/me`, {
            headers: { Authorization: `Bearer ${token.value}` },
          })
          if (!retry.ok) { logout(); return }
          user.value = await retry.json()
          return
        }
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

  // Schedule refresh on store init if we have a stored token with expiry
  scheduleRefresh()

  return { token, user, loading, error, isLoggedIn, setTokens, logout, fetchUser, tryRefresh }
})
