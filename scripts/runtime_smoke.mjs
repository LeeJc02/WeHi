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

async function requestFailure(base, path, init = {}, token) {
  const response = await fetch(base + path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init.headers ?? {}),
    },
  })
  const payload = await response.json()
  if (response.ok && payload.code === 0) {
    throw new Error(`${path}: expected failure`)
  }
  return payload
}

async function uploadObject(base, path, body, headers = {}, token) {
  const response = await fetch(base + path, {
    method: 'PUT',
    body,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(headers ?? {}),
    },
  })
  if (response.status !== 204) {
    const payload = await response.text()
    throw new Error(`${path}: upload failed ${response.status} ${payload}`)
  }
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

async function connectWebSocket(token) {
  const events = []
  const ws = new WebSocket(`${wsBase}?token=${encodeURIComponent(token)}`)
  ws.addEventListener('message', (event) => events.push(JSON.parse(event.data)))
  await waitForEvent(events, (event) => event.type === 'auth.ok')
  return { ws, events }
}

async function waitForCondition(fn, timeout = 10000, interval = 200) {
  const started = Date.now()
  while (Date.now() - started < timeout) {
    const result = await fn()
    if (result) {
      return result
    }
    await new Promise((resolve) => setTimeout(resolve, interval))
  }
  throw new Error('timeout waiting for condition')
}

