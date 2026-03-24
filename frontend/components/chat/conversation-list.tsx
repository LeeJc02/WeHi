'use client'

import { useMemo } from 'react'
import { cn } from '@/lib/utils'
import { useChatStore } from '@/lib/chat-store'
import { useAuth } from '@/lib/auth-context'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Search, Pin, BellOff, Users } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import type { Conversation } from '@/lib/types'

export function ConversationList() {
  const { user } = useAuth()
  const {
    conversations,
    currentConversation,
    setCurrentConversation,
    searchQuery,
    setSearchQuery,
    currentMembers,
  } = useChatStore()

  // 搜索过滤
  const filteredConversations = useMemo(() => {
    if (!searchQuery.trim()) return conversations
    const query = searchQuery.toLowerCase()
    return conversations.filter((c) =>
      c.name?.toLowerCase().includes(query) ||
      c.last_message_preview?.toLowerCase().includes(query)
    )
  }, [conversations, searchQuery])

  // 格式化时间
  function formatTime(dateStr: string) {
    if (!dateStr) return ''
    try {
      const date = new Date(dateStr)
      const now = new Date()
      const diffDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24))
      
      if (diffDays === 0) {
        return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
      } else if (diffDays === 1) {
        return '昨天'
      } else if (diffDays < 7) {
        return formatDistanceToNow(date, { addSuffix: true, locale: zhCN })
      } else {
        return date.toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' })
      }
    } catch {
      return ''
    }
  }

  // 获取会话显示名称
  function getConversationName(conversation: typeof conversations[0]) {
    if (conversation.name) return conversation.name
    if (conversation.type === 'direct') {
      // 对于私聊，显示对方名称
      const otherMember = currentMembers.find((m) => m.user_id !== user?.id)
      return otherMember?.display_name || '私聊'
    }
    return '群聊'
  }

  // 获取头像显示
  function getAvatarText(conversation: typeof conversations[0]) {
    const name = getConversationName(conversation)
    return name.charAt(0)
  }

  return (
    <div className="flex h-full flex-col">
      {/* 头部搜索 */}
      <div className="border-b border-panel-border/70 px-4 pb-4 pt-5">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <p className="text-xs font-medium uppercase tracking-[0.22em] text-muted-foreground/70">会话</p>
            <h2 className="mt-1 text-lg font-semibold tracking-[-0.03em] text-foreground">最近聊天</h2>
          </div>
          <div className="rounded-full bg-wechat-green/8 px-3 py-1 text-xs font-medium text-wechat-green">
            {filteredConversations.length} 个会话
          </div>
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="搜索"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-11 rounded-2xl border-black/5 bg-white/90 pl-10 shadow-sm focus-visible:ring-4"
          />
        </div>
      </div>

      {/* 会话列表 */}
      <ScrollArea className="flex-1">
        <div className="space-y-1 px-2 py-2">
          {filteredConversations.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground text-sm">
              {searchQuery ? '没有找到匹配的会话' : '暂无会话'}
            </div>
          ) : (
            filteredConversations.map((conversation) => (
              <ConversationItem
                key={conversation.id}
                conversation={conversation}
                isActive={currentConversation?.id === conversation.id}
                onClick={() => setCurrentConversation(conversation)}
                formatTime={formatTime}
                getAvatarText={getAvatarText}
                getConversationName={getConversationName}
              />
            ))
          )}
        </div>
      </ScrollArea>
    </div>
  )
}

interface ConversationItemProps {
  conversation: Conversation
  isActive: boolean
  onClick: () => void
  formatTime: (dateStr: string) => string
  getAvatarText: (conversation: ConversationItemProps['conversation']) => string
  getConversationName: (conversation: ConversationItemProps['conversation']) => string
}

function ConversationItem({
  conversation,
  isActive,
  onClick,
  formatTime,
  getAvatarText,
  getConversationName,
}: ConversationItemProps) {
  return (
    <button
      onClick={onClick}
      className={cn(
        'w-full rounded-[22px] border border-transparent px-3 py-3 text-left transition-all duration-200',
        'hover:border-black/5 hover:bg-white/88 hover:shadow-[0_8px_18px_rgba(15,23,42,0.06)]',
        isActive && 'border-wechat-green/14 bg-[linear-gradient(180deg,rgba(34,197,94,0.1),rgba(255,255,255,0.94))] shadow-[0_12px_26px_rgba(34,197,94,0.08)]',
        conversation.pinned && !isActive && 'bg-white/60'
      )}
    >
      {/* 头像 */}
      <Avatar className="h-12 w-12 rounded-[18px] flex-shrink-0 shadow-sm">
        <AvatarFallback
          className={cn(
            'rounded-[18px] text-white font-medium',
            conversation.type === 'group'
              ? 'bg-[linear-gradient(145deg,#60a5fa,#2563eb)]'
              : 'bg-[linear-gradient(145deg,var(--color-wechat-green),#0f9d58)]'
          )}
        >
          {conversation.type === 'group' ? (
            <Users className="h-5 w-5" />
          ) : (
            getAvatarText(conversation)
          )}
        </AvatarFallback>
      </Avatar>

      {/* 内容 */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center justify-between gap-2">
          <span className="font-medium text-sm truncate text-foreground">
            {getConversationName(conversation)}
          </span>
          <span className="text-xs text-muted-foreground flex-shrink-0">
            {formatTime(conversation.last_message_at)}
          </span>
        </div>
        <div className="mt-1 flex items-center justify-between gap-2">
          <span className={cn('truncate text-sm', conversation.draft ? 'font-medium text-amber-700' : 'text-muted-foreground')}>
            {conversation.draft ? `草稿：${conversation.draft}` : conversation.last_message_preview || '暂无消息'}
          </span>
          <div className="flex items-center gap-1 flex-shrink-0">
            {conversation.pinned && (
              <Pin className="h-3 w-3 text-muted-foreground" />
            )}
            {conversation.is_muted && (
              <BellOff className="h-3 w-3 text-muted-foreground" />
            )}
            {conversation.unread_count > 0 && (
              <span className="min-w-[18px] h-[18px] rounded-full bg-red-500 text-white text-xs flex items-center justify-center px-1">
                {conversation.unread_count > 99 ? '99+' : conversation.unread_count}
              </span>
            )}
          </div>
        </div>
      </div>
    </button>
  )
}
