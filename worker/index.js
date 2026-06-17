const API = 'https://api.intra.42.fr'

export default {
  async fetch(request, env) {
    const url = new URL(request.url)

    if (request.method === 'OPTIONS') {
      return cors(null, 204)
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

      return cors(await res.text(), res.status, 'application/json')
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
