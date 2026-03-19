// ============ 基础类型 ============

// 用户类型
export interface User {
  id: number
  username: string
  display_name: string
  avatar_url: string
}

export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
  error_code?: string
}

// 认证响应
export interface AuthResponse {
  access_token: string
  refresh_token: string
  user: User
}

export interface AdminProfile {
  id: number
  username: string
  must_change_password: boolean
}

export interface AdminAuthResponse {
  access_token: string
  admin: AdminProfile
}

export interface AIProviderConfig {
  enabled: boolean
  api_key: string
  base_url: string
  models: string[]
}

export interface AIConfig {
  bot: {
    enabled: boolean
    username: string
    display_name: string
    system_prompt: string
    default_provider: string
    default_model: string
    context_messages: number
    async_timeout_seconds: number
  }
  providers: {
    zhipu: AIProviderConfig
    openai: AIProviderConfig
    anthropic: AIProviderConfig
  }
  audit: {
    enabled: boolean
    retention_days: number
  }
}

export interface MonitorServiceStatus {
  name: string
  healthy: boolean
  status: string
  error?: string
  checked_at: string
}

export interface MonitorOverview {
  services: MonitorServiceStatus[]
  total_requests: number
  client_errors: number
  server_errors: number
  average_latency_ms: number
  websocket_connections: number
  ai_retry_pending: number
  ai_retry_completed: number
  ai_retry_exhausted: number
  snapshot_at: string
}

export interface MonitorPoint {
  timestamp: string
  total_requests: number
  client_errors: number
  server_errors: number
  average_latency_ms: number
  websocket_connections: number
  ai_retry_pending: number
  ai_retry_completed: number
  ai_retry_exhausted: number
}

export interface MonitorTimeseries {
  points: MonitorPoint[]
}

export interface MessageJourneyStage {
  name: string
  occurred_at: string
  recipient_id: number
  note: string
}

export interface MessageJourney {
  message_id: number
  conversation_id: number
  client_msg_id: string
  sender_id: number
  message_type: string
  delivery_status: string
  created_at: string
  recalled_at: string
  stages: MessageJourneyStage[]
}

export interface MessageLookupResult {
  message_id: number
  conversation_id: number
  sender_id: number
  client_msg_id: string
}

export interface ConversationConsistencyMember {
  user_id: number
  username: string
  display_name: string
  avatar_url: string
  role: string
  last_read_seq: number
  unread_count: number
  current_cursor: number
  online: boolean
}

export interface ConversationConsistency {
  conversation_id: number
  last_message_seq: number
  last_message_at: string
  online_count: number
  current_event_lag: number
  members: ConversationConsistencyMember[]
}

// 会话信息
export interface Session {
  id: string
  device_id: string
  user_agent: string
  last_seen_at: string
  expires_at: string
  current: boolean
}

export interface Friend extends User {
  remark_name: string
  is_ai_bot?: boolean
}

export interface AIAuditLog {
  id: number
  user_id: number
  conversation_id: number
  request_id: string
  provider: string
  model: string
  status: string
  duration_ms: number
  input_tokens: number
  output_tokens: number
  total_tokens: number
  input_preview: string
  output_preview: string
  error_code: string
  error_message: string
  created_at: string
}

export interface AIAuditLogDetail extends AIAuditLog {
  request_payload_json: string
  response_payload_json: string
}

export interface AIRetryJob {
  id: number
  user_id: number
  conversation_id: number
  status: string
  attempt_count: number
  next_attempt_at: string
  last_error: string
  created_at: string
  updated_at: string
}

export interface AIRetryJobDetail extends AIRetryJob {}

// 好友请求
export interface FriendRequest {
  id: number
  status: 'pending' | 'accepted' | 'rejected'
  direction: 'incoming' | 'outgoing'
  message: string
  requester: User
  addressee: User
  created_at: string
  updated_at: string
}

// 群组请求（当前前端预留结构，待后端审批接口接入）
export interface GroupRequest {
  id: number
  status: 'pending' | 'accepted' | 'rejected'
  direction: 'incoming' | 'outgoing'
  group_name: string
  message: string
  actor: User
  created_at: string
  updated_at: string
}

// 会话/对话
export interface Conversation {
  id: number
  type: 'direct' | 'group'
  name: string
  announcement: string
  creator_id: number
  member_count: number
  pinned: boolean
  pinned_at: string
  is_muted: boolean
  draft: string
  last_read_seq: number
  unread_count: number
  last_message_seq: number
  last_message_sender: number
  last_message_preview: string
  last_message_type: string
  last_message_at: string
}

// 会话成员
export interface ConversationMember {
  user_id: number
  username: string
  display_name: string
  avatar_url: string
  remark_name: string
  role: 'owner' | 'admin' | 'member'
  last_read_seq: number
  joined_at: string
  online: boolean
}

// 消息
export interface Message {
  id: number
  conversation_id: number
  seq: number
  sender_id: number
  message_type: 'text' | 'image' | 'file' | 'system'
  content: string
  reply_to_message_id?: number
  reply_to?: MessageReference
  attachment?: Attachment
  client_msg_id: string
  status: 'sending' | 'sent' | 'delivered' | 'read' | 'failed'
  delivery_status: 'sending' | 'sent' | 'delivered' | 'read' | 'failed'
  created_at: string
  recalled_at: string
}

