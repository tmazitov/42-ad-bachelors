// The 42 CDN rejects unauthenticated requests, so <img> tags cannot load
// profile pictures directly. The backend fetches them with an app token.
export function proxiedImage(url: string | null | undefined): string | null {
  if (!url) return null
  return `/api/v1/avatar?url=${encodeURIComponent(url)}`
}
