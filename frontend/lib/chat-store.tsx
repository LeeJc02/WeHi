'use client'

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from 'react'
import { useWebSocket } from '@/hooks/use-websocket'
import { useAuth } from './auth-context'
import {
  approveFriendRequest as apiApproveFriendRequest,
  addConversationMembers as apiAddConversationMembers,
  createDirectConversation,
  createGroupConversation as apiCreateGroupConversation,
  createFriendRequest as apiCreateFriendRequest,
  getConversationMembers,
  getConversations,
  getFriendRequests,
  getFriends,
  getMessages,
  getSyncCursor,
  getSyncEvents,
  getUsers,
  leaveConversation as apiLeaveConversation,
  markAsRead as apiMarkAsRead,
  recallMessage as apiRecallMessage,
  removeConversationMember as apiRemoveConversationMember,
  rejectFriendRequest as apiRejectFriendRequest,
  renameConversation as apiRenameConversation,
  sendMessage as apiSendMessage,
  updateConversationSettings as apiUpdateConversationSettings,
  updateFriendRemark as apiUpdateFriendRemark,
  updateTypingStatus as apiUpdateTypingStatus,
} from './api'
import type {
  Conversation,
  ConversationMember,
  Friend,
  FriendRequest,
  GroupRequest,
  Message,
  SyncEvent,
  User,
} from './types'

type ActiveView = 'chat' | 'contacts' | 'settings'

interface ChatStoreContextType {
  activeView: ActiveView
  setActiveView: (view: ActiveView) => void
  conversations: Conversation[]
  currentConversation: Conversation | null
  currentMessages: Message[]
  currentMembers: ConversationMember[]
  setCurrentConversation: (conversation: Conversation | null) => void
  loadConversations: () => Promise<void>
  loadMessages: (conversationId: number) => Promise<Message[]>
  sendMessage: (payload: {
    content: string
    messageType?: Message['message_type']
    replyToMessageId?: number
    attachment?: Message['attachment']
    clientMsgId?: string
    existingMessageId?: number
  }) => Promise<void>
  retryMessage: (messageId: number) => Promise<void>
  recallMessage: (messageId: number) => Promise<void>
  sendTyping: (conversationId: number, isTyping: boolean) => Promise<void>
  typingUserIds: number[]
  markAsRead: (conversationId: number, seq?: number) => Promise<void>
  updateConversationSettings: (conversationId: number, updates: Partial<Conversation>) => Promise<void>
  renameConversation: (conversationId: number, name: string) => Promise<void>
  addConversationMembers: (conversationId: number, memberIds: number[]) => Promise<void>
  removeConversationMember: (conversationId: number, userId: number) => Promise<void>
  leaveConversation: (conversationId: number) => Promise<void>
  friends: Friend[]
  friendRequests: FriendRequest[]
  groupRequests: GroupRequest[]
  allUsers: User[]
  loadFriends: () => Promise<void>
  loadFriendRequests: () => Promise<void>
  loadAllUsers: () => Promise<void>
  approveFriendRequest: (id: number) => Promise<void>
  rejectFriendRequest: (id: number) => Promise<void>
  sendFriendRequest: (userId: number, message?: string) => Promise<void>
  updateFriendRemark: (friendId: number, remarkName: string) => Promise<void>
  startDirectChat: (userId: number) => Promise<void>
  createGroupChat: (name: string, memberIds: number[]) => Promise<void>
  searchQuery: string
  setSearchQuery: (query: string) => void
  isLoading: boolean
  isConnected: boolean
}

const ChatStoreContext = createContext<ChatStoreContextType | null>(null)

function sortConversations(rows: Conversation[]) {
  return [...rows].sort((a, b) => {
    if (a.pinned !== b.pinned) {
      return a.pinned ? -1 : 1
    }
    if (a.pinned_at !== b.pinned_at) {
      return new Date(b.pinned_at || 0).getTime() - new Date(a.pinned_at || 0).getTime()
    }
    return new Date(b.last_message_at || 0).getTime() - new Date(a.last_message_at || 0).getTime()
  })
}

