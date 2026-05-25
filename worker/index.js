export default {
  async fetch(request, env) {
    if (request.method === 'OPTIONS') {
      return reply(null, 204)
    }

    if (request.method !== 'POST') {
      return reply('Method not allowed', 405)
    }

    let body
    try {
      body = await request.json()
    } catch {
      return reply('Invalid JSON', 400)
    }

    const res = await fetch('https://api.intra.42.fr/oauth/token', {
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

    const data = await res.text()
    return reply(data, res.status, 'application/json')
  },
}

function reply(body, status = 200, contentType = 'text/plain') {
  return new Response(body, {
    status,
    headers: {
      'Content-Type': contentType,
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'POST, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type',
    },
  })
}
