'use client'

import type {
  ApiEnvelope,
  AuthResponse,
  Conversation,
  ConversationMember,
  CreateDirectConversationRequest,
  CreateFriendRequestBody,
  CreateGroupConversationRequest,
  Friend,
  FriendRequest,
  LoginRequest,
  MarkReadRequest,
  Message,
  RefreshRequest,
  RegisterRequest,
  SearchResponse,
  Session,
  SendMessageRequest,
  StatusResponse,
  SyncCursor,
  SyncEventResponse,
  UpdateProfileRequest,
  User,
} from './types'

const AUTH_BASE_URL = process.env.NEXT_PUBLIC_AUTH_BASE_URL || 'http://127.0.0.1:8081'
const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://127.0.0.1:8082'
const REALTIME_BASE_URL = process.env.NEXT_PUBLIC_REALTIME_BASE_URL || 'ws://127.0.0.1:8083'
const DEVICE_ID = 'wechat-template-workbench'
const ACCESS_TOKEN_KEY = 'chat_access_token'
const REFRESH_TOKEN_KEY = 'chat_refresh_token'

let accessToken: string | null = null
let refreshToken: string | null = null
let refreshPromise: Promise<AuthResponse> | null = null

export class ApiRequestError extends Error {
  status: number
  errorCode?: string

  constructor(message: string, status: number, errorCode?: string) {
    super(message)
    this.status = status
    this.errorCode = errorCode
  }
}

function isBrowser() {
  return typeof window !== 'undefined'
}

export function initTokens() {
  if (!isBrowser()) {
    return
  }
  accessToken = window.sessionStorage.getItem(ACCESS_TOKEN_KEY)
  refreshToken = window.sessionStorage.getItem(REFRESH_TOKEN_KEY)
  window.localStorage.removeItem(ACCESS_TOKEN_KEY)
  window.localStorage.removeItem(REFRESH_TOKEN_KEY)
  window.localStorage.removeItem('access_token')
  window.localStorage.removeItem('refresh_token')
}

export function saveTokens(access: string, refresh?: string) {
  accessToken = access
  if (typeof refresh === 'string' && refresh.length > 0) {
    refreshToken = refresh
  }
  if (!isBrowser()) {
    return
  }
  window.sessionStorage.setItem(ACCESS_TOKEN_KEY, access)
  if (typeof refresh === 'string' && refresh.length > 0) {
    window.sessionStorage.setItem(REFRESH_TOKEN_KEY, refresh)
  }
}

export function clearTokens() {
  accessToken = null
  refreshToken = null
  if (!isBrowser()) {
    return
  }
  window.sessionStorage.removeItem(ACCESS_TOKEN_KEY)
  window.sessionStorage.removeItem(REFRESH_TOKEN_KEY)
  window.localStorage.removeItem(ACCESS_TOKEN_KEY)
  window.localStorage.removeItem(REFRESH_TOKEN_KEY)
  window.localStorage.removeItem('access_token')
  window.localStorage.removeItem('refresh_token')
}

export function getAccessToken() {
  return accessToken
}

export function getRefreshToken() {
  return refreshToken
}

async function parseEnvelope<T>(response: Response): Promise<T> {
  const raw = await response.text()
  let payload: ApiEnvelope<T> | null = null

  try {
    payload = JSON.parse(raw) as ApiEnvelope<T>
  } catch {
    const snippet = raw.trim().slice(0, 120)
    throw new ApiRequestError(snippet || '服务返回了无法解析的响应', response.status)
  }

  if (!response.ok || payload.code !== 0) {
    throw new ApiRequestError(payload.message || '请求失败', response.status, payload.error_code)
  }

  return payload.data
}

async function refreshAccessToken() {
  if (!refreshToken) {
    throw new ApiRequestError('登录状态已失效，请重新登录', 401)
  }

  if (!refreshPromise) {
    refreshPromise = request<AuthResponse>(
      AUTH_BASE_URL,
      '/api/v1/auth/refresh',
      {
        method: 'POST',
        body: JSON.stringify({ refresh_token: refreshToken } satisfies RefreshRequest),
      },
      false,
      false,
    ).finally(() => {
      refreshPromise = null
    })
  }

  const next = await refreshPromise
  saveTokens(next.access_token, next.refresh_token)
  return next
}

async function request<T>(
  baseUrl: string,
  path: string,
  init: RequestInit = {},
  requireAuth = true,
  allowRefresh = true,
): Promise<T> {
  let response: Response
  try {
    response = await fetch(`${baseUrl}${path}`, {
      ...init,
      headers: {
        'Content-Type': 'application/json',
        ...(requireAuth && accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
        ...(init.headers ?? {}),
      },
    })
  } catch {
    throw new ApiRequestError(`无法连接到服务：${baseUrl}`, 0)
  }

  if (response.status === 401 && requireAuth && allowRefresh && refreshToken) {
    try {
      await refreshAccessToken()
    } catch {
      clearTokens()
      throw new ApiRequestError('登录状态已失效，请重新登录', 401)
    }

    return request<T>(baseUrl, path, init, requireAuth, false)
  }

  return parseEnvelope<T>(response)
}

function searchPath(q: string, scope: 'all' | 'messages' | 'conversations', conversationId?: number) {
  const query = new URLSearchParams({ q, scope, limit: '8' })
  if (conversationId) {
    query.set('conversation_id', String(conversationId))
  }
  return `/api/v1/search?${query.toString()}`
}

export async function register(data: RegisterRequest) {
  return request<User>(AUTH_BASE_URL, '/api/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify(data),
  }, false, false)
}

