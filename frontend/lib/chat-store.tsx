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
  createDirectConversation,
  createGroupConversation as apiCreateGroupConversation,
  createFriendRequest as apiCreateFriendRequest,
  getConversationMembers,
  getConversations,
  getFriendRequests,
  getFriends,
  getMessages,
  getUsers,
  markAsRead as apiMarkAsRead,
  rejectFriendRequest as apiRejectFriendRequest,
  sendMessage as apiSendMessage,
} from './api'
import type {
  Conversation,
  ConversationMember,
  Friend,
  FriendRequest,
  GroupRequest,
  Message,
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
  sendMessage: (content: string) => Promise<void>
  markAsRead: (conversationId: number, seq?: number) => Promise<void>
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
    name: other.display_name || other.username || '私聊',
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

export function ChatStoreProvider({ children }: { children: ReactNode }) {
  const { user, isAuthenticated } = useAuth()
  const { isConnected, subscribe } = useWebSocket(isAuthenticated)
  const membersCacheRef = useRef<Record<number, ConversationMember[]>>({})
  const conversationsRef = useRef<Conversation[]>([])
  const currentConversationIdRef = useRef<number | null>(null)
  const currentUserIdRef = useRef<number | undefined>(undefined)
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

  useEffect(() => {
    conversationsRef.current = conversations
  }, [conversations])

  useEffect(() => {
    currentConversationIdRef.current = currentConversation?.id ?? null
  }, [currentConversation?.id])

  useEffect(() => {
    currentUserIdRef.current = user?.id
  }, [user?.id])

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

  const sendMessage = useCallback(async (content: string) => {
    if (!currentConversation) {
      return
    }
    const message = await apiSendMessage(currentConversation.id, {
      content,
      message_type: 'text',
      client_msg_id: `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`,
    })
    setCurrentMessages((previous) => {
      if (previous.some((item) => item.id === message.id)) {
        return previous
      }
      return sortMessages([...previous, message])
    })
    updateConversationState((rows) => {
      const nextConversation = rows.find((row) => row.id === currentConversation.id)
      const merged = {
        ...(nextConversation ?? currentConversation),
        last_message_at: message.created_at,
        last_message_preview: message.content,
        last_message_sender: message.sender_id,
        last_message_seq: message.seq,
        last_message_type: message.message_type,
      }
      return [
        ...rows.filter((row) => row.id !== currentConversation.id),
        merged,
      ]
    })
  }, [currentConversation, updateConversationState])

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
      membersCacheRef.current = {}
      return
    }

    void Promise.all([
      loadConversations(),
      loadFriends(),
      loadFriendRequests(),
    ])
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
      const currentUserId = currentUserIdRef.current
      switch (event.type) {
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
              return sortMessages([...previous, event.payload.message])
            })
            if (event.payload.message.sender_id !== currentUserId) {
              void markAsRead(event.payload.conversation.id, event.payload.message.seq)
            }
          }
          break
        }
        case 'conversation.read':
          if (currentConversationIdRef.current === event.payload.conversation_id) {
            setCurrentMembers((previous) =>
              previous.map((member) =>
                member.user_id === event.payload.reader_id
                  ? { ...member, last_read_seq: event.payload.last_read_seq }
                  : member,
              ),
            )
          }
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
        case 'sync.notify':
          void Promise.all([loadConversations(), loadFriends(), loadFriendRequests()])
          break
        default:
          break
      }
    })
  }, [
    getCachedMembers,
    isAuthenticated,
    loadConversations,
    loadFriendRequests,
    loadFriends,
    markAsRead,
    subscribe,
    updateConversationState,
  ])

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
        markAsRead,
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