function sortMessages(rows: Message[]) {
  return [...rows].sort((a, b) => {
    if (a.seq !== b.seq) {
      return a.seq - b.seq
    }
    return new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
  })
}

function resolveDirectName(conversation: Conversation, currentUserId?: number, members: ConversationMember[] = []) {
  if (conversation.name || conversation.type !== 'direct') {
    return conversation
  }
  const other = members.find((member) => member.user_id !== currentUserId)
  if (!other) {
    return conversation
  }
  return {
    ...conversation,
    name: other.remark_name || other.display_name || other.username || '私聊',
  }
}

function mergeConversation(previous: Conversation | undefined, incoming: Conversation) {
  if (!previous) {
    return incoming
  }
  return {
    ...previous,
    ...incoming,
    name: incoming.name || previous.name,
  }
}

function messagePreview(message: Pick<Message, 'message_type' | 'content' | 'attachment' | 'recalled_at'>) {
  if (message.recalled_at) {
    return '[消息已撤回]'
  }
  if (message.message_type === 'image') {
    return '[图片]'
  }
  if (message.message_type === 'file') {
    return `[文件] ${message.attachment?.filename || message.content || ''}`.trim()
  }
  return message.content
}

function replaceMessage(rows: Message[], next: Message, fallbackId?: number) {
  const replaced = rows.map((item) => {
    if (item.id === next.id) {
      return next
    }
    if (item.client_msg_id && item.client_msg_id === next.client_msg_id) {
      return next
    }
    if (fallbackId !== undefined && item.id === fallbackId) {
      return next
    }
    return item
  })
  if (replaced.some((item) => item.id === next.id)) {
    return sortMessages(replaced)
  }
  return sortMessages([...replaced, next])
}