async function main() {
  const alice = { username: `alice_runtime_${suffix}`, display_name: 'Alice Runtime', password: 'pass12345' }
  const bob = { username: `bob_runtime_${suffix}`, display_name: 'Bob Runtime', password: 'pass12345' }
  const charlie = { username: `charlie_runtime_${suffix}`, display_name: 'Charlie Runtime', password: 'pass12345' }

  await request(authBase, '/api/v1/auth/register', { method: 'POST', body: JSON.stringify(alice) })
  await request(authBase, '/api/v1/auth/register', { method: 'POST', body: JSON.stringify(bob) })
  await request(authBase, '/api/v1/auth/register', { method: 'POST', body: JSON.stringify(charlie) })

  const aliceLogin = await request(authBase, '/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: alice.username, password: alice.password }),
  })
  const bobLogin = await request(authBase, '/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: bob.username, password: bob.password }),
  })
  const aliceSecondLogin = await request(authBase, '/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: alice.username, password: alice.password }),
  })
  const charlieLogin = await request(authBase, '/api/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: charlie.username, password: charlie.password }),
  })

  const refreshed = await request(authBase, '/api/v1/auth/refresh', {
    method: 'POST',
    body: JSON.stringify({ refresh_token: aliceLogin.refresh_token }),
  })
  if (!refreshed.access_token) {
    throw new Error('refresh flow did not return access token')
  }

  const adminLogin = await request(apiBase, '/api/v1/admin/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: 'root', password: '123456' }),
  })
  if (!adminLogin.access_token || !adminLogin.admin.must_change_password) {
    throw new Error('admin login did not require initial password change')
  }

  const adminProfileBeforeChange = await request(apiBase, '/api/v1/admin/auth/me', {}, adminLogin.access_token)
  if (!adminProfileBeforeChange.must_change_password) {
    throw new Error('admin profile did not reflect password change requirement')
  }

  const adminMonitorForbidden = await requestFailure(apiBase, '/api/v1/admin/monitor/overview', {}, adminLogin.access_token)
  if (adminMonitorForbidden.error_code !== 'ADMIN_PASSWORD_CHANGE_REQUIRED') {
    throw new Error('admin protected endpoint did not enforce password change requirement')
  }

  const adminNewPassword = `admin-${suffix}`
  await request(apiBase, '/api/v1/admin/auth/change-password', {
    method: 'POST',
    body: JSON.stringify({ current_password: '123456', new_password: adminNewPassword }),
  }, adminLogin.access_token)

  const adminLoginAfterChange = await request(apiBase, '/api/v1/admin/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: 'root', password: adminNewPassword }),
  })
  if (adminLoginAfterChange.admin.must_change_password) {
    throw new Error('admin password change did not clear must_change_password flag')
  }

  const adminProfileAfterChange = await request(apiBase, '/api/v1/admin/auth/me', {}, adminLoginAfterChange.access_token)
  if (adminProfileAfterChange.must_change_password) {
    throw new Error('admin profile still requires password change after update')
  }

  const monitorOverview = await request(apiBase, '/api/v1/admin/monitor/overview', {}, adminLoginAfterChange.access_token)
  if (!Array.isArray(monitorOverview.services) || monitorOverview.services.length === 0) {
    throw new Error('admin monitor overview returned no services')
  }
  const monitorTimeseries = await request(apiBase, '/api/v1/admin/monitor/timeseries', {}, adminLoginAfterChange.access_token)
  if (!Array.isArray(monitorTimeseries.points) || monitorTimeseries.points.length === 0) {
    throw new Error('admin monitor timeseries returned no points')
  }

  const aiConfig = await request(apiBase, '/api/v1/admin/ai-config', {}, adminLoginAfterChange.access_token)
  if (!aiConfig.bot?.username || !aiConfig.bot?.default_provider) {
    throw new Error('admin AI config payload is incomplete')
  }

  const friendRequest = await request(apiBase, '/api/v1/friend-requests', {
    method: 'POST',
    body: JSON.stringify({ addressee_id: bobLogin.user.id, message: 'runtime smoke' }),
  }, aliceLogin.access_token)

  await request(apiBase, `/api/v1/friend-requests/${friendRequest.id}/approve`, { method: 'POST' }, bobLogin.access_token)

  const aliceFriends = await request(apiBase, '/api/v1/friends', {}, aliceLogin.access_token)
  const aiBot = aliceFriends.find((friend) => friend.is_ai_bot)
  if (!aiBot) {
    throw new Error('AI Bot was not provisioned in friend list')
  }
  if (aliceFriends[0]?.id !== aiBot.id) {
    throw new Error('AI Bot is not pinned at the top of friend list')
  }

  const direct = await request(apiBase, '/api/v1/conversations/direct', {
    method: 'POST',
    body: JSON.stringify({ target_user_id: bobLogin.user.id }),
  }, aliceLogin.access_token)

  await requestFailure(apiBase, `/api/v1/conversations/${direct.id}/members`, {
    method: 'POST',
    body: JSON.stringify({ member_ids: [charlieLogin.user.id] }),
  }, aliceLogin.access_token)

  const bobSocket = await connectWebSocket(bobLogin.access_token)
  const aliceSocketA = await connectWebSocket(aliceLogin.access_token)
  const aliceSocketB = await connectWebSocket(aliceSecondLogin.access_token)

  const sent = await request(apiBase, `/api/v1/conversations/${direct.id}/messages`, {
    method: 'POST',
    body: JSON.stringify({
      content: 'runtime websocket hello',
      message_type: 'text',
      client_msg_id: `runtime-${suffix}`,
    }),
  }, aliceLogin.access_token)

  await waitForEvent(
    bobSocket.events,
    (event) => event.type === 'message.persisted' && event.payload.message.client_msg_id === `runtime-${suffix}`,
  )

  const duplicate = await request(apiBase, `/api/v1/conversations/${direct.id}/messages`, {
    method: 'POST',
    body: JSON.stringify({
      content: 'runtime websocket hello',
      message_type: 'text',
      client_msg_id: `runtime-${suffix}`,
    }),
  }, aliceLogin.access_token)
  if (duplicate.id !== sent.id) {
    throw new Error('duplicate client_msg_id created a second message')
  }

  await request(apiBase, `/api/v1/conversations/${direct.id}/read`, {
    method: 'POST',
    body: JSON.stringify({ seq: sent.seq }),
  }, bobLogin.access_token)

  await waitForEvent(
    bobSocket.events,
    (event) => event.type === 'message.read' && event.payload.last_read_seq === sent.seq,
  )
  await waitForEvent(
    aliceSocketA.events,
    (event) => event.type === 'message.read' && event.payload.last_read_seq === sent.seq,
  )
  await waitForEvent(
    aliceSocketB.events,
    (event) => event.type === 'message.read' && event.payload.last_read_seq === sent.seq,
  )

  await request(apiBase, `/api/v1/messages/${sent.id}/recall`, { method: 'POST' }, aliceLogin.access_token)
  await waitForCondition(async () => {
    const rows = await request(apiBase, `/api/v1/conversations/${direct.id}/messages?limit=10`, {}, aliceLogin.access_token)
    return rows.some((message) => message.id === sent.id && message.recalled_at) ? rows : null
  }, 5000, 200)

  const resolvedMessage = await request(
    apiBase,
    `/api/v1/admin/messages/resolve?client_msg_id=${encodeURIComponent(`runtime-${suffix}`)}&sender_id=${aliceLogin.user.id}&conversation_id=${direct.id}`,
    {},
    adminLoginAfterChange.access_token,
  )
  if (resolvedMessage.message_id !== sent.id) {
    throw new Error('admin message resolver returned unexpected message id')
  }

  const messageJourney = await request(apiBase, `/api/v1/admin/message-journey/${sent.id}`, {}, adminLoginAfterChange.access_token)
  if (!messageJourney.stages.some((stage) => stage.name === 'read')) {
    throw new Error('admin message journey did not include read stage')
  }
  if (!messageJourney.stages.some((stage) => stage.name === 'recalled')) {
    throw new Error('admin message journey did not include recalled stage')
  }

  const consistency = await request(apiBase, `/api/v1/admin/conversations/${direct.id}/consistency`, {}, adminLoginAfterChange.access_token)
  if (!Array.isArray(consistency.members) || consistency.members.length !== 2) {
    throw new Error('admin conversation consistency returned unexpected member count')
  }

  const conversationEvents = await request(apiBase, `/api/v1/admin/conversations/${direct.id}/events?limit=20`, {}, adminLoginAfterChange.access_token)
  if (!conversationEvents.some((event) => event.event_type === 'message.read')) {
    throw new Error('admin conversation events did not include read event')
  }

  const group = await request(apiBase, '/api/v1/conversations/group', {
    method: 'POST',
    body: JSON.stringify({ name: 'Runtime Group', member_ids: [bobLogin.user.id, charlieLogin.user.id] }),
  }, aliceLogin.access_token)

  const groupMembers = await request(apiBase, `/api/v1/conversations/${group.id}/members`, {}, aliceLogin.access_token)
  if (groupMembers.length !== 3) {
    throw new Error('group members were not provisioned correctly')
  }

  await request(apiBase, `/api/v1/conversations/${group.id}/members/${charlieLogin.user.id}`, { method: 'DELETE' }, aliceLogin.access_token)
  const groupMembersAfterRemove = await request(apiBase, `/api/v1/conversations/${group.id}/members`, {}, aliceLogin.access_token)
  if (groupMembersAfterRemove.some((member) => member.user_id === charlieLogin.user.id)) {
    throw new Error('group member removal did not converge')
  }

  await requestFailure(apiBase, `/api/v1/conversations/${group.id}/leave`, { method: 'POST' }, aliceLogin.access_token)

  await request(apiBase, `/api/v1/conversations/${group.id}/members`, {
    method: 'POST',
    body: JSON.stringify({ member_ids: [charlieLogin.user.id] }),
  }, aliceLogin.access_token)
  const groupMembersAfterReAdd = await request(apiBase, `/api/v1/conversations/${group.id}/members`, {}, aliceLogin.access_token)
  if (!groupMembersAfterReAdd.some((member) => member.user_id === charlieLogin.user.id)) {
    throw new Error('removed group member could not be added back')
  }

  await request(apiBase, `/api/v1/conversations/${group.id}/members/${bobLogin.user.id}`, { method: 'DELETE' }, aliceLogin.access_token)
  await request(apiBase, `/api/v1/conversations/${group.id}/members/${charlieLogin.user.id}`, { method: 'DELETE' }, aliceLogin.access_token)
  await request(apiBase, `/api/v1/conversations/${group.id}/leave`, { method: 'POST' }, aliceLogin.access_token)
  const conversationsAfterDissolve = await request(apiBase, '/api/v1/conversations', {}, aliceLogin.access_token)
  if (conversationsAfterDissolve.some((conversation) => conversation.id === group.id)) {
    throw new Error('last member leave did not dissolve the group conversation')
  }

  const bobCursor = await request(apiBase, '/api/v1/sync/cursor', {}, bobLogin.access_token)
  bobSocket.ws.close()
  const reconnectClientMsgID = `runtime-reconnect-${suffix}`
  await request(apiBase, `/api/v1/conversations/${direct.id}/messages`, {
    method: 'POST',
    body: JSON.stringify({
      content: 'runtime reconnect hello',
      message_type: 'text',
      client_msg_id: reconnectClientMsgID,
    }),
  }, aliceLogin.access_token)

  const bobCatchup = await waitForCondition(async () => {
    const payload = await request(apiBase, `/api/v1/sync/events?cursor=${bobCursor.cursor}&limit=20`, {}, bobLogin.access_token)
    return payload.events.some((event) => event.event_type === 'message.persisted' && event.payload.message.client_msg_id === reconnectClientMsgID)
      ? payload
      : null
  }, 8000, 250)
  if (!bobCatchup.events.some((event) => event.event_type === 'message.persisted' && event.payload.message.client_msg_id === reconnectClientMsgID)) {
    throw new Error('reconnect sync catch-up did not include missed message')
  }
  const bobSocketAfterReconnect = await connectWebSocket(bobLogin.access_token)
  bobSocketAfterReconnect.ws.close()

  const conversations = await request(apiBase, '/api/v1/conversations', {}, aliceLogin.access_token)
  const botConversation = conversations.find((conversation) => conversation.type === 'direct' && conversation.name === 'AI Bot')
  if (!botConversation) {
    throw new Error('AI Bot conversation missing from conversation list')
  }
  if (conversations[0]?.id !== botConversation.id || !botConversation.pinned) {
    throw new Error('AI Bot conversation is not pinned at the top of conversation list')
  }

  const botClientMsgID = `runtime-bot-${suffix}`
  await request(apiBase, `/api/v1/conversations/${botConversation.id}/messages`, {
    method: 'POST',
    body: JSON.stringify({
      content: 'ping ai bot',
      message_type: 'text',
      client_msg_id: botClientMsgID,
    }),
  }, aliceLogin.access_token)

  const botMessages = await waitForCondition(async () => {
    const rows = await request(apiBase, `/api/v1/conversations/${botConversation.id}/messages?limit=10`, {}, aliceLogin.access_token)
    const hasUserMessage = rows.some((message) => message.client_msg_id === botClientMsgID)
    const hasBotReply = rows.some((message) => message.sender_id === aiBot.id && message.client_msg_id !== botClientMsgID)
    return hasUserMessage && hasBotReply ? rows : null
  }, 35000, 400)

  if (!botMessages.some((message) => message.sender_id === aiBot.id)) {
    throw new Error('AI Bot did not respond asynchronously')
  }

  const auditLogs = await request(
    apiBase,
    `/api/v1/admin/audit/ai-calls?conversation_id=${botConversation.id}&limit=10`,
    {},
    adminLoginAfterChange.access_token,
  )
  if (!Array.isArray(auditLogs)) {
    throw new Error('admin AI audit list returned an invalid payload')
  }

  const search = await request(
    apiBase,
    `/api/v1/search?q=${encodeURIComponent('runtime websocket')}&scope=messages&limit=8`,
    {},
    bobLogin.access_token,
  )
  if (!search.messages.some((item) => item.content.includes('runtime websocket hello'))) {
    throw new Error('search did not return the sent message')
  }

  const uploadContent = `runtime upload ${suffix}`
  const presigned = await request(apiBase, '/api/v1/uploads/presign', {
    method: 'POST',
    body: JSON.stringify({
      filename: 'runtime-smoke.txt',
      content_type: 'text/plain',
      size_bytes: uploadContent.length,
    }),
  }, aliceLogin.access_token)
  await uploadObject(apiBase, presigned.upload_path, uploadContent, presigned.headers, aliceLogin.access_token)
  const completedUpload = await request(apiBase, '/api/v1/uploads/complete', {
    method: 'POST',
    body: JSON.stringify({
      object_key: presigned.object_key,
      filename: 'runtime-smoke.txt',
      content_type: 'text/plain',
      size_bytes: uploadContent.length,
    }),
  }, aliceLogin.access_token)
  if (completedUpload.attachment.size_bytes !== uploadContent.length) {
    throw new Error('upload completion returned unexpected size')
  }
  const uploadedBody = await fetch(apiBase + completedUpload.attachment.url).then((response) => response.text())
  if (uploadedBody !== uploadContent) {
    throw new Error('uploaded object could not be downloaded correctly')
  }

  const sessions = await request(authBase, '/api/v1/auth/sessions', {}, aliceLogin.access_token)
  if (!Array.isArray(sessions) || sessions.length === 0) {
    throw new Error('sessions endpoint returned empty result')
  }

  aliceSocketA.ws.close()
  aliceSocketB.ws.close()
  console.log('Runtime smoke passed')
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