export interface Attachment {
  object_key: string
  url: string
  filename: string
  content_type: string
  size_bytes: number
}

export interface MessageReference {
  id: number
  sender_id: number
  message_type: 'text' | 'image' | 'file' | 'system'
  content: string
  attachment?: Attachment
  recalled_at: string
}

// 搜索结果 - 会话
export interface SearchConversation {
  conversation_id: number
  name: string
  type: string
  updated_at: string
}

// 搜索结果 - 消息
export interface SearchMessage {
  message_id: number
  conversation_id: number
  conversation_name: string
  sender_id: number
  message_type: string
  content: string
  created_at: string
}

// 搜索响应
export interface SearchResponse {
  conversations: SearchConversation[]
  messages: SearchMessage[]
  next_cursor: string
}

// 同步游标
export interface SyncCursor {
  cursor: number
}

// 同步事件
export interface SyncEvent {
  event_id: number
  event_type: string
  aggregate_id: string
  cursor: number
  type: string
  payload: Record<string, unknown>
  created_at: string
}

// 同步事件响应
export interface SyncEventResponse {
  events: SyncEvent[]
  next_cursor: number
  current_cursor: number
  has_more: boolean
}

// ============ WebSocket 事件类型 ============

export interface WsAuthOkEvent {
  type: 'auth.ok'
  payload: {
    user_id: number
    session_id: string
  }
}

export interface WsMessageNewEvent {
  type: 'message.persisted' | 'message.new'
  payload: {
    recipients: number[]
    conversation_id: number
    message: Message
    conversation: Conversation
  }
}

export interface WsConversationReadEvent {
  type: 'message.read' | 'conversation.read'
  payload: {
    recipients: number[]
    conversation_id: number
    reader_id: number
    last_read_seq: number
  }
}

export interface WsMessageAcceptedEvent {
  type: 'message.accepted'
  payload: {
    recipients: number[]
    conversation_id: number
    client_msg_id: string
    accepted_at: string
  }
}

export interface WsMessageDeliveredEvent {
  type: 'message.delivered'
  payload: {
    recipients: number[]
    conversation_id: number
    message_id: number
    client_msg_id: string
    delivery_status: Message['delivery_status']
    updated_at: string
  }
}

export interface WsTypingUpdatedEvent {
  type: 'typing.updated'
  payload: {
    recipients: number[]
    conversation_id: number
    user_id: number
    is_typing: boolean
  }
}

export interface WsFriendRequestEvent {
  type: 'friend.request'
  payload: {
    recipients: number[]
    request: FriendRequest
  }
}

export interface WsSyncNotifyEvent {
  type: 'sync.notify'
  payload: {
    recipients: number[]
  }
}

export interface WsMessageRecalledEvent {
  type: 'message.recalled'
  payload: {
    recipients: number[]
    conversation_id: number
    message_id: number
    recalled_at: string
  }
}

export type WsEvent =
  | WsAuthOkEvent
  | WsMessageAcceptedEvent
  | WsMessageNewEvent
  | WsMessageDeliveredEvent
  | WsMessageRecalledEvent
  | WsConversationReadEvent
  | WsTypingUpdatedEvent
  | WsFriendRequestEvent
  | WsSyncNotifyEvent

// ============ API 请求类型 ============

export interface RegisterRequest {
  username: string
  display_name: string
  password: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface AdminLoginRequest {
  username: string
  password: string
}

export interface AdminChangePasswordRequest {
  current_password: string
  new_password: string
}

export interface RefreshRequest {
  refresh_token: string
}

export interface LogoutRequest {
  refresh_token: string
}

export interface UpdateProfileRequest {
  display_name: string
  avatar_url?: string
}

export interface CreateFriendRequestBody {
  addressee_id: number
  message?: string
}

export interface UpdateFriendRemarkRequest {
  remark_name: string
}

export interface CreateDirectConversationRequest {
  target_user_id: number
}

export interface CreateGroupConversationRequest {
  name: string
  member_ids: number[]
}

export interface RenameConversationRequest {
  name: string
}

export interface AddConversationMembersRequest {
  member_ids: number[]
}

export interface TransferOwnerRequest {
  user_id: number
}

export interface UpdateConversationSettingsRequest {
  pinned?: boolean
  is_muted?: boolean
  draft?: string
  announcement?: string
}

export interface SendMessageRequest {
  message_type?: string
  content: string
  client_msg_id?: string
  reply_to_message_id?: number
  attachment?: Attachment
}

export interface UploadPresignRequest {
  filename: string
  content_type: string
  size_bytes: number
}

export interface UploadPresignResponse {
  object_key: string
  upload_path: string
  method: string
  headers: Record<string, string>
  public_url: string
}

export interface UploadCompleteRequest {
  object_key: string
  filename: string
  content_type: string
  size_bytes: number
}

export interface UploadCompleteResponse {
  attachment: Attachment
}

export interface MarkReadRequest {
  seq?: number
}

export interface TypingStatusRequest {
  is_typing: boolean
}

// ============ API 响应类型 ============

export interface ApiError {
  code: number
  message: string
  error_code?: string
}

export interface StatusResponse {
  status: string
}