export function ChatStoreProvider({ children }: { children: ReactNode }) {
  const { user, isAuthenticated } = useAuth()
  const { isConnected, subscribe } = useWebSocket(isAuthenticated)
  const membersCacheRef = useRef<Record<number, ConversationMember[]>>({})
  const conversationsRef = useRef<Conversation[]>([])
  const currentMessagesRef = useRef<Message[]>([])
  const currentConversationIdRef = useRef<number | null>(null)
  const currentUserIdRef = useRef<number | undefined>(undefined)
  const syncCursorRef = useRef(0)
  const syncReadyRef = useRef(false)
  const syncReplayInFlightRef = useRef(false)
  const syncReplayPendingRef = useRef(false)
  const [activeView, setActiveView] = useState<ActiveView>('chat')
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [currentConversation, setCurrentConversation] = useState<Conversation | null>(null)
  const [currentMessages, setCurrentMessages] = useState<Message[]>([])
  const [currentMembers, setCurrentMembers] = useState<ConversationMember[]>([])
  const [friends, setFriends] = useState<Friend[]>([])
  const [friendRequests, setFriendRequests] = useState<FriendRequest[]>([])
  const [groupRequests, setGroupRequests] = useState<GroupRequest[]>([])
  const [allUsers, setAllUsers] = useState<User[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [typingUserIds, setTypingUserIds] = useState<number[]>([])

  useEffect(() => {
    conversationsRef.current = conversations
  }, [conversations])

  useEffect(() => {
    currentMessagesRef.current = currentMessages
  }, [currentMessages])

  useEffect(() => {
    currentConversationIdRef.current = currentConversation?.id ?? null
  }, [currentConversation?.id])

  useEffect(() => {
    currentUserIdRef.current = user?.id
  }, [user?.id])

  useEffect(() => {
    setTypingUserIds([])
  }, [currentConversation?.id])

  const cacheMembers = useCallback((conversationId: number, members: ConversationMember[]) => {
    membersCacheRef.current = {
      ...membersCacheRef.current,
      [conversationId]: members,
    }
  }, [])

  const getCachedMembers = useCallback((conversationId: number) => {
    return membersCacheRef.current[conversationId] ?? []
  }, [])

  const loadConversationMembers = useCallback(async (conversationId: number, force = false) => {
    if (!force) {
      const cached = membersCacheRef.current[conversationId]
      if (cached) {
        return cached
      }
    }
    const members = await getConversationMembers(conversationId)
    cacheMembers(conversationId, members)
    return members
  }, [cacheMembers])

  const decorateConversations = useCallback(async (rows: Conversation[]) => {
    if (!user?.id) {
      return sortConversations(rows)
    }

    const targets = rows.filter((item) => item.type === 'direct' && !item.name)
    if (targets.length === 0) {
      return sortConversations(rows)
    }

    const decoratedMap = new Map<number, Conversation>()
    await Promise.all(
      targets.map(async (item) => {
        try {
          const members = await loadConversationMembers(item.id)
          decoratedMap.set(item.id, resolveDirectName(item, user.id, members))
        } catch {
          decoratedMap.set(item.id, item)
        }
      }),
    )

    return sortConversations(rows.map((item) => decoratedMap.get(item.id) ?? item))
  }, [loadConversationMembers, user?.id])

  const updateConversationState = useCallback((updater: (rows: Conversation[]) => Conversation[]) => {
    setConversations((previous) => {
      const next = sortConversations(updater(previous))
      setCurrentConversation((current) => {
        if (!current) {
          return current
        }
        return next.find((item) => item.id === current.id) ?? current
      })
      return next
    })
  }, [])

  const loadConversations = useCallback(async () => {
    if (!isAuthenticated) {
      return
    }
    const data = await getConversations()
    const next = await decorateConversations(data)
    setConversations(next)
    setCurrentConversation((current) => {
      if (!current) {
        return current
      }
      return next.find((item) => item.id === current.id) ?? null
    })
  }, [decorateConversations, isAuthenticated])

  const loadMessages = useCallback(async (conversationId: number) => {
    setIsLoading(true)
    try {
      const [messageRows, memberRows] = await Promise.all([
        getMessages(conversationId),
        loadConversationMembers(conversationId, true),
      ])
      const sortedMessages = sortMessages(messageRows)
      setCurrentMessages(sortedMessages)
      setCurrentMembers(memberRows)
      if (user?.id) {
        updateConversationState((rows) =>
          rows.map((row) =>
            row.id === conversationId ? resolveDirectName(row, user.id, memberRows) : row,
          ),
        )
      }
      return sortedMessages
    } finally {
      setIsLoading(false)
    }
  }, [loadConversationMembers, updateConversationState, user?.id])

  const markAsRead = useCallback(async (conversationId: number, seq?: number) => {
    await apiMarkAsRead(conversationId, seq ? { seq } : {})
    updateConversationState((rows) =>
      rows.map((row) =>
        row.id === conversationId
          ? {
              ...row,
              unread_count: 0,
              last_read_seq: seq ?? row.last_read_seq,
            }
          : row,
      ),
    )
  }, [updateConversationState])

  const sendMessage = useCallback(async (payload: {
    content: string
    messageType?: Message['message_type']
    replyToMessageId?: number
    attachment?: Message['attachment']
    clientMsgId?: string
    existingMessageId?: number
  }) => {
    if (!currentConversation) {
      return
    }
    const clientMsgId = payload.clientMsgId ?? `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
    const replyTo = payload.replyToMessageId
      ? currentMessagesRef.current.find((item) => item.id === payload.replyToMessageId)
      : undefined
    const optimisticMessage: Message = {
      id: payload.existingMessageId ?? -Date.now(),
      conversation_id: currentConversation.id,
      seq: (currentMessagesRef.current.at(-1)?.seq ?? 0) + 1,
      sender_id: user?.id ?? 0,
      message_type: payload.messageType ?? 'text',
      content: payload.content,
      reply_to_message_id: payload.replyToMessageId,
      reply_to: replyTo
        ? {
            id: replyTo.id,
            sender_id: replyTo.sender_id,
            message_type: replyTo.message_type,
            content: replyTo.content,
            attachment: replyTo.attachment,
            recalled_at: replyTo.recalled_at,
          }
        : undefined,
      attachment: payload.attachment,
      client_msg_id: clientMsgId,
      status: 'sending',
      delivery_status: 'sending',
      created_at: new Date().toISOString(),
      recalled_at: '',
    }

    setCurrentMessages((previous) => replaceMessage(previous, optimisticMessage, payload.existingMessageId))
    updateConversationState((rows) => {
      const nextConversation = rows.find((row) => row.id === currentConversation.id)
      const merged = {
        ...(nextConversation ?? currentConversation),
        last_message_at: optimisticMessage.created_at,
        last_message_preview: messagePreview(optimisticMessage),
        last_message_sender: optimisticMessage.sender_id,
        last_message_seq: optimisticMessage.seq,
        last_message_type: optimisticMessage.message_type,
      }
      return [...rows.filter((row) => row.id !== currentConversation.id), merged]
    })

    try {
      const message = await apiSendMessage(currentConversation.id, {
        content: payload.content,
        message_type: payload.messageType ?? 'text',
        client_msg_id: clientMsgId,
        reply_to_message_id: payload.replyToMessageId,
        attachment: payload.attachment,
      })
      setCurrentMessages((previous) => replaceMessage(previous, message, optimisticMessage.id))
      updateConversationState((rows) => {
        const nextConversation = rows.find((row) => row.id === currentConversation.id)
        const merged = {
          ...(nextConversation ?? currentConversation),
          last_message_at: message.created_at,
          last_message_preview: messagePreview(message),
          last_message_sender: message.sender_id,
          last_message_seq: message.seq,
          last_message_type: message.message_type,
        }
        return [...rows.filter((row) => row.id !== currentConversation.id), merged]
      })
    } catch (error) {
      setCurrentMessages((previous) =>
        previous.map((item) =>
          item.client_msg_id === clientMsgId
            ? { ...item, status: 'failed', delivery_status: 'failed' }
            : item,
        ),
      )
      throw error
    }
  }, [currentConversation, updateConversationState, user?.id])

  const retryMessage = useCallback(async (messageId: number) => {
    const target = currentMessagesRef.current.find((item) => item.id === messageId)
    if (!target || target.status !== 'failed') {
      return
    }
    await sendMessage({
      content: target.content,
      messageType: target.message_type,
      replyToMessageId: target.reply_to_message_id,
      attachment: target.attachment,
      clientMsgId: target.client_msg_id,
      existingMessageId: target.id,
    })
  }, [sendMessage])

  const recallMessage = useCallback(async (messageId: number) => {
    await apiRecallMessage(messageId)
  }, [])

  const sendTyping = useCallback(async (conversationId: number, isTyping: boolean) => {
    await apiUpdateTypingStatus(conversationId, { is_typing: isTyping })
  }, [])

  const updateConversationSettings = useCallback(async (conversationId: number, updates: Partial<Conversation>) => {
    const conversation = await apiUpdateConversationSettings(conversationId, {
      pinned: updates.pinned,
      is_muted: updates.is_muted,
      draft: updates.draft,
      announcement: updates.announcement,
    })
    updateConversationState((rows) => [
      ...rows.filter((row) => row.id !== conversation.id),
      mergeConversation(rows.find((row) => row.id === conversation.id), conversation),
    ])
    setCurrentConversation((current) => {
      if (!current || current.id !== conversation.id) {
        return current
      }
      return mergeConversation(current, conversation)
    })
  }, [updateConversationState])

  const renameConversation = useCallback(async (conversationId: number, name: string) => {
    const conversation = await apiRenameConversation(conversationId, name)
    updateConversationState((rows) => [
      ...rows.filter((row) => row.id !== conversation.id),
      mergeConversation(rows.find((row) => row.id === conversation.id), conversation),
    ])
  }, [updateConversationState])

  const addConversationMembers = useCallback(async (conversationId: number, memberIds: number[]) => {
    await apiAddConversationMembers(conversationId, memberIds)
    const members = await loadConversationMembers(conversationId, true)
    setCurrentMembers((current) => (currentConversationIdRef.current === conversationId ? members : current))
    await loadConversations()
  }, [loadConversationMembers, loadConversations])

  const removeConversationMember = useCallback(async (conversationId: number, userId: number) => {
    await apiRemoveConversationMember(conversationId, userId)
    const isCurrentConversation = currentConversationIdRef.current === conversationId
    if (isCurrentConversation && currentUserIdRef.current === userId) {
      setCurrentConversation(null)
      setCurrentMessages([])
      setCurrentMembers([])
    } else {
      const members = await loadConversationMembers(conversationId, true)
      if (isCurrentConversation) {
        setCurrentMembers(members)
      }
    }
    await loadConversations()
  }, [loadConversationMembers, loadConversations])

  const leaveConversation = useCallback(async (conversationId: number) => {
    await apiLeaveConversation(conversationId)
    setConversations((previous) => previous.filter((item) => item.id !== conversationId))
    setCurrentConversation((current) => (current?.id === conversationId ? null : current))
    setCurrentMessages((previous) => (currentConversationIdRef.current === conversationId ? [] : previous))
    setCurrentMembers((previous) => (currentConversationIdRef.current === conversationId ? [] : previous))
    delete membersCacheRef.current[conversationId]
  }, [])

  const loadFriends = useCallback(async () => {
    if (!isAuthenticated) {
      return
    }
    setFriends(await getFriends())
  }, [isAuthenticated])

  const loadFriendRequests = useCallback(async () => {
    if (!isAuthenticated) {
      return
    }
    setFriendRequests(await getFriendRequests())
  }, [isAuthenticated])

  const loadAllUsers = useCallback(async () => {
    if (!isAuthenticated) {
      return
    }
    const users = await getUsers()
    setAllUsers(users.filter((item) => item.id !== user?.id))
  }, [isAuthenticated, user?.id])

  const approveFriendRequest = useCallback(async (id: number) => {
    await apiApproveFriendRequest(id)
    await Promise.all([loadFriendRequests(), loadFriends()])
  }, [loadFriendRequests, loadFriends])

  const rejectFriendRequest = useCallback(async (id: number) => {
    await apiRejectFriendRequest(id)
    await loadFriendRequests()
  }, [loadFriendRequests])

  const sendFriendRequest = useCallback(async (userId: number, message?: string) => {
    await apiCreateFriendRequest({ addressee_id: userId, message: message || '' })
    await loadFriendRequests()
  }, [loadFriendRequests])

  const updateFriendRemark = useCallback(async (friendId: number, remarkName: string) => {
    await apiUpdateFriendRemark(friendId, { remark_name: remarkName })
    await Promise.all([loadFriends(), loadConversations()])
  }, [loadConversations, loadFriends])

  const startDirectChat = useCallback(async (userId: number) => {
    const conversation = await createDirectConversation({ target_user_id: userId })
    const members = await loadConversationMembers(conversation.id, true)
    const nextConversation = resolveDirectName(conversation, user?.id, members)
    updateConversationState((rows) => [
      ...rows.filter((row) => row.id !== nextConversation.id),
      nextConversation,
    ])
    setCurrentConversation(nextConversation)
    setCurrentMembers(members)
    setActiveView('chat')
  }, [loadConversationMembers, updateConversationState, user?.id])

  const createGroupChat = useCallback(async (name: string, memberIds: number[]) => {
    const conversation = await apiCreateGroupConversation({ name, member_ids: memberIds })
    const members = await loadConversationMembers(conversation.id, true)
    updateConversationState((rows) => [
      ...rows.filter((row) => row.id !== conversation.id),
      conversation,
    ])
    setCurrentConversation(conversation)
    setCurrentMembers(members)
    setActiveView('chat')
  }, [loadConversationMembers, updateConversationState])

  const applyIncomingEvent = useCallback((event: { type: string; payload: any }) => {
    const currentUserId = currentUserIdRef.current
    switch (event.type) {
      case 'message.accepted':
        if (currentConversationIdRef.current === event.payload.conversation_id) {
          setCurrentMessages((previous) =>
            previous.map((item) =>
              item.client_msg_id === event.payload.client_msg_id && item.delivery_status === 'sending'
                ? { ...item, status: 'sent', delivery_status: 'sent' }
                : item,
            ),
          )
        }
        break
      case 'message.persisted':
      case 'message.new': {
        const cachedMembers = getCachedMembers(event.payload.conversation.id)
        const nextConversation = resolveDirectName(
          mergeConversation(
            conversationsRef.current.find((item) => item.id === event.payload.conversation.id),
            event.payload.conversation,
          ),
          currentUserId,
          cachedMembers,
        )

        updateConversationState((rows) => [
          ...rows.filter((item) => item.id !== nextConversation.id),
          nextConversation,
        ])

        if (currentConversationIdRef.current === event.payload.conversation.id) {
          setCurrentMessages((previous) => {
            if (previous.some((item) => item.id === event.payload.message.id)) {
              return previous
            }
            return replaceMessage(previous, event.payload.message)
          })
          if (event.payload.message.sender_id !== currentUserId) {
            void markAsRead(event.payload.conversation.id, event.payload.message.seq)
          }
        }
        break
      }
      case 'message.delivered':
        if (currentConversationIdRef.current === event.payload.conversation_id) {
          setCurrentMessages((previous) =>
            previous.map((item) =>
              item.id === event.payload.message_id || item.client_msg_id === event.payload.client_msg_id
                ? {
                    ...item,
                    status: event.payload.delivery_status,
                    delivery_status: event.payload.delivery_status,
                  }
                : item,
            ),
          )
        }
        break
      case 'message.read':
      case 'conversation.read':
        if (currentConversationIdRef.current === event.payload.conversation_id) {
          setCurrentMembers((previous) =>
            previous.map((member) =>
              member.user_id === event.payload.reader_id
                ? { ...member, last_read_seq: event.payload.last_read_seq }
                : member,
            ),
          )
          if (event.payload.reader_id !== currentUserId) {
            setCurrentMessages((previous) =>
              previous.map((item) =>
                item.sender_id === currentUserId && item.seq <= event.payload.last_read_seq
                  ? { ...item, status: 'read', delivery_status: 'read' }
                  : item,
              ),
            )
          }
        }
        break
      case 'typing.updated':
        if (currentConversationIdRef.current === event.payload.conversation_id) {
          setTypingUserIds((previous) => {
            const exists = previous.includes(event.payload.user_id)
            if (event.payload.is_typing) {
              return exists ? previous : [...previous, event.payload.user_id]
            }
            return previous.filter((item) => item !== event.payload.user_id)
          })
        }
        break
      case 'message.recalled':
        updateConversationState((rows) =>
          rows.map((row) =>
            row.id === event.payload.conversation_id
              ? { ...row, last_message_preview: '[消息已撤回]' }
              : row,
          ),
        )
        if (currentConversationIdRef.current === event.payload.conversation_id) {
          setCurrentMessages((previous) =>
            previous.map((item) =>
              item.id === event.payload.message_id
                ? {
                    ...item,
                    recalled_at: event.payload.recalled_at,
                    status: item.status === 'failed' ? item.status : 'read',
                    delivery_status: item.delivery_status === 'failed' ? item.delivery_status : 'read',
                  }
                : item,
            ),
          )
        }
        break
      case 'conversation.updated':
        updateConversationState((rows) => [
          ...rows.filter((item) => item.id !== event.payload.conversation.id),
          event.payload.conversation,
        ])
        break
      case 'conversation.removed':
        setConversations((previous) => previous.filter((item) => item.id !== event.payload.conversation_id))
        setCurrentConversation((current) => (current?.id === event.payload.conversation_id ? null : current))
        if (currentConversationIdRef.current === event.payload.conversation_id) {
          setCurrentMessages([])
          setCurrentMembers([])
        }
        delete membersCacheRef.current[event.payload.conversation_id]
        break
      case 'member.joined':
      case 'member.left':
        if (currentConversationIdRef.current === event.payload.conversation_id) {
          void loadConversationMembers(event.payload.conversation_id, true).then((members) => {
            if (currentConversationIdRef.current === event.payload.conversation_id) {
              setCurrentMembers(members)
            }
          })
        }
        void loadConversations()
        break
      case 'friend.request':
        setFriendRequests((previous) => {
          const exists = previous.some((item) => item.id === event.payload.request.id)
          if (exists) {
            return previous.map((item) =>
              item.id === event.payload.request.id ? event.payload.request : item,
            )
          }
          return [event.payload.request, ...previous]
        })
        break
      default:
        break
    }
  }, [getCachedMembers, loadConversationMembers, loadConversations, markAsRead, updateConversationState])

  const replaySyncEvents = useCallback(async () => {
    if (!isAuthenticated || !syncReadyRef.current) {
      return
    }
    if (syncReplayInFlightRef.current) {
      syncReplayPendingRef.current = true
      return
    }
    syncReplayInFlightRef.current = true
    try {
      let cursor = syncCursorRef.current
      while (true) {
        const result = await getSyncEvents(cursor, 50)
        for (const event of result.events) {
          applyIncomingEvent({
            type: event.event_type || event.type,
            payload: event.payload,
          })
          cursor = event.cursor
        }
        syncCursorRef.current = result.next_cursor || result.current_cursor || cursor
        if (!result.has_more || result.events.length === 0) {
          break
        }
        cursor = result.next_cursor
      }
    } finally {
      syncReplayInFlightRef.current = false
      if (syncReplayPendingRef.current) {
        syncReplayPendingRef.current = false
        void replaySyncEvents()
      }
    }
  }, [applyIncomingEvent, isAuthenticated])

  useEffect(() => {
    if (!isAuthenticated) {
      setConversations([])
      setCurrentConversation(null)
      setCurrentMessages([])
      setCurrentMembers([])
      setFriends([])
      setFriendRequests([])
      setGroupRequests([])
      setAllUsers([])
      syncCursorRef.current = 0
      syncReadyRef.current = false
      syncReplayInFlightRef.current = false
      syncReplayPendingRef.current = false
      membersCacheRef.current = {}
      return
    }

    void (async () => {
      await Promise.all([
        loadConversations(),
        loadFriends(),
        loadFriendRequests(),
      ])
      const cursor = await getSyncCursor()
      syncCursorRef.current = cursor.cursor
      syncReadyRef.current = true
    })()
  }, [isAuthenticated, loadConversations, loadFriendRequests, loadFriends])

  useEffect(() => {
    if (!currentConversation?.id) {
      setCurrentMessages([])
      setCurrentMembers([])
      return
    }
    const conversationId = currentConversation.id

    let cancelled = false

    async function hydrateConversation() {
      const messages = await loadMessages(conversationId)
      if (cancelled) {
        return
      }
      const lastSeq = messages.at(-1)?.seq
      if (lastSeq) {
        await markAsRead(conversationId, lastSeq)
      }
    }

    void hydrateConversation()

    return () => {
      cancelled = true
    }
  }, [currentConversation?.id, loadMessages, markAsRead])

  useEffect(() => {
    if (!isAuthenticated) {
      return
    }

    return subscribe((event) => {
      if (event.type === 'sync.notify') {
        void replaySyncEvents()
        return
      }
      applyIncomingEvent(event)
    })
  }, [
    applyIncomingEvent,
    isAuthenticated,
    replaySyncEvents,
    subscribe,
  ])

  useEffect(() => {
    if (!isConnected || !syncReadyRef.current) {
      return
    }
    void replaySyncEvents()
  }, [isConnected, replaySyncEvents])

  return (
    <ChatStoreContext.Provider
      value={{
        activeView,
        setActiveView,
        conversations,
        currentConversation,
        currentMessages,
        currentMembers,
        setCurrentConversation,
        loadConversations,
        loadMessages,
        sendMessage,
        retryMessage,
        recallMessage,
        sendTyping,
        typingUserIds,
        markAsRead,
        updateConversationSettings,
        renameConversation,
        addConversationMembers,
        removeConversationMember,
        leaveConversation,
        friends,
        friendRequests,
        groupRequests,
        allUsers,
        loadFriends,
        loadFriendRequests,
        loadAllUsers,
        approveFriendRequest,
        rejectFriendRequest,
        sendFriendRequest,
        updateFriendRemark,
        startDirectChat,
        createGroupChat,
        searchQuery,
        setSearchQuery,
        isLoading,
        isConnected,
      }}
    >
      {children}
    </ChatStoreContext.Provider>
  )
}

export function useChatStore() {
  const context = useContext(ChatStoreContext)
  if (!context) {
    throw new Error('useChatStore must be used within a ChatStoreProvider')
  }
  return context
}
