const CLIENT_ID = import.meta.env.VITE_42_CLIENT_ID as string
const CLIENT_SECRET = import.meta.env.VITE_42_CLIENT_SECRET as string
const REDIRECT_URI = import.meta.env.VITE_42_REDIRECT_URI as string
const API_BASE = (import.meta.env.VITE_42_API_BASE as string) || ''

export function redirectToLogin(): void {
  const params = new URLSearchParams({
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    response_type: 'code',
    scope: 'public projects',
  })
  window.location.href = `https://api.intra.42.fr/oauth/authorize?${params}`
}

export async function exchangeCode(code: string): Promise<string> {
  const res = await fetch(`${API_BASE}/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      grant_type: 'authorization_code',
      client_id: CLIENT_ID,
      client_secret: CLIENT_SECRET,
      code,
      redirect_uri: REDIRECT_URI,
    }),
  })

  if (!res.ok) throw new Error(`Token exchange failed: ${res.status}`)

  const data = await res.json()
  return data.access_token as string
}
