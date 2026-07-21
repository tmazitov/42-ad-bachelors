const API = 'https://api.intra.42.fr'

// ── Project data mirrored from src/utils/data.ts ─────────────────────────────
// Format: [id_or_ids, credits]  null id = no API tracking (e.g. seminars)
const STANDARD = [
  [1314, 4], [1327, 3], [1316, 4], [[1994, 2686], 2],
  [[1471, 2687], 3], [[2724, 2005, 2723, 2004], 3],
  [[2720, 2008, 2721, 1476, 2722, 2009], 3],
  [1334, 4], [[1331, 2727], 4], [2007, 3],
  [[2725, 1326, 2726, 1315], 4],
  [[1338, 1339, 1340, 1341, 1342], 4],
  [[1343, 1344, 1345, 1346, 2309], 4],
  [1983, 4], [[1332, 1336], 4], [1337, 4],
  [2267, 2], [2295, 2], [1453, 4], [1457, 4],
  [2551, 4], [1460, 4], [1391, 3], [2372, 4],
  [1382, 3], [2077, 4], [2623, 2],
  [1337, 4], // BAI 498 — same project ID as CCP 161 (ft_transcendence)
  // BAI 499 has no API project ID
]
const STREAM = [
  [1387, 4], [1436, 3], [1386, 3], [1463, 3], [1462, 3],
  [1394, 2], [1416, 3], [1421, 3], [1379, 2], [1395, 2],
  [1427, 4], [2552, 3], [1397, 2], [1399, 2], [1430, 3],
  [1404, 4], [1415, 2], [1479, 2], [1330, 2], [1381, 3],
  [2606, 2], [1396, 2], [1401, 3], [1429, 3], [[1405, 1814], 3],
]
const ELECTIVE = [
  [1418, 2], [1385, 3], [1393, 3], [1470, 3], [1403, 3],
  [1384, 3], [2485, 3], [2600, 3], [1414, 3], [2364, 2],
  [2228, 2], [2550, 3], [2064, 4], [1409, 2], [2225, 3],
  [2076, 4], [1433, 4], [2355, 2], [2346, 2], [1464, 3],
]

const CREDITS_MAX = { standard: 101, stream: 14, elective: 12, total: 132 }

function earned(defs, ids, max) {
  let sum = 0
  for (const [id, credit] of defs) {
    const arr = Array.isArray(id) ? id : [id]
    if (arr.some(i => ids.has(i))) sum += credit
  }
  return Math.min(sum, max)
}