export async function login(data: LoginRequest) {
  const result = await request<AuthResponse>(
    AUTH_BASE_URL,
    '/api/v1/auth/login',
    {
      method: 'POST',
      headers: {
        'X-Device-Id': DEVICE_ID,
      },
      body: JSON.stringify(data),
    },
    false,
    false,
  )
  saveTokens(result.access_token, result.refresh_token)
  return result
}

export async function logout() {
  const currentRefreshToken = getRefreshToken()
  try {
    if (currentRefreshToken) {
      await request<StatusResponse>(
        AUTH_BASE_URL,
        '/api/v1/auth/logout',
        {
          method: 'POST',
          body: JSON.stringify({ refresh_token: currentRefreshToken }),
        },
        true,
        false,
      )
    }
  } finally {
    clearTokens()
  }
}

export async function logoutAll() {
  const result = await request<StatusResponse>(
    AUTH_BASE_URL,
    '/api/v1/auth/logout-all',
    { method: 'POST' },
    true,
    false,
  )
  clearTokens()
  return result
}

export function getCurrentUser() {
  return request<User>(API_BASE_URL, '/api/v1/users/me')
}

export function updateProfile(data: UpdateProfileRequest) {
  return request<User>(API_BASE_URL, '/api/v1/users/me', {
    method: 'PATCH',
    body: JSON.stringify(data),
  })
}

export function getSessions() {
  return request<Session[]>(AUTH_BASE_URL, '/api/v1/auth/sessions')
}

export function getUsers() {
  return request<User[]>(API_BASE_URL, '/api/v1/users')
}

export function getFriends() {
  return request<Friend[]>(API_BASE_URL, '/api/v1/friends')
}

export async function getFriendRequests() {
  const requests = await request<FriendRequest[]>(API_BASE_URL, '/api/v1/friend-requests')
  return requests.map((item): FriendRequest => {
    const status: FriendRequest['status'] =
      item.status === 'accepted' ? 'accepted' : item.status === 'rejected' ? 'rejected' : 'pending'
    return {
      ...item,
      status,
    }
  })
}

export function createFriendRequest(data: CreateFriendRequestBody) {
  return request<FriendRequest>(API_BASE_URL, '/api/v1/friend-requests', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function approveFriendRequest(id: number) {
  return request<StatusResponse>(API_BASE_URL, `/api/v1/friend-requests/${id}/approve`, {
    method: 'POST',
  })
}

export function rejectFriendRequest(id: number) {
  return request<StatusResponse>(API_BASE_URL, `/api/v1/friend-requests/${id}/reject`, {
    method: 'POST',
  })
}

export function getConversations() {
  return request<Conversation[]>(API_BASE_URL, '/api/v1/conversations')
}

export function createDirectConversation(data: CreateDirectConversationRequest) {
  return request<Conversation>(API_BASE_URL, '/api/v1/conversations/direct', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function createGroupConversation(data: CreateGroupConversationRequest) {
  return request<Conversation>(API_BASE_URL, '/api/v1/conversations/group', {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function getConversationMembers(id: number) {
  return request<ConversationMember[]>(API_BASE_URL, `/api/v1/conversations/${id}/members`)
}

export function pinConversation(id: number, pinned: boolean) {
  return request<StatusResponse>(API_BASE_URL, `/api/v1/conversations/${id}/pin`, {
    method: 'POST',
    body: JSON.stringify({ pinned }),
  })
}

export function leaveConversation(id: number) {
  return request<StatusResponse>(API_BASE_URL, `/api/v1/conversations/${id}/leave`, {
    method: 'POST',
  })
}

export function getMessages(conversationId: number, cursor = '', limit = 20) {
  const suffix = cursor ? `?cursor=${encodeURIComponent(cursor)}&limit=${limit}` : `?limit=${limit}`
  return request<Message[]>(API_BASE_URL, `/api/v1/conversations/${conversationId}/messages${suffix}`)
}

export function sendMessage(conversationId: number, data: SendMessageRequest) {
  return request<Message>(API_BASE_URL, `/api/v1/conversations/${conversationId}/messages`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function markAsRead(conversationId: number, data: MarkReadRequest = {}) {
  return request<StatusResponse>(API_BASE_URL, `/api/v1/conversations/${conversationId}/read`, {
    method: 'POST',
    body: JSON.stringify(data),
  })
}

export function search(q: string, scope: 'all' | 'messages' | 'conversations' = 'all', conversationId?: number) {
  return request<SearchResponse>(API_BASE_URL, searchPath(q, scope, conversationId))
}

export function getSyncCursor() {
  return request<SyncCursor>(API_BASE_URL, '/api/v1/sync/cursor')
}

export function getSyncEvents(cursor = 0, limit = 50) {
  const query = new URLSearchParams({ cursor: String(cursor), limit: String(limit) })
  return request<SyncEventResponse>(API_BASE_URL, `/api/v1/sync/events?${query.toString()}`)
}

export function getWebSocketUrl() {
  const token = getAccessToken()
  const base = REALTIME_BASE_URL.replace(/^http/, 'ws').replace(/\/$/, '')
  return `${base}/ws?token=${encodeURIComponent(token ?? '')}`
}
