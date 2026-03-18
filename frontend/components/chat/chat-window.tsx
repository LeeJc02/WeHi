'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
import { cn } from '@/lib/utils'
import { useChatStore } from '@/lib/chat-store'
import { useAuth } from '@/lib/auth-context'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Smile,
  Paperclip,
  Image as ImageIcon,
  Folder,
  Send,
  MoreVertical,
  Phone,
  Video,
  Users,
  Pin,
  Bell,
  Trash2,
} from 'lucide-react'
import type { Message } from '@/lib/types'

export function ChatWindow() {
  const { user } = useAuth()
  const {
    currentConversation,
    currentMessages,
    currentMembers,
    sendMessage,
    isLoading,
  } = useChatStore()

  const [inputValue, setInputValue] = useState('')
  const [isSending, setIsSending] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)

  // 滚动到底部
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  // 当消息更新时滚动到底部
  useEffect(() => {
    scrollToBottom()
  }, [currentMessages, scrollToBottom])

  // 发送消息
  async function handleSend() {
    if (!inputValue.trim() || isSending) return

    setIsSending(true)
    try {
      await sendMessage(inputValue.trim())
      setInputValue('')
      textareaRef.current?.focus()
    } catch (error) {
      console.error('Failed to send message:', error)
    } finally {
      setIsSending(false)
    }
  }

  // 键盘事件处理
  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  // 获取会话名称
  function getConversationName() {
    if (!currentConversation) return ''
    if (currentConversation.name) return currentConversation.name
    if (currentConversation.type === 'direct') {
      const otherMember = currentMembers.find((m) => m.user_id !== user?.id)
      return otherMember?.display_name || '私聊'
    }
    return '群聊'
  }

  // 获取发送者信息
  function getSenderInfo(senderId: number) {
    const member = currentMembers.find((m) => m.user_id === senderId)
    return {
      name: member?.display_name || member?.username || '未知用户',
      initial: member?.display_name?.charAt(0) || member?.username?.charAt(0) || '?',
    }
  }

  // 格式化时间
  function formatMessageTime(dateStr: string) {
    try {
      const date = new Date(dateStr)
      return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
      })
    } catch {
      return ''
    }
  }

  // 按日期分组消息
  function groupMessagesByDate(messages: Message[]) {
    const groups: { date: string; messages: Message[] }[] = []
    let currentDate = ''

    messages.forEach((message) => {
      const messageDate = new Date(message.created_at).toLocaleDateString('zh-CN', {
        year: 'numeric',
        month: 'long',
        day: 'numeric',
      })

      if (messageDate !== currentDate) {
        currentDate = messageDate
        groups.push({ date: messageDate, messages: [message] })
      } else {
        groups[groups.length - 1].messages.push(message)
      }
    })

    return groups
  }

  if (!currentConversation) {
    return (
      <div className="h-full flex flex-col items-center justify-center text-muted-foreground">
        <div className="mb-4 flex h-32 w-32 items-center justify-center rounded-full bg-white/16 backdrop-blur-xl">
          <Users className="w-16 h-16 text-muted-foreground/50" />
        </div>
        <p className="text-lg">选择一个会话开始聊天</p>
        <p className="text-sm mt-2">或从通讯录中选择好友</p>
      </div>
    )
  }

  const messageGroups = groupMessagesByDate(currentMessages)

  return (
    <div className="h-full flex flex-col">
      {/* 头部 */}
      <div className="h-14 px-4 flex items-center justify-between border-b border-white/15 bg-white/14 backdrop-blur-xl">
        <div className="flex items-center gap-3">
          <h2 className="font-medium text-foreground">{getConversationName()}</h2>
          {currentConversation.type === 'group' && (
            <span className="text-sm text-muted-foreground">
              ({currentConversation.member_count}人)
            </span>
          )}
        </div>
        <div className="flex items-center gap-1">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Phone className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>语音通话</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Video className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>视频通话</TooltipContent>
            </Tooltip>
          </TooltipProvider>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {currentConversation.type === 'group' && (
                <DropdownMenuItem>
                  <Users className="mr-2 h-4 w-4" />
                  查看群成员
                </DropdownMenuItem>
              )}
              <DropdownMenuItem>
                <Pin className="mr-2 h-4 w-4" />
                {currentConversation.pinned ? '取消置顶' : '置顶聊天'}
              </DropdownMenuItem>
              <DropdownMenuItem>
                <Bell className="mr-2 h-4 w-4" />
                消息免打扰
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem className="text-destructive focus:text-destructive">
                <Trash2 className="mr-2 h-4 w-4" />
                删除聊天记录
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* 消息区域 */}
      <ScrollArea className="flex-1 px-4">
        {isLoading ? (
          <div className="h-full flex items-center justify-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-wechat-green" />
          </div>
        ) : (
          <div className="py-4">
            {messageGroups.map((group) => (
              <div key={group.date}>
                {/* 日期分隔 */}
                <div className="flex items-center justify-center my-4">
                  <span className="rounded-full bg-black/28 px-3 py-1 text-xs text-white/78 backdrop-blur-md">
                    {group.date}
                  </span>
                </div>
                {/* 消息列表 */}
                {group.messages.map((message) => {
                  const isSelf = message.sender_id === user?.id
                  const sender = getSenderInfo(message.sender_id)

                  return (
                    <MessageBubble
                      key={message.id}
                      message={message}
                      isSelf={isSelf}
                      senderName={sender.name}
                      senderInitial={sender.initial}
                      time={formatMessageTime(message.created_at)}
                      showSender={currentConversation.type === 'group' && !isSelf}
                    />
                  )
                })}
              </div>
            ))}
            <div ref={messagesEndRef} />
          </div>
        )}
      </ScrollArea>

      {/* 输入区域 */}
      <div className="border-t border-white/15 bg-white/14 p-3 backdrop-blur-xl">
        {/* 工具栏 */}
        <div className="flex items-center gap-1 mb-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Smile className="h-5 w-5 text-muted-foreground" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>表情</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <ImageIcon className="h-5 w-5 text-muted-foreground" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>图片</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Paperclip className="h-5 w-5 text-muted-foreground" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>文件</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Folder className="h-5 w-5 text-muted-foreground" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>聊天记录</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>

        {/* 输入框 */}
        <div className="flex items-end gap-2">
          <Textarea
            ref={textareaRef}
            placeholder="输入消息..."
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={handleKeyDown}
            className="min-h-[80px] max-h-[200px] resize-none border-0 bg-transparent focus-visible:ring-0 p-0"
            rows={3}
          />
          <Button
            onClick={handleSend}
            disabled={!inputValue.trim() || isSending}
            className="h-9 px-4 bg-wechat-green hover:bg-wechat-green-dark"
          >
            <Send className="h-4 w-4 mr-1" />
            发送
          </Button>
        </div>
      </div>
    </div>
  )
}

