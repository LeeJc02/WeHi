// ============ 基础类型 ============

// 用户类型
export interface User {
  id: number
  username: string
  display_name: string
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

// 会话信息
export interface Session {
  id: string
  device_id: string
  user_agent: string
  last_seen_at: string
  expires_at: string
  current: boolean
}

// 好友（与User相同结构）
export type Friend = User

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
  creator_id: number
  member_count: number
  pinned: boolean
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
  client_msg_id: string
  status: 'sending' | 'sent' | 'delivered' | 'read' | 'failed'
  created_at: string
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
  type: 'message.new'
  payload: {
    recipients: number[]
    conversation_id: number
    message: Message
    conversation: Conversation
  }
}

export interface WsConversationReadEvent {
  type: 'conversation.read'
  payload: {
    recipients: number[]
    conversation_id: number
    reader_id: number
    last_read_seq: number
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

export type WsEvent =
  | WsAuthOkEvent
  | WsMessageNewEvent
  | WsConversationReadEvent
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

export interface RefreshRequest {
  refresh_token: string
}

export interface LogoutRequest {
  refresh_token: string
}

export interface UpdateProfileRequest {
  display_name: string
}

export interface CreateFriendRequestBody {
  addressee_id: number
  message?: string
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

export interface PinConversationRequest {
  pinned: boolean
}

export interface SendMessageRequest {
  message_type?: string
  content: string
  client_msg_id?: string
}

export interface MarkReadRequest {
  seq?: number
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
