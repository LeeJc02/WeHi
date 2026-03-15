import { useCallback, useEffect, useState } from 'react'
import './App.css'
import { api } from './api'
import type { Conversation, Friend, Message, User } from './types'

type Mode = 'login' | 'register'

const defaultCredentials = {
  username: '',
  display_name: '',
  password: '',
}

function App() {
  const [mode, setMode] = useState<Mode>('login')
  const [credentials, setCredentials] = useState(defaultCredentials)
  const [token, setToken] = useState<string>(() => localStorage.getItem('chat_token') ?? '')
  const [me, setMe] = useState<User | null>(null)
  const [users, setUsers] = useState<User[]>([])
  const [friends, setFriends] = useState<Friend[]>([])
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [activeConversation, setActiveConversation] = useState<Conversation | null>(null)
  const [messages, setMessages] = useState<Message[]>([])
  const [messageDraft, setMessageDraft] = useState('')
  const [groupName, setGroupName] = useState('')
  const [selectedMemberIds, setSelectedMemberIds] = useState<number[]>([])
  const [notice, setNotice] = useState('使用 MVC REST 后端的前端工作台。')
  const [busy, setBusy] = useState(false)

  const refreshDashboard = useCallback(async (authToken: string) => {
    try {
      const [currentUser, userList, friendList, conversationList] = await Promise.all([
        api.me(authToken),
        api.users(authToken),
        api.friends(authToken),
        api.conversations(authToken),
      ])
      setMe(currentUser)
      setUsers(userList.filter((user) => user.id !== currentUser.id))
      setFriends(friendList)
      setConversations(conversationList)

      const nextConversation =
        activeConversation && conversationList.find((item) => item.id === activeConversation.id)
          ? conversationList.find((item) => item.id === activeConversation.id) ?? null
          : conversationList[0] ?? null
      setActiveConversation(nextConversation)
      if (nextConversation) {
        const nextMessages = await api.messages(authToken, nextConversation.id)
        setMessages([...nextMessages].reverse())
      } else {
        setMessages([])
      }
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '加载失败')
    }
  }, [activeConversation])

  useEffect(() => {
    if (!token) {
      setMe(null)
      return
    }

    void refreshDashboard(token)
  }, [token, refreshDashboard])

  async function handleAuth() {
    setBusy(true)
    try {
      if (mode === 'register') {
        await api.register(credentials)
        setNotice(`用户 ${credentials.username} 注册成功，请直接登录。`)
        setMode('login')
        return
      }
      const payload = await api.login({
        username: credentials.username,
        password: credentials.password,
      })
      localStorage.setItem('chat_token', payload.token)
      setToken(payload.token)
      setNotice(`欢迎回来，${payload.user.display_name}`)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '认证失败')
    } finally {
      setBusy(false)
    }
  }

  async function openDirectConversation(userId: number) {
    if (!token) return
    setBusy(true)
    try {
      const conversation = await api.createDirect(token, userId)
      await refreshDashboard(token)
      setActiveConversation(conversation)
      const nextMessages = await api.messages(token, conversation.id)
      setMessages([...nextMessages].reverse())
      setNotice('已打开单聊会话。')
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '创建单聊失败')
    } finally {
      setBusy(false)
    }
  }

  async function addFriend(userId: number) {
    if (!token) return
    setBusy(true)
    try {
      await api.addFriend(token, userId)
      await refreshDashboard(token)
      setNotice('好友添加成功。')
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '添加好友失败')
    } finally {
      setBusy(false)
    }
  }

  async function createGroupConversation() {
    if (!token || !groupName.trim() || selectedMemberIds.length < 2) {
      setNotice('群聊需要名称，并至少选择两名成员。')
      return
    }
    setBusy(true)
    try {
      const conversation = await api.createGroup(token, {
        name: groupName,
        member_ids: selectedMemberIds,
      })
      setGroupName('')
      setSelectedMemberIds([])
      await refreshDashboard(token)
      setActiveConversation(conversation)
      setNotice('群聊创建成功。')
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '创建群聊失败')
    } finally {
      setBusy(false)
    }
  }

  async function selectConversation(conversation: Conversation) {
    if (!token) return
    setActiveConversation(conversation)
    const nextMessages = await api.messages(token, conversation.id)
    setMessages([...nextMessages].reverse())
    await api.markRead(token, conversation.id)
    await refreshDashboard(token)
  }

  async function sendMessage() {
    if (!token || !activeConversation || !messageDraft.trim()) return
    setBusy(true)
    try {
      await api.sendMessage(token, activeConversation.id, messageDraft)
      setMessageDraft('')
      await selectConversation(activeConversation)
      setNotice('消息发送成功。')
    } catch (error) {
      setNotice(error instanceof Error ? error.message : '发送失败')
    } finally {
      setBusy(false)
    }
  }

  function logout() {
    localStorage.removeItem('chat_token')
    setToken('')
    setMessages([])
    setConversations([])
    setFriends([])
    setUsers([])
    setActiveConversation(null)
    setNotice('已退出登录。')
  }

  return (
    <div className="shell">
      <header className="hero">
        <div>
          <p className="eyebrow">Split Frontend + MVC Backend</p>
          <h1>Conversation Studio</h1>
          <p className="subtitle">
            用一个更克制的 REST MVC 后端，配一块面向产品演示的前端工作台。
          </p>
        </div>
        <a className="ghost-link" href={api.openapi()} target="_blank" rel="noreferrer">
          打开 OpenAPI
        </a>
      </header>

      <main className="layout">
        <section className="panel auth-panel">
          <div className="panel-header">
            <h2>身份入口</h2>
            <div className="pill-row">
              <button className={mode === 'login' ? 'pill active' : 'pill'} onClick={() => setMode('login')}>
                登录
              </button>
              <button className={mode === 'register' ? 'pill active' : 'pill'} onClick={() => setMode('register')}>
                注册
              </button>
            </div>
          </div>

          <label>
            用户名
            <input
              value={credentials.username}
              onChange={(event) => setCredentials({ ...credentials, username: event.target.value })}
              placeholder="例如 alice"
            />
          </label>
          {mode === 'register' && (
            <label>
              展示名
              <input
                value={credentials.display_name}
                onChange={(event) => setCredentials({ ...credentials, display_name: event.target.value })}
                placeholder="例如 Alice"
              />
            </label>
          )}
          <label>
            密码
            <input
              type="password"
              value={credentials.password}
              onChange={(event) => setCredentials({ ...credentials, password: event.target.value })}
              placeholder="pass123"
            />
          </label>
          <button className="cta" onClick={handleAuth} disabled={busy}>
            {mode === 'login' ? '进入工作台' : '创建账号'}
          </button>
          <p className="notice">{notice}</p>
          {me && (
            <div className="identity-card">
              <div>
                <strong>{me.display_name}</strong>
                <span>@{me.username}</span>
              </div>
              <button className="pill" onClick={logout}>
                退出
              </button>
            </div>
          )}
        </section>

        <section className="panel column-panel">
          <div className="panel-header">
            <h2>用户与好友</h2>
            <span>{users.length} 位用户</span>
          </div>
          <div className="stack-list">
            {users.map((user) => (
              <article key={user.id} className="list-card">
                <div>
                  <strong>{user.display_name}</strong>
                  <span>@{user.username}</span>
                </div>
                <div className="inline-actions">
                  <button className="mini-button" onClick={() => addFriend(user.id)}>
                    加好友
                  </button>
                  <button className="mini-button strong" onClick={() => openDirectConversation(user.id)}>
                    发消息
                  </button>
                </div>
              </article>
            ))}
          </div>

          <div className="panel-subsection">
            <div className="panel-header">
              <h3>好友列表</h3>
              <span>{friends.length} 位</span>
            </div>
            <div className="stack-list compact">
              {friends.length === 0 && <p className="empty">还没有好友，先从上方加人。</p>}
              {friends.map((friend) => (
                <button key={friend.id} className="friend-chip" onClick={() => openDirectConversation(friend.id)}>
                  <span>{friend.display_name}</span>
                  <small>@{friend.username}</small>
                </button>
              ))}
            </div>
          </div>
        </section>

        <section className="panel column-panel">
          <div className="panel-header">
            <h2>群聊工坊</h2>
            <span>选人建组</span>
          </div>
          <label>
            群聊名称
            <input value={groupName} onChange={(event) => setGroupName(event.target.value)} placeholder="例如 产品评审组" />
          </label>
          <div className="member-grid">
            {users.map((user) => {
              const checked = selectedMemberIds.includes(user.id)
              return (
                <button
                  key={user.id}
                  className={checked ? 'member-tile active' : 'member-tile'}
                  onClick={() =>
                    setSelectedMemberIds((current) =>
                      checked ? current.filter((item) => item !== user.id) : [...current, user.id],
                    )
                  }
                >
                  <strong>{user.display_name}</strong>
                  <small>@{user.username}</small>
                </button>
              )
            })}
          </div>
          <button className="cta muted" onClick={createGroupConversation} disabled={busy}>
            创建群聊
          </button>
        </section>
      </main>

      <section className="workspace">
        <aside className="panel conversation-panel">
          <div className="panel-header">
            <h2>会话列表</h2>
            <span>{conversations.length} 条</span>
          </div>
          <div className="stack-list">
            {conversations.map((conversation) => (
              <button
                key={conversation.id}
                className={activeConversation?.id === conversation.id ? 'conversation-card active' : 'conversation-card'}
                onClick={() => void selectConversation(conversation)}
              >
                <div className="conversation-topline">
                  <strong>{conversation.name}</strong>
                  <span>{conversation.type === 'group' ? '群聊' : '单聊'}</span>
                </div>
                <p>{conversation.last_message_preview ?? '暂无消息'}</p>
                <div className="conversation-meta">
                  <small>{conversation.last_message_at ?? '未开始'}</small>
                  {conversation.unread_count > 0 && <b>{conversation.unread_count}</b>}
                </div>
              </button>
            ))}
          </div>
        </aside>

        <section className="panel chat-panel">
          <div className="panel-header">
            <h2>{activeConversation ? activeConversation.name : '选择一个会话'}</h2>
            <span>{activeConversation ? `${activeConversation.member_count} 名成员` : '等待选择'}</span>
          </div>
          <div className="message-board">
            {messages.length === 0 && <p className="empty">选中会话后，这里会显示消息历史。</p>}
            {messages.map((message) => (
              <article
                key={message.id}
                className={message.sender_id === me?.id ? 'message-bubble self' : 'message-bubble'}
              >
                <p>{message.content}</p>
                <small>
                  #{message.sender_id} · {new Date(message.created_at).toLocaleString()}
                </small>
              </article>
            ))}
          </div>
          <div className="composer">
            <textarea
              value={messageDraft}
              onChange={(event) => setMessageDraft(event.target.value)}
              placeholder="输入消息，回车前先想清楚它会进入 SQLite。"
            />
            <button className="cta" onClick={sendMessage} disabled={!activeConversation || busy}>
              发送
            </button>
          </div>
        </section>
      </section>
    </div>
  )
}

export default App
