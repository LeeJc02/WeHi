export type User = {
  id: number
  username: string
  display_name: string
}

export type Friend = {
  id: number
  username: string
  display_name: string
}

export type Conversation = {
  id: number
  type: 'direct' | 'group'
  name: string
  creator_id: number
  member_count: number
  unread_count: number
  last_message_id?: number
  last_message_sender?: number
  last_message_preview?: string
  last_message_at?: string
}

export type Message = {
  id: number
  conversation_id: number
  sender_id: number
  content: string
  created_at: string
}

export type ApiEnvelope<T> = {
  code: number
  message: string
  data: T
}
