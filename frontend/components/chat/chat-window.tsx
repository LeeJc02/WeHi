'use client'

import { useState, useRef, useEffect, useCallback } from 'react'
import { completeUpload, presignUpload, uploadObject } from '@/lib/api'
import { cn } from '@/lib/utils'
import { useChatStore } from '@/lib/chat-store'
import { useAuth } from '@/lib/auth-context'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
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
  Reply,
  Video,
  Users,
  Pin,
  Bell,
  Crown,
  Shield,
  Trash2,
  UserMinus,
  UserPlus,
} from 'lucide-react'
import type { ConversationMember, Friend, Message } from '@/lib/types'

export function ChatWindow() {
  const { user } = useAuth()
  const {
    currentConversation,
    currentMessages,
    currentMembers,
    typingUserIds,
    recallMessage,
    retryMessage,
    friends,
    sendMessage,
    sendTyping,
    isLoading,
    updateConversationSettings,
    renameConversation,
    addConversationMembers,
    removeConversationMember,
    leaveConversation,
  } = useChatStore()

  const [inputValue, setInputValue] = useState('')
  const [isSending, setIsSending] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [replyTarget, setReplyTarget] = useState<Message | null>(null)
  const [showGroupDetails, setShowGroupDetails] = useState(false)
  const [groupNameInput, setGroupNameInput] = useState('')
  const [selectedMemberIds, setSelectedMemberIds] = useState<number[]>([])
  const [isSavingGroup, setIsSavingGroup] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const imageInputRef = useRef<HTMLInputElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const typingStateRef = useRef(false)
  const skipNextTypingSyncRef = useRef(false)

  // 滚动到底部
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  // 当消息更新时滚动到底部
  useEffect(() => {
    scrollToBottom()
  }, [currentMessages, scrollToBottom])

  useEffect(() => {
    setInputValue(currentConversation?.draft || '')
    setReplyTarget(null)
    skipNextTypingSyncRef.current = true
  }, [currentConversation?.id, currentConversation?.draft])

  useEffect(() => {
    setGroupNameInput(currentConversation?.name || '')
    setSelectedMemberIds([])
  }, [currentConversation?.id, currentConversation?.name])

  useEffect(() => {
    if (!currentConversation) {
      return
    }
    const normalizedInput = inputValue.trim()
    const normalizedDraft = (currentConversation.draft || '').trim()
    if (normalizedInput === normalizedDraft) {
      return
    }
    const timeoutId = window.setTimeout(() => {
      void updateConversationSettings(currentConversation.id, { draft: inputValue })
    }, 400)
    return () => window.clearTimeout(timeoutId)
  }, [currentConversation, inputValue, updateConversationSettings])

  useEffect(() => {
    if (!currentConversation) {
      typingStateRef.current = false
      return
    }
    if (skipNextTypingSyncRef.current) {
      skipNextTypingSyncRef.current = false
      return
    }
    const hasContent = inputValue.trim().length > 0
    let timeoutId: number | undefined

    if (hasContent && !typingStateRef.current) {
      typingStateRef.current = true
      void sendTyping(currentConversation.id, true)
    }

    if (typingStateRef.current) {
      timeoutId = window.setTimeout(() => {
        typingStateRef.current = false
        void sendTyping(currentConversation.id, false)
      }, hasContent ? 1500 : 0)
    }

    return () => {
      if (timeoutId) {
        window.clearTimeout(timeoutId)
      }
    }
  }, [currentConversation, inputValue, sendTyping])

  useEffect(() => {
    return () => {
      if (currentConversation && typingStateRef.current) {
        void sendTyping(currentConversation.id, false)
      }
    }
  }, [currentConversation, sendTyping])

  // 发送消息
  async function handleSend() {
    if (!inputValue.trim() || isSending) return
    if (!currentConversation) return

    setIsSending(true)
    const conversationId = currentConversation.id
    try {
      await sendMessage({
        content: inputValue.trim(),
        messageType: 'text',
        replyToMessageId: replyTarget?.id,
      })
      if (typingStateRef.current) {
        typingStateRef.current = false
        void sendTyping(conversationId, false)
      }
      await updateConversationSettings(conversationId, { draft: '' })
      setInputValue('')
      setReplyTarget(null)
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
      return otherMember?.remark_name || otherMember?.display_name || '私聊'
    }
    return '群聊'
  }

  function getMessagePreview(message: Pick<Message, 'message_type' | 'content' | 'attachment' | 'recalled_at'>) {
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

  async function handleTogglePinned() {
    if (!currentConversation) {
      return
    }
    await updateConversationSettings(currentConversation.id, { pinned: !currentConversation.pinned })
  }

  async function handleToggleMuted() {
    if (!currentConversation) {
      return
    }
    await updateConversationSettings(currentConversation.id, { is_muted: !currentConversation.is_muted })
  }

  async function handleEditAnnouncement() {
    if (!currentConversation || currentConversation.type !== 'group') {
      return
    }
    const nextAnnouncement = window.prompt('编辑群公告', currentConversation.announcement || '')
    if (nextAnnouncement === null) {
      return
    }
    await updateConversationSettings(currentConversation.id, { announcement: nextAnnouncement })
  }

  const currentMember = currentMembers.find((member) => member.user_id === user?.id)
  const isGroupManager = currentMember?.role === 'owner' || currentMember?.role === 'admin'
  const canManageGroupName = currentConversation?.type === 'group' && isGroupManager
  const selectableFriends = friends.filter((friend) => !currentMembers.some((member) => member.user_id === friend.id))

  async function handleSaveGroupDetails() {
    if (!currentConversation || currentConversation.type !== 'group') {
      return
    }
    setIsSavingGroup(true)
    try {
      const normalizedName = groupNameInput.trim()
      if (canManageGroupName && normalizedName && normalizedName !== currentConversation.name) {
        await renameConversation(currentConversation.id, normalizedName)
      }
      if (selectedMemberIds.length > 0) {
        await addConversationMembers(currentConversation.id, selectedMemberIds)
        setSelectedMemberIds([])
      }
      setShowGroupDetails(false)
    } finally {
      setIsSavingGroup(false)
    }
  }

  async function handleRemoveMember(member: ConversationMember) {
    if (!currentConversation || currentConversation.type !== 'group') {
      return
    }
    await removeConversationMember(currentConversation.id, member.user_id)
  }

  async function handleLeaveConversation() {
    if (!currentConversation || currentConversation.type !== 'group') {
      return
    }
    await leaveConversation(currentConversation.id)
    setShowGroupDetails(false)
  }

  function toggleSelectedMember(friend: Friend) {
    setSelectedMemberIds((previous) =>
      previous.includes(friend.id)
        ? previous.filter((id) => id !== friend.id)
        : [...previous, friend.id],
    )
  }

  // 获取发送者信息
  function getSenderInfo(senderId: number) {
    const member = currentMembers.find((m) => m.user_id === senderId)
    return {
      name: member?.remark_name || member?.display_name || member?.username || '未知用户',
      initial: member?.remark_name?.charAt(0) || member?.display_name?.charAt(0) || member?.username?.charAt(0) || '?',
    }
  }

  async function handleUpload(file: File, messageType: 'image' | 'file') {
    if (!currentConversation) {
      return
    }
    setIsUploading(true)
    try {
      const presigned = await presignUpload({
        filename: file.name,
        content_type: file.type,
        size_bytes: file.size,
      })
      await uploadObject(presigned.upload_path, file, presigned.headers)
      const completed = await completeUpload({
        object_key: presigned.object_key,
        filename: file.name,
        content_type: file.type,
        size_bytes: file.size,
      })
      await sendMessage({
        content: file.name,
        messageType,
        attachment: completed.attachment,
        replyToMessageId: replyTarget?.id,
      })
      await updateConversationSettings(currentConversation.id, { draft: '' })
      setReplyTarget(null)
    } finally {
      setIsUploading(false)
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
  const typingNames = currentMembers
    .filter((member) => typingUserIds.includes(member.user_id) && member.user_id !== user?.id)
    .map((member) => member.remark_name || member.display_name || member.username)

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
                <DropdownMenuItem onClick={() => setShowGroupDetails(true)}>
                  <Users className="mr-2 h-4 w-4" />
                  群聊信息
                </DropdownMenuItem>
              )}
              {currentConversation.type === 'group' && (
                <DropdownMenuItem onClick={() => void handleEditAnnouncement()}>
                  <Bell className="mr-2 h-4 w-4" />
                  编辑群公告
                </DropdownMenuItem>
              )}
              <DropdownMenuItem onClick={() => void handleTogglePinned()}>
                <Pin className="mr-2 h-4 w-4" />
                {currentConversation.pinned ? '取消置顶' : '置顶聊天'}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => void handleToggleMuted()}>
                <Bell className="mr-2 h-4 w-4" />
                {currentConversation.is_muted ? '关闭免打扰' : '消息免打扰'}
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

      {currentConversation.announcement && (
        <div className="border-b border-white/10 bg-amber-500/10 px-4 py-2 text-sm text-amber-900">
          群公告：{currentConversation.announcement}
        </div>
      )}

      {typingNames.length > 0 && (
        <div className="border-b border-white/10 bg-sky-500/10 px-4 py-2 text-sm text-sky-900">
          {typingNames.join('、')} 正在输入...
        </div>
      )}

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
                      onReply={() => setReplyTarget(message)}
                      onRecall={isSelf && !message.recalled_at ? () => void recallMessage(message.id) : undefined}
                      onRetry={isSelf && message.status === 'failed' ? () => void retryMessage(message.id) : undefined}
                      getMessagePreview={getMessagePreview}
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
                <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => imageInputRef.current?.click()}>
                  <ImageIcon className="h-5 w-5 text-muted-foreground" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>图片</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8" onClick={() => fileInputRef.current?.click()}>
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

        <input
          ref={imageInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={(event) => {
            const file = event.target.files?.[0]
            if (file) {
              void handleUpload(file, 'image')
            }
            event.target.value = ''
          }}
        />
        <input
          ref={fileInputRef}
          type="file"
          className="hidden"
          onChange={(event) => {
            const file = event.target.files?.[0]
            if (file) {
              void handleUpload(file, 'file')
            }
            event.target.value = ''
          }}
        />

        {replyTarget && (
          <div className="mb-2 flex items-center justify-between rounded-lg border bg-muted/30 px-3 py-2 text-sm">
            <div className="min-w-0">
              <p className="font-medium">回复消息</p>
              <p className="truncate text-muted-foreground">{getMessagePreview(replyTarget)}</p>
            </div>
            <Button variant="ghost" size="sm" onClick={() => setReplyTarget(null)}>
              取消
            </Button>
          </div>
        )}

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
            disabled={!inputValue.trim() || isSending || isUploading}
            className="h-9 px-4 bg-wechat-green hover:bg-wechat-green-dark"
          >
            <Send className="h-4 w-4 mr-1" />
            {isUploading ? '上传中...' : '发送'}
          </Button>
        </div>
      </div>

      <Dialog open={showGroupDetails} onOpenChange={setShowGroupDetails}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>群聊信息</DialogTitle>
            <DialogDescription>查看成员、修改群信息并管理群成员。</DialogDescription>
          </DialogHeader>

          <div className="space-y-5">
            <div className="space-y-2">
              <label className="text-sm font-medium">群名称</label>
              <Input
                value={groupNameInput}
                onChange={(event) => setGroupNameInput(event.target.value)}
                disabled={!canManageGroupName}
                placeholder="输入群名称"
              />
            </div>

            <div className="space-y-2">
              <div className="flex items-center justify-between gap-2">
                <label className="text-sm font-medium">群公告</label>
                {isGroupManager && (
                  <Button variant="ghost" size="sm" onClick={() => void handleEditAnnouncement()}>
                    编辑公告
                  </Button>
                )}
              </div>
              <div className="rounded-lg border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
                {currentConversation.announcement || '暂无群公告'}
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">群成员</label>
              <ScrollArea className="h-56 rounded-lg border">
                <div className="space-y-2 p-3">
                  {currentMembers.map((member) => (
                    <div key={member.user_id} className="flex items-center gap-3 rounded-lg border bg-white/60 px-3 py-2">
                      <Avatar className="h-9 w-9 rounded-lg">
                        <AvatarFallback className="rounded-lg bg-blue-500 text-white">
                          {(member.display_name || member.username).charAt(0)}
                        </AvatarFallback>
                      </Avatar>
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">{member.display_name}</p>
                        <p className="truncate text-xs text-muted-foreground">@{member.username}</p>
                      </div>
                      <MemberRoleBadge role={member.role} />
                      {isGroupManager && member.user_id !== user?.id && (
                        <Button variant="ghost" size="icon" onClick={() => void handleRemoveMember(member)}>
                          <UserMinus className="h-4 w-4 text-destructive" />
                        </Button>
                      )}
                    </div>
                  ))}
                </div>
              </ScrollArea>
            </div>

            {isGroupManager && selectableFriends.length > 0 && (
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <UserPlus className="h-4 w-4" />
                  <label className="text-sm font-medium">添加成员</label>
                </div>
                <ScrollArea className="h-40 rounded-lg border">
                  <div className="space-y-2 p-3">
                    {selectableFriends.map((friend) => {
                      const selected = selectedMemberIds.includes(friend.id)
                      return (
                        <button
                          key={friend.id}
                          onClick={() => toggleSelectedMember(friend)}
                          className={cn(
                            'flex w-full items-center gap-3 rounded-lg border px-3 py-2 text-left transition-colors',
                            selected ? 'border-wechat-green bg-green-50' : 'bg-white/60 hover:bg-accent',
                          )}
                        >
                          <Avatar className="h-9 w-9 rounded-lg">
                            <AvatarFallback className="rounded-lg bg-wechat-green text-white">
                              {friend.display_name.charAt(0)}
                            </AvatarFallback>
                          </Avatar>
                          <div className="min-w-0 flex-1">
                            <p className="truncate text-sm font-medium">{friend.display_name}</p>
                            <p className="truncate text-xs text-muted-foreground">@{friend.username}</p>
                          </div>
                          {selected && <Badge>已选择</Badge>}
                        </button>
                      )
                    })}
                  </div>
                </ScrollArea>
              </div>
            )}
          </div>

          <DialogFooter className="flex gap-2 sm:justify-between">
            <Button variant="destructive" onClick={() => void handleLeaveConversation()}>
              退出群聊
            </Button>
            <div className="flex gap-2">
              <Button variant="ghost" onClick={() => setShowGroupDetails(false)}>
                取消
              </Button>
              <Button
                onClick={() => void handleSaveGroupDetails()}
                disabled={isSavingGroup || (!canManageGroupName && selectedMemberIds.length === 0)}
                className="bg-wechat-green hover:bg-wechat-green-dark"
              >
                {isSavingGroup ? '保存中...' : '保存'}
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function MemberRoleBadge({ role }: { role: ConversationMember['role'] }) {
  if (role === 'owner') {
    return (
      <Badge variant="secondary" className="gap-1">
        <Crown className="h-3 w-3" />
        群主
      </Badge>
    )
  }
  if (role === 'admin') {
    return (
      <Badge variant="outline" className="gap-1">
        <Shield className="h-3 w-3" />
        管理员
      </Badge>
    )
  }
  return <Badge variant="outline">成员</Badge>
}

interface MessageBubbleProps {
  message: Message
  isSelf: boolean
  senderName: string
  senderInitial: string
  time: string
  showSender: boolean
  onReply: () => void
  onRecall?: () => void
  onRetry?: () => void
  getMessagePreview: (message: Pick<Message, 'message_type' | 'content' | 'attachment' | 'recalled_at'>) => string
}

function MessageBubble({
  message,
  isSelf,
  senderName,
  senderInitial,
  time,
  showSender,
  onReply,
  onRecall,
  onRetry,
  getMessagePreview,
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
          {message.reply_to && !message.recalled_at && (
            <div className="mb-2 rounded-md border-l-2 border-black/15 bg-black/5 px-2 py-1 text-xs text-muted-foreground">
              {getMessagePreview(message.reply_to)}
            </div>
          )}
          {message.recalled_at ? (
            <p className="text-sm text-muted-foreground italic">消息已撤回</p>
          ) : message.message_type === 'text' ? (
            <p className="text-sm whitespace-pre-wrap break-words leading-relaxed">
              {message.content}
            </p>
          ) : message.message_type === 'image' ? (
            <div className="max-w-[300px]">
              <img
                src={message.attachment?.url || message.content}
                alt="图片消息"
                className="rounded max-w-full"
              />
            </div>
          ) : message.message_type === 'file' ? (
            <a
              href={message.attachment?.url || '#'}
              target="_blank"
              rel="noreferrer"
              className="block rounded-md border bg-background/80 px-3 py-2 text-sm hover:bg-accent"
            >
              {message.attachment?.filename || message.content || '文件'}
            </a>
          ) : (
            <p className="text-sm text-muted-foreground">[不支持的消息类型]</p>
          )}
        </div>

        <div className="mt-1 flex items-center gap-2 px-1 text-xs text-muted-foreground">
          <button onClick={onReply} className="inline-flex items-center gap-1 hover:text-foreground">
            <Reply className="h-3 w-3" />
            回复
          </button>
          {onRecall && (
            <button onClick={onRecall} className="hover:text-destructive">
              撤回
            </button>
          )}
          {onRetry && (
            <button onClick={onRetry} className="hover:text-foreground">
              重发
            </button>
          )}
        </div>

        {/* 时间和状态 */}
        <span className="text-xs text-muted-foreground mt-1 px-1">
          {time}
          {isSelf && message.delivery_status === 'sending' && ' 发送中'}
          {isSelf && message.delivery_status === 'sent' && ' 已发送'}
          {isSelf && message.delivery_status === 'delivered' && ' 已送达'}
          {isSelf && message.delivery_status === 'read' && ' 已读'}
          {isSelf && message.delivery_status === 'failed' && (
            <span className="text-destructive"> 发送失败</span>
          )}
        </span>
      </div>
    </div>
  )
}
