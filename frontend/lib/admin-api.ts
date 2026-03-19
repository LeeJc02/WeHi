'use client'

import type {
  AIConfig,
  AIAuditLog,
  AIAuditLogDetail,
  AdminAuthResponse,
  AdminChangePasswordRequest,
  AdminLoginRequest,
  AdminProfile,
  AIRetryJob,
  AIRetryJobDetail,
  ApiEnvelope,
  ConversationConsistency,
  MessageLookupResult,
  MessageJourney,
  MonitorOverview,
  MonitorTimeseries,
  SyncEvent,
} from './types'

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://127.0.0.1:8082'
const ADMIN_ACCESS_TOKEN_KEY = 'admin_access_token'

let adminAccessToken: string | null = null

export class AdminApiError extends Error {
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

export function initAdminToken() {
  if (!isBrowser()) {
    return
  }
  adminAccessToken = window.sessionStorage.getItem(ADMIN_ACCESS_TOKEN_KEY)
}

export function saveAdminToken(token: string) {
  adminAccessToken = token
  if (!isBrowser()) {
    return
  }
  window.sessionStorage.setItem(ADMIN_ACCESS_TOKEN_KEY, token)
}

export function clearAdminToken() {
  adminAccessToken = null
  if (!isBrowser()) {
    return
  }
  window.sessionStorage.removeItem(ADMIN_ACCESS_TOKEN_KEY)
}

export function getAdminToken() {
  return adminAccessToken
}

async function parseEnvelope<T>(response: Response): Promise<T> {
  const raw = await response.text()
  let payload: ApiEnvelope<T> | null = null
  try {
    payload = JSON.parse(raw) as ApiEnvelope<T>
  } catch {
    throw new AdminApiError(raw || '管理端返回了无法解析的响应', response.status)
  }
  if (!response.ok || payload.code !== 0) {
    throw new AdminApiError(payload.message || '请求失败', response.status, payload.error_code)
  }
  return payload.data
}

async function request<T>(path: string, init: RequestInit = {}, requireAuth = true): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(requireAuth && adminAccessToken ? { Authorization: `Bearer ${adminAccessToken}` } : {}),
      ...(init.headers ?? {}),
    },
  })
  if (response.status === 401 && requireAuth) {
    clearAdminToken()
  }
  return parseEnvelope<T>(response)
}

export async function adminLogin(data: AdminLoginRequest) {
  const result = await request<AdminAuthResponse>('/api/v1/admin/auth/login', {
    method: 'POST',
    body: JSON.stringify(data),
  }, false)
  saveAdminToken(result.access_token)
  return result
}

export function getAdminMe() {
  return request<AdminProfile>('/api/v1/admin/auth/me')
}

export async function adminChangePassword(data: AdminChangePasswordRequest) {
  const result = await request<{ status: string }>('/api/v1/admin/auth/change-password', {
    method: 'POST',
    body: JSON.stringify(data),
  })
  return result
}

export function getAIConfig() {
  return request<AIConfig>('/api/v1/admin/ai-config')
}

export function updateAIConfig(data: AIConfig) {
  return request<AIConfig>('/api/v1/admin/ai-config', {
    method: 'PUT',
    body: JSON.stringify(data),
  })
}

export function getAIRetryJobs(query: {
  limit?: number
  status?: string
} = {}) {
  const params = new URLSearchParams()
  if (query.limit) params.set('limit', String(query.limit))
  if (query.status) params.set('status', query.status)
  const suffix = params.toString() ? `?${params.toString()}` : ''
  return request<AIRetryJob[]>(`/api/v1/admin/ai/retry-jobs${suffix}`)
}

export function getAIRetryJobDetail(id: number) {
  return request<AIRetryJobDetail>(`/api/v1/admin/ai/retry-jobs/${id}`)
}

export function retryAIJobNow(id: number) {
  return request<{ status: string }>(`/api/v1/admin/ai/retry-jobs/${id}/retry-now`, {
    method: 'POST',
  })
}

export function retryAIJobs(ids: number[]) {
  return request<{ status: string }>('/api/v1/admin/ai/retry-jobs/retry-batch', {
    method: 'POST',
    body: JSON.stringify({ ids }),
  })
}

export function cleanupAIRetryJobs(statuses: string[]) {
  return request<{ status: string }>('/api/v1/admin/ai/retry-jobs/cleanup', {
    method: 'POST',
    body: JSON.stringify({ statuses }),
  })
}

export function getMonitorOverview() {
  return request<MonitorOverview>('/api/v1/admin/monitor/overview')
}

export function getMonitorTimeseries() {
  return request<MonitorTimeseries>('/api/v1/admin/monitor/timeseries')
}

export function getAIAuditLogs(query: {
  limit?: number
  status?: string
  provider?: string
  model?: string
  userId?: number
  conversationId?: number
} = {}) {
  const params = new URLSearchParams()
  if (query.limit) params.set('limit', String(query.limit))
  if (query.status) params.set('status', query.status)
  if (query.provider) params.set('provider', query.provider)
  if (query.model) params.set('model', query.model)
  if (query.userId) params.set('user_id', String(query.userId))
  if (query.conversationId) params.set('conversation_id', String(query.conversationId))
  const suffix = params.toString() ? `?${params.toString()}` : ''
  return request<AIAuditLog[]>(`/api/v1/admin/audit/ai-calls${suffix}`)
}

export function getAIAuditLogDetail(id: number) {
  return request<AIAuditLogDetail>(`/api/v1/admin/audit/ai-calls/${id}`)
}

export function getMessageJourney(messageId: number) {
  return request<MessageJourney>(`/api/v1/admin/message-journey/${messageId}`)
}

export function resolveMessageByClientMsgID(query: {
  clientMsgId: string
  senderId?: number
  conversationId?: number
}) {
  const params = new URLSearchParams()
  params.set('client_msg_id', query.clientMsgId)
  if (query.senderId) params.set('sender_id', String(query.senderId))
  if (query.conversationId) params.set('conversation_id', String(query.conversationId))
  return request<MessageLookupResult>(`/api/v1/admin/messages/resolve?${params.toString()}`)
}

export function getConversationConsistency(conversationId: number) {
  return request<ConversationConsistency>(`/api/v1/admin/conversations/${conversationId}/consistency`)
}

export function getConversationEvents(conversationId: number, limit = 100) {
  return request<SyncEvent[]>(`/api/v1/admin/conversations/${conversationId}/events?limit=${limit}`)
}

export function triggerSearchReindex() {
  return request<{ status: string }>('/api/v1/admin/search/reindex', {
    method: 'POST',
  })
}
