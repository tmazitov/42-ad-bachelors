// The 42 CDN rejects unauthenticated requests, so <img> tags cannot load
// profile pictures directly. The backend fetches them with an app token.
//
// AVATAR_VERSION is part of the URL so browsers drop images cached under an
// older version (bump it when cached avatars must be refetched; v2: the
// backend once served wrong photos, fixed in 8d6ac65).
const AVATAR_VERSION = '2'

export function proxiedImage(url: string | null | undefined): string | null {
  if (!url) return null
  return `/api/v1/avatar?url=${encodeURIComponent(url)}&v=${AVATAR_VERSION}`
}
