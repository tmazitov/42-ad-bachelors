// Calls the backend API. It is served from the same origin as the app (the
// Vite dev proxy locally, a reverse proxy in production), so the httpOnly
// session cookie is sent automatically.

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message)
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const res = await fetch(`/api/v1${path}`, { credentials: 'same-origin', ...init })
  if (!res.ok) {
    let message = `${res.status}`
    try {
      message = (await res.json()).error ?? message
    } catch {
      // not JSON
    }
    throw new ApiError(res.status, message)
  }
  return res.status === 204 ? (undefined as T) : res.json()
}
