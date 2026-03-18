const authBase = process.argv[2] ?? 'http://127.0.0.1:8081'
const apiBase = process.argv[3] ?? 'http://127.0.0.1:8082'
const suffix = `${Date.now()}`

async function request(baseUrl, path, init = {}, token) {
  const response = await fetch(`${baseUrl}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init.headers ?? {}),
    },
  })
  const payload = await response.json()
  if (!response.ok || payload.code !== 0) {
    throw new Error(`${path} failed: ${payload.message}`)
  }
  return payload.data
}

async function main() {
  await request(apiBase, '/api/v1/system/ready')

  const alice = {
    username: `alice_enterprise_${suffix}`,
    display_name: 'Alice Enterprise',
    password: 'pass12345',
  }
  const bob = {
    username: `bob_enterprise_${suffix}`,
    display_name: 'Bob Enterprise',
    password: 'pass12345',
  }

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

  const aliceToken = aliceLogin.access_token
  const bobUserId = bobLogin.user.id

  await request(apiBase, '/api/v1/friend-requests', {
    method: 'POST',
    body: JSON.stringify({ addressee_id: bobUserId, message: 'smoke' }),
  }, aliceToken)

  const conversation = await request(apiBase, '/api/v1/conversations/direct', {
    method: 'POST',
    body: JSON.stringify({ target_user_id: bobUserId }),
  }, aliceToken)

  await request(apiBase, `/api/v1/conversations/${conversation.id}/messages`, {
    method: 'POST',
    body: JSON.stringify({ content: 'hello from enterprise smoke', message_type: 'text', client_msg_id: `smoke-${suffix}` }),
  }, aliceToken)

  const messages = await request(apiBase, `/api/v1/conversations/${conversation.id}/messages?limit=20`, {}, aliceToken)
  if (!messages.some((message) => message.content === 'hello from enterprise smoke')) {
    throw new Error('message history does not contain the sent message')
  }

  console.log('Enterprise API smoke passed')
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
