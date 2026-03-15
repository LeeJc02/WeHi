import type { ApiEnvelope, Conversation, Friend, Message, User } from './types'

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://127.0.0.1:8081'

async function request<T>(path: string, init: RequestInit = {}, token?: string): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init.headers ?? {}),
    },
  })

  const payload = (await response.json()) as ApiEnvelope<T>
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || 'Request failed')
  }
  return payload.data
}

export const api = {
  register(input: { username: string; display_name: string; password: string }) {
    return request<User>('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },
  login(input: { username: string; password: string }) {
    return request<{ token: string; user: User }>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },
  me(token: string) {
    return request<User>('/api/v1/users/me', {}, token)
  },
  users(token: string) {
    return request<User[]>('/api/v1/users', {}, token)
  },
  friends(token: string) {
    return request<Friend[]>('/api/v1/friends', {}, token)
  },
  addFriend(token: string, friend_id: number) {
    return request<{ friend_id: number }>('/api/v1/friends', {
      method: 'POST',
      body: JSON.stringify({ friend_id }),
    }, token)
  },
  conversations(token: string) {
    return request<Conversation[]>('/api/v1/conversations', {}, token)
  },
  createDirect(token: string, target_user_id: number) {
    return request<Conversation>('/api/v1/conversations/direct', {
      method: 'POST',
      body: JSON.stringify({ target_user_id }),
    }, token)
  },
  createGroup(token: string, input: { name: string; member_ids: number[] }) {
    return request<Conversation>('/api/v1/conversations/group', {
      method: 'POST',
      body: JSON.stringify(input),
    }, token)
  },
  messages(token: string, conversationId: number) {
    return request<Message[]>(`/api/v1/conversations/${conversationId}/messages?limit=50`, {}, token)
  },
  sendMessage(token: string, conversationId: number, content: string) {
    return request<Message>(`/api/v1/conversations/${conversationId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ content }),
    }, token)
  },
  markRead(token: string, conversationId: number) {
    return request<{ conversation_id: number; status: string }>(`/api/v1/conversations/${conversationId}/read`, {
      method: 'POST',
    }, token)
  },
  openapi() {
    return `${API_BASE}/openapi.yaml`
  },
}
