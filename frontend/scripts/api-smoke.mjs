const baseUrl = process.argv[2] ?? 'http://127.0.0.1:8081'
const suffix = `${Date.now()}`

async function request(path, init = {}, token) {
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
  const alice = {
    username: `alice_ui_${suffix}`,
    display_name: 'Alice UI',
    password: 'pass123',
  }
  const bob = {
    username: `bob_ui_${suffix}`,
    display_name: 'Bob UI',
    password: 'pass123',
  }

  await request('/api/v1/auth/register', { method: 'POST', body: JSON.stringify(alice) })
  await request('/api/v1/auth/register', { method: 'POST', body: JSON.stringify(bob) })

  const aliceLogin = await request('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: alice.username, password: alice.password }),
  })
  const bobLogin = await request('/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: bob.username, password: bob.password }),
  })

  const aliceToken = aliceLogin.token
  const bobUserId = bobLogin.user.id

  const users = await request('/api/v1/users', {}, aliceToken)
  if (!users.some((user) => user.id === bobUserId)) {
    throw new Error('users list does not contain Bob')
  }

  await request('/api/v1/friends', {
    method: 'POST',
    body: JSON.stringify({ friend_id: bobUserId }),
  }, aliceToken)

  const directConversation = await request('/api/v1/conversations/direct', {
    method: 'POST',
    body: JSON.stringify({ target_user_id: bobUserId }),
  }, aliceToken)

  await request(`/api/v1/conversations/${directConversation.id}/messages`, {
    method: 'POST',
    body: JSON.stringify({ content: 'hello from frontend smoke' }),
  }, aliceToken)

  const messages = await request(`/api/v1/conversations/${directConversation.id}/messages?limit=20`, {}, aliceToken)
  if (!messages.some((message) => message.content === 'hello from frontend smoke')) {
    throw new Error('message history does not contain the sent message')
  }

  console.log('Frontend API smoke passed')
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
