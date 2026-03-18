const authBase = process.argv[2] ?? 'http://127.0.0.1:19081'
const apiBase = process.argv[3] ?? 'http://127.0.0.1:19082'
const wsBase = process.argv[4] ?? 'ws://127.0.0.1:19083/ws'
const suffix = Date.now().toString()

async function request(base, path, init = {}, token) {
  const response = await fetch(base + path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init.headers ?? {}),
    },
  })
  const payload = await response.json()
  if (!response.ok || payload.code !== 0) {
    throw new Error(`${path}: ${payload.message}`)
  }
  return payload.data
}

function waitForEvent(events, predicate, timeout = 5000) {
  const started = Date.now()
  return new Promise((resolve, reject) => {
    const timer = setInterval(() => {
      const match = events.find(predicate)
      if (match) {
        clearInterval(timer)
        resolve(match)
        return
      }
      if (Date.now() - started > timeout) {
        clearInterval(timer)
        reject(new Error('timeout waiting for realtime event'))
      }
    }, 50)
  })
}

async function main() {
  const alice = { username: `alice_runtime_${suffix}`, display_name: 'Alice Runtime', password: 'pass12345' }
  const bob = { username: `bob_runtime_${suffix}`, display_name: 'Bob Runtime', password: 'pass12345' }

  await request(authBase, '/api/v1/auth/register', { method: 'POST', body: JSON.stringify(alice) })
  await request(authBase, '/api/v1/auth/register', { method: 'POST', body: JSON.stringify(bob) })

  const aliceLogin = await request(authBase, '/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: alice.username, password: alice.password }),
  })
  const bobLogin = await request(authBase, '/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: bob.username, password: bob.password }),
  })

  const refreshed = await request(authBase, '/api/v1/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: aliceLogin.refresh_token }),
  })
  if (!refreshed.access_token) {
    throw new Error('refresh flow did not return access token')
  }

  const friendRequest = await request(apiBase, '/api/v1/friend-requests', {
    method: 'POST',
    body: JSON.stringify({ addressee_id: bobLogin.user.id, message: 'runtime smoke' }),
  }, aliceLogin.access_token)

  await request(apiBase, `/api/v1/friend-requests/${friendRequest.id}/approve`, { method: 'POST' }, bobLogin.access_token)

  const direct = await request(apiBase, '/api/v1/conversations/direct', {
    method: 'POST',
    body: JSON.stringify({ target_user_id: bobLogin.user.id }),
  }, aliceLogin.access_token)

  const events = []
  const ws = new WebSocket(`${wsBase}?token=${encodeURIComponent(bobLogin.access_token)}`)
  ws.addEventListener('message', (event) => events.push(JSON.parse(event.data)))

  await waitForEvent(events, (event) => event.type === 'auth.ok')

  const sent = await request(apiBase, `/api/v1/conversations/${direct.id}/messages`, {
    method: 'POST',
    body: JSON.stringify({
      content: 'runtime websocket hello',
      message_type: 'text',
      client_msg_id: `runtime-${suffix}`,
    }),
  }, aliceLogin.access_token)

  await waitForEvent(
    events,
    (event) => event.type === 'message.new' && event.payload.message.client_msg_id === `runtime-${suffix}`,
  )

  await request(apiBase, `/api/v1/conversations/${direct.id}/read`, {
    method: 'POST',
    body: JSON.stringify({ seq: sent.seq }),
  }, bobLogin.access_token)

  await waitForEvent(
    events,
    (event) => event.type === 'conversation.read' && event.payload.last_read_seq === sent.seq,
  )

  const search = await request(
    apiBase,
    `/api/v1/search?q=${encodeURIComponent('runtime websocket')}&scope=messages&limit=8`,
    {},
    bobLogin.access_token,
  )
  if (!search.messages.some((item) => item.content.includes('runtime websocket hello'))) {
    throw new Error('search did not return the sent message')
  }

  const sessions = await request(authBase, '/api/v1/auth/sessions', {}, aliceLogin.access_token)
  if (!Array.isArray(sessions) || sessions.length === 0) {
    throw new Error('sessions endpoint returned empty result')
  }

  ws.close()
  console.log('Runtime smoke passed')
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
