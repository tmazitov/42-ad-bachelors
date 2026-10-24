import { useAuthStore } from '@/stores/auth'

export async function apiFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const auth = useAuthStore()

  const makeOptions = (token: string): RequestInit => ({
    ...options,
    headers: { ...options.headers, Authorization: `Bearer ${token}` },
  })

  const signal = options.signal as AbortSignal | undefined

  const res = await fetch(url, makeOptions(auth.token!))
  if (res.status !== 401) return res

  if (signal?.aborted) return res

  const refreshed = await auth.tryRefresh()
  if (!refreshed || !auth.token) return res

  if (signal?.aborted) return res

  return fetch(url, makeOptions(auth.token))
}