// ── App token (client credentials) ───────────────────────────────────────────
async function getAppToken(env) {
  const res = await fetch(`${API}/oauth/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'client_credentials',
      client_id: env.CLIENT_ID,
      client_secret: env.CLIENT_SECRET,
    }),
  })
  const { access_token } = await res.json()
  return access_token
}

// ── Fetch paginated 42 API resource ──────────────────────────────────────────
async function fetchAll(url, token) {
  const results = []
  let page = 1
  while (true) {
    const res = await fetch(`${url}&page=${page}&per_page=100`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok) break
    const data = await res.json()
    results.push(...data)
    const total = Number(res.headers.get('X-Total') ?? 0)
    if (results.length >= total || data.length < 100) break
    page++
  }
  return results
}

// ── Compute progress for one student ─────────────────────────────────────────
async function computeStudent(cursusUser, token) {
  const { user } = cursusUser
  try {
    const projects = await fetchAll(
      `${API}/v2/users/${user.login}/projects_users?filter[status]=finished`,
      token,
    )
    const ids = new Set(projects.filter(p => p['validated?']).map(p => p.project.id))

    // General credits (GEN seminars) cannot be tracked via API — always 0
    const standard = earned(STANDARD, ids, CREDITS_MAX.standard)
    const stream    = earned(STREAM,    ids, CREDITS_MAX.stream)
    const elective  = earned(ELECTIVE,  ids, CREDITS_MAX.elective)
    const total     = standard + stream + elective
    const percentage = Math.round(total / CREDITS_MAX.total * 1000) / 10

    return {
      login:        user.login,
      display_name: user.displayname,
      image:        user.image?.versions?.medium ?? user.image?.link ?? null,
      general: 0,
      standard,
      stream,
      elective,
      total,
      percentage,
    }
  } catch {
    return null
  }
}

// ── Register a user in KV when they log in via OAuth ─────────────────────────
async function registerUser(accessToken, env) {
  try {
    const res = await fetch(`${API}/v2/me`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    })
    if (!res.ok) return
    const user = await res.json()

    const stored = JSON.parse(await env.LEADERBOARD_KV.get('app_users') || '[]')
    const idx = stored.findIndex(u => u.login === user.login)
    const entry = {
      login:        user.login,
      display_name: user.displayname,
      image:        user.image?.versions?.medium ?? user.image?.link ?? null,
    }
    if (idx >= 0) stored[idx] = entry
    else stored.push(entry)

    await env.LEADERBOARD_KV.put('app_users', JSON.stringify(stored))
  } catch {
    // Non-fatal — don't block the login response
  }
}

// ── Leaderboard builder (runs every hour via cron) ────────────────────────────
async function buildLeaderboard(env) {
  const token = await getAppToken(env)

  const appUsers = JSON.parse(await env.LEADERBOARD_KV.get('app_users') || '[]')
  if (!appUsers.length) return

  const BATCH = 10
  const students = []
  for (let i = 0; i < appUsers.length; i += BATCH) {
    const batch = appUsers.slice(i, i + BATCH)
    const results = await Promise.all(batch.map(u => computeStudent({ user: u }, token)))
    for (const r of results) if (r) students.push(r)
  }

  students.sort((a, b) => b.percentage - a.percentage)

  await env.LEADERBOARD_KV.put('leaderboard', JSON.stringify({
    updated_at: new Date().toISOString(),
    students,
  }))
}

// ── Main export ───────────────────────────────────────────────────────────────
export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url)

    if (request.method === 'OPTIONS') {
      return cors(null, 204)
    }

    // Leaderboard — served from KV cache
    if (url.pathname === '/leaderboard' && request.method === 'GET') {
      const data = await env.LEADERBOARD_KV?.get('leaderboard')
      if (!data) return cors(JSON.stringify({ students: [], updated_at: null }), 200, 'application/json')
      return cors(data, 200, 'application/json')
    }

    // Token refresh — inject server-side secrets
    if (url.pathname === '/oauth/refresh' && request.method === 'POST') {
      let body
      try { body = await request.json() } catch { return cors('Invalid JSON', 400) }

      const res = await fetch(`${API}/oauth/token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          grant_type: 'refresh_token',
          client_id: env.CLIENT_ID,
          client_secret: env.CLIENT_SECRET,
          refresh_token: body.refresh_token,
        }),
      })

      return cors(await res.text(), res.status, 'application/json')
    }

    // Token exchange — inject server-side secrets
    if (url.pathname === '/oauth/token' && request.method === 'POST') {
      let body
      try { body = await request.json() } catch { return cors('Invalid JSON', 400) }

      const res = await fetch(`${API}/oauth/token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body: new URLSearchParams({
          grant_type: 'authorization_code',
          client_id: env.CLIENT_ID,
          client_secret: env.CLIENT_SECRET,
          code: body.code,
          redirect_uri: body.redirect_uri,
        }),
      })

      const text = await res.text()
      if (res.ok && env.LEADERBOARD_KV) {
        try {
          const { access_token } = JSON.parse(text)
          if (access_token) ctx.waitUntil(registerUser(access_token, env))
        } catch {}
      }
      return cors(text, res.status, 'application/json')
    }

    // General proxy for all other 42 API calls
    const proxied = await fetch(`${API}${url.pathname}${url.search}`, {
      method: request.method,
      headers: request.headers,
      body: request.method !== 'GET' && request.method !== 'HEAD' ? request.body : undefined,
    })

    const body = await proxied.text()
    const extra = {}
    for (const h of ['X-Total', 'X-Page', 'X-Per-Page']) {
      const v = proxied.headers.get(h)
      if (v) extra[h] = v
    }
    return cors(body, proxied.status, proxied.headers.get('Content-Type') ?? 'application/json', extra)
  },

  async scheduled(event, env, ctx) {
    ctx.waitUntil(buildLeaderboard(env))
  },
}

function cors(body, status = 200, contentType = 'text/plain', extra = {}) {
  return new Response(body, {
    status,
    headers: {
      'Content-Type': contentType,
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization',
      'Access-Control-Expose-Headers': 'X-Total, X-Page, X-Per-Page',
      ...extra,
    },
  })
}