interface MessageBubbleProps {
  message: Message
  isSelf: boolean
  senderName: string
  senderInitial: string
  time: string
  showSender: boolean
}

function MessageBubble({
  message,
  isSelf,
  senderName,
  senderInitial,
  time,
  showSender,
}: MessageBubbleProps) {
  return (
    <div
      className={cn(
        'flex gap-2 mb-4',
        isSelf ? 'flex-row-reverse' : 'flex-row'
      )}
    >
      {/* 头像 */}
      <Avatar className="h-9 w-9 rounded-lg flex-shrink-0">
        <AvatarFallback
          className={cn(
            'rounded-lg text-white text-sm',
            isSelf ? 'bg-wechat-green' : 'bg-blue-500'
          )}
        >
          {senderInitial}
        </AvatarFallback>
      </Avatar>

      {/* 消息内容 */}
      <div
        className={cn(
          'flex flex-col max-w-[70%]',
          isSelf ? 'items-end' : 'items-start'
        )}
      >
        {/* 发送者名称（群聊中显示） */}
        {showSender && (
          <span className="text-xs text-muted-foreground mb-1 px-1">
            {senderName}
          </span>
        )}

        {/* 消息气泡 */}
        <div
          className={cn(
            'relative px-3 py-2 rounded-lg',
            isSelf
              ? 'bg-chat-bubble-self text-foreground'
              : 'bg-chat-bubble-other text-foreground shadow-sm',
            // 气泡尖角
            isSelf
              ? 'rounded-tr-sm'
              : 'rounded-tl-sm'
          )}
        >
          {message.message_type === 'text' ? (
            <p className="text-sm whitespace-pre-wrap break-words leading-relaxed">
              {message.content}
            </p>
          ) : message.message_type === 'image' ? (
            <div className="max-w-[300px]">
              <img
                src={message.content}
                alt="图片消息"
                className="rounded max-w-full"
              />
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">[不支持的消息类型]</p>
          )}
        </div>

        {/* 时间和状态 */}
        <span className="text-xs text-muted-foreground mt-1 px-1">
          {time}
          {isSelf && message.status === 'sent' && ' 已发送'}
          {isSelf && message.status === 'delivered' && ' 已送达'}
          {isSelf && message.status === 'read' && ' 已读'}
          {isSelf && message.status === 'failed' && (
            <span className="text-destructive"> 发送失败</span>
          )}
        </span>
      </div>
    </div>
  )
}
