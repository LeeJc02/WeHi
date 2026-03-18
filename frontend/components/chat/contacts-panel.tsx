'use client'

import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { useChatStore } from '@/lib/chat-store'
import { useAuth } from '@/lib/auth-context'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Textarea } from '@/components/ui/textarea'
import { Bell, Check, MessageCircle, Search, UserPlus, Users, X } from 'lucide-react'
import type { Conversation, Friend, FriendRequest, GroupRequest, User } from '@/lib/types'

const dictionarySorter = new Intl.Collator('zh-Hans-CN', {
  numeric: true,
  sensitivity: 'base',
})

type ContactsTab = 'friends' | 'groups' | 'requests'

function sortByDictionaryOrder<T>(items: T[], getPrimary: (item: T) => string, getSecondary?: (item: T) => string) {
  return [...items].sort((left, right) => {
    const primary = dictionarySorter.compare(getPrimary(left), getPrimary(right))
    if (primary !== 0) {
      return primary
    }
    if (!getSecondary) {
      return 0
    }
    return dictionarySorter.compare(getSecondary(left), getSecondary(right))
  })
}

export function ContactsPanel() {
  const { user } = useAuth()
  const {
    friends,
    friendRequests,
    groupRequests,
    conversations,
    allUsers,
    loadAllUsers,
    approveFriendRequest,
    rejectFriendRequest,
    sendFriendRequest,
    startDirectChat,
    createGroupChat,
    setCurrentConversation,
    setActiveView,
  } = useChatStore()

  const [searchQuery, setSearchQuery] = useState('')
  const [activeTab, setActiveTab] = useState<ContactsTab>('friends')
  const [addFriendOpen, setAddFriendOpen] = useState(false)
  const [createGroupOpen, setCreateGroupOpen] = useState(false)

  useEffect(() => {
    loadAllUsers()
  }, [loadAllUsers])

  const filteredFriends = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    const rows = !query
      ? friends
      : friends.filter(
          (friend) =>
            friend.display_name.toLowerCase().includes(query) ||
            friend.username.toLowerCase().includes(query),
        )

    return sortByDictionaryOrder(
      rows,
      (friend) => friend.display_name || friend.username,
      (friend) => friend.username,
    )
  }, [friends, searchQuery])

  const filteredGroups = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    const rows = conversations.filter((conversation) => {
      if (conversation.type !== 'group') {
        return false
      }
      if (!query) {
        return true
      }
      return (
        conversation.name.toLowerCase().includes(query) ||
        conversation.last_message_preview?.toLowerCase().includes(query)
      )
    })

    return sortByDictionaryOrder(
      rows,
      (conversation) => conversation.name || '群聊',
      (conversation) => String(conversation.id),
    )
  }, [conversations, searchQuery])

  const filteredFriendRequests = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    if (!query) {
      return friendRequests
    }
    return friendRequests.filter((request) => {
      const target = request.direction === 'incoming' ? request.requester : request.addressee
      return (
        target.display_name.toLowerCase().includes(query) ||
        target.username.toLowerCase().includes(query) ||
        request.message.toLowerCase().includes(query)
      )
    })
  }, [friendRequests, searchQuery])

  const filteredGroupRequests = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    if (!query) {
      return groupRequests
    }
    return groupRequests.filter((request) => {
      return (
        request.group_name.toLowerCase().includes(query) ||
        request.actor.display_name.toLowerCase().includes(query) ||
        request.actor.username.toLowerCase().includes(query) ||
        request.message.toLowerCase().includes(query)
      )
    })
  }, [groupRequests, searchQuery])

  const pendingRequestsCount =
    friendRequests.filter((request) => request.status === 'pending' && request.direction === 'incoming').length +
    groupRequests.filter((request) => request.status === 'pending' && request.direction === 'incoming').length

  function openConversation(conversation: Conversation) {
    setCurrentConversation(conversation)
    setActiveView('chat')
  }

  const searchPlaceholder =
    activeTab === 'groups'
      ? '搜索群组'
      : activeTab === 'requests'
        ? '搜索新的请求'
        : '搜索联系人'

  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-panel-border/60 p-3">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={searchPlaceholder}
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            className="border-0 bg-white/45 pl-9 focus-visible:ring-1 focus-visible:ring-wechat-green"
          />
        </div>
      </div>

      <div className="border-b border-panel-border/60 px-3 py-3">
        <div className="flex gap-2">
          <Dialog open={addFriendOpen} onOpenChange={setAddFriendOpen}>
            <DialogTrigger asChild>
              <Button variant="ghost" size="sm" className="flex-1 justify-start gap-2 bg-white/30 hover:bg-white/45">
                <UserPlus className="h-4 w-4" />
                添加好友
              </Button>
            </DialogTrigger>
            <AddFriendDialog
              allUsers={allUsers}
              currentUser={user}
              friends={friends}
              friendRequests={friendRequests}
              onSendRequest={sendFriendRequest}
              onClose={() => setAddFriendOpen(false)}
            />
          </Dialog>
          <Dialog open={createGroupOpen} onOpenChange={setCreateGroupOpen}>
            <DialogTrigger asChild>
              <Button variant="ghost" size="sm" className="flex-1 justify-start gap-2 bg-white/30 hover:bg-white/45">
                <Users className="h-4 w-4" />
                拉群组
              </Button>
            </DialogTrigger>
            <CreateGroupDialog
              friends={friends}
              onCreateGroup={createGroupChat}
              onClose={() => setCreateGroupOpen(false)}
            />
          </Dialog>
        </div>
      </div>

      <Tabs
        value={activeTab}
        onValueChange={(value) => setActiveTab(value as ContactsTab)}
        className="flex min-h-0 flex-1 flex-col"
      >
        <div className="border-b border-panel-border/60 px-3 py-2">
          <TabsList className="grid h-10 w-full grid-cols-3 bg-white/24">
            <TabsTrigger value="friends">好友列表</TabsTrigger>
            <TabsTrigger value="groups">群组列表</TabsTrigger>
            <TabsTrigger value="requests" className="gap-2">
              <span>新的请求</span>
              {pendingRequestsCount > 0 && (
                <Badge variant="destructive" className="h-5 min-w-[20px] px-1">
                  {pendingRequestsCount > 99 ? '99+' : pendingRequestsCount}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="friends" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <FriendList friends={filteredFriends} onStartChat={startDirectChat} />
          </ScrollArea>
        </TabsContent>

        <TabsContent value="groups" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <GroupList groups={filteredGroups} onOpenConversation={openConversation} />
          </ScrollArea>
        </TabsContent>

        <TabsContent value="requests" className="mt-0 min-h-0 flex-1">
          <ScrollArea className="h-full">
            <RequestsList
              friendRequests={filteredFriendRequests}
              groupRequests={filteredGroupRequests}
              onApproveFriendRequest={approveFriendRequest}
              onRejectFriendRequest={rejectFriendRequest}
            />
          </ScrollArea>
        </TabsContent>
      </Tabs>
    </div>
  )
}

interface FriendListProps {
  friends: Friend[]
  onStartChat: (userId: number) => Promise<void>
}

function FriendList({ friends, onStartChat }: FriendListProps) {
  if (friends.length === 0) {
    return <EmptyState title="暂无好友" description="先添加好友，再从这里发起聊天。" />
  }

  return (
    <div className="py-2">
      {friends.map((friend) => (
        <button
          key={friend.id}
          onClick={() => onStartChat(friend.id)}
          className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-white/28"
        >
          <Avatar className="h-10 w-10 rounded-lg">
            <AvatarFallback className="rounded-lg bg-wechat-green text-white">
              {friend.display_name.charAt(0)}
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{friend.display_name}</p>
            <p className="truncate text-xs text-muted-foreground">@{friend.username}</p>
          </div>
          <MessageCircle className="h-4 w-4 text-muted-foreground" />
        </button>
      ))}
    </div>
  )
}

interface GroupListProps {
  groups: Conversation[]
  onOpenConversation: (conversation: Conversation) => void
}

function GroupList({ groups, onOpenConversation }: GroupListProps) {
  if (groups.length === 0) {
    return <EmptyState title="暂无群组" description="使用“拉群组”即可从好友中创建新的群聊。" />
  }

  return (
    <div className="py-2">
      {groups.map((group) => (
        <button
          key={group.id}
          onClick={() => onOpenConversation(group)}
          className="flex w-full items-center gap-3 px-4 py-3 text-left transition-colors hover:bg-white/28"
        >
          <Avatar className="h-10 w-10 rounded-lg">
            <AvatarFallback className="rounded-lg bg-blue-500 text-white">
              <Users className="h-5 w-5" />
            </AvatarFallback>
          </Avatar>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium">{group.name || '群聊'}</p>
            <p className="truncate text-xs text-muted-foreground">
              {group.member_count} 人
              {group.last_message_preview ? ` · ${group.last_message_preview}` : ''}
            </p>
          </div>
          {group.unread_count > 0 && (
            <span className="flex h-[18px] min-w-[18px] items-center justify-center rounded-full bg-red-500 px-1 text-xs text-white">
              {group.unread_count > 99 ? '99+' : group.unread_count}
            </span>
          )}
        </button>
      ))}
    </div>
  )
}

interface RequestsListProps {
  friendRequests: FriendRequest[]
  groupRequests: GroupRequest[]
  onApproveFriendRequest: (id: number) => Promise<void>
  onRejectFriendRequest: (id: number) => Promise<void>
}

function RequestsList({
  friendRequests,
  groupRequests,
  onApproveFriendRequest,
  onRejectFriendRequest,
}: RequestsListProps) {
  const [loadingId, setLoadingId] = useState<number | null>(null)

  const incomingFriendRequests = friendRequests.filter(
    (request) => request.status === 'pending' && request.direction === 'incoming',
  )
  const outgoingFriendRequests = friendRequests.filter(
    (request) => request.status === 'pending' && request.direction === 'outgoing',
  )
  const friendHistory = friendRequests.filter((request) => request.status !== 'pending')

  const incomingGroupRequests = groupRequests.filter(
    (request) => request.status === 'pending' && request.direction === 'incoming',
  )
  const outgoingGroupRequests = groupRequests.filter(
    (request) => request.status === 'pending' && request.direction === 'outgoing',
  )
  const groupHistory = groupRequests.filter((request) => request.status !== 'pending')

  async function handleApprove(id: number) {
    setLoadingId(id)
    try {
      await onApproveFriendRequest(id)
    } finally {
      setLoadingId(null)
    }
  }

  async function handleReject(id: number) {
    setLoadingId(id)
    try {
      await onRejectFriendRequest(id)
    } finally {
      setLoadingId(null)
    }
  }

  if (friendRequests.length === 0 && groupRequests.length === 0) {
    return <EmptyState title="暂无新的请求" description="收到好友申请或群组邀请后，会统一显示在这里。" />
  }

  return (
    <div className="space-y-5 px-3 py-3">
      <RequestSection
        title="好友请求"
        icon={<Bell className="h-4 w-4" />}
        count={incomingFriendRequests.length + outgoingFriendRequests.length}
      >
        {incomingFriendRequests.length > 0 && (
          <div className="space-y-2">
            {incomingFriendRequests.map((request) => (
              <RequestCard
                key={request.id}
                name={request.requester.display_name}
                subtitle={request.message || `@${request.requester.username}`}
                initial={request.requester.display_name.charAt(0)}
                actions={
                  <>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                      onClick={() => handleReject(request.id)}
                      disabled={loadingId === request.id}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                    <Button
                      size="sm"
                      className="h-8 w-8 p-0 bg-wechat-green hover:bg-wechat-green-dark"
                      onClick={() => handleApprove(request.id)}
                      disabled={loadingId === request.id}
                    >
                      <Check className="h-4 w-4" />
                    </Button>
                  </>
                }
              />
            ))}
          </div>
        )}

        {outgoingFriendRequests.length > 0 && (
          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">我发出的请求</p>
            {outgoingFriendRequests.map((request) => (
              <RequestCard
                key={request.id}
                name={request.addressee.display_name}
                subtitle="等待对方验证"
                initial={request.addressee.display_name.charAt(0)}
              />
            ))}
          </div>
        )}

        {friendHistory.length > 0 && (
          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">历史记录</p>
            {friendHistory.slice(0, 6).map((request) => {
              const target = request.direction === 'incoming' ? request.requester : request.addressee
              return (
                <RequestCard
                  key={request.id}
                  name={target.display_name}
                  subtitle={request.status === 'accepted' ? '已通过' : '已拒绝'}
                  initial={target.display_name.charAt(0)}
                  muted
                />
              )
            })}
          </div>
        )}

        {incomingFriendRequests.length === 0 &&
          outgoingFriendRequests.length === 0 &&
          friendHistory.length === 0 && (
            <p className="rounded-xl border border-dashed border-white/20 bg-white/10 px-4 py-5 text-sm text-muted-foreground">
              当前没有好友请求。
            </p>
          )}
      </RequestSection>

      <RequestSection
        title="群组请求"
        icon={<Users className="h-4 w-4" />}
        count={incomingGroupRequests.length + outgoingGroupRequests.length}
      >
        {incomingGroupRequests.length > 0 && (
          <div className="space-y-2">
            {incomingGroupRequests.map((request) => (
              <RequestCard
                key={request.id}
                name={request.group_name}
                subtitle={request.message || `${request.actor.display_name} 邀请你加入`}
                initial={request.group_name.charAt(0)}
              />
            ))}
          </div>
        )}

        {outgoingGroupRequests.length > 0 && (
          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">我发出的请求</p>
            {outgoingGroupRequests.map((request) => (
              <RequestCard
                key={request.id}
                name={request.group_name}
                subtitle="等待群主或管理员处理"
                initial={request.group_name.charAt(0)}
              />
            ))}
          </div>
        )}

        {groupHistory.length > 0 && (
          <div className="space-y-2">
            <p className="text-xs font-medium text-muted-foreground">历史记录</p>
            {groupHistory.slice(0, 6).map((request) => (
              <RequestCard
                key={request.id}
                name={request.group_name}
                subtitle={request.status === 'accepted' ? '已加入群组' : '已拒绝'}
                initial={request.group_name.charAt(0)}
                muted
              />
            ))}
          </div>
        )}

        {incomingGroupRequests.length === 0 &&
          outgoingGroupRequests.length === 0 &&
          groupHistory.length === 0 && (
            <p className="rounded-xl border border-dashed border-white/20 bg-white/10 px-4 py-5 text-sm text-muted-foreground">
              当前没有群组请求。后端接入群组审批后会直接显示在这里。
            </p>
          )}
      </RequestSection>
    </div>
  )
}

interface RequestSectionProps {
  title: string
  icon: ReactNode
  count: number
  children: ReactNode
}

function RequestSection({ title, icon, count, children }: RequestSectionProps) {
  return (
    <section className="space-y-3 rounded-2xl border border-white/18 bg-white/12 p-3 shadow-sm backdrop-blur-xl">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-sm font-medium">
          {icon}
          <span>{title}</span>
        </div>
        <Badge variant={count > 0 ? 'destructive' : 'secondary'}>{count}</Badge>
      </div>
      {children}
    </section>
  )
}

interface RequestCardProps {
  name: string
  subtitle: string
  initial: string
  actions?: React.ReactNode
  muted?: boolean
}

function RequestCard({ name, subtitle, initial, actions, muted = false }: RequestCardProps) {
  return (
    <div
      className={cn(
        'flex items-center gap-3 rounded-xl border border-white/18 bg-white/18 px-3 py-3',
        muted && 'opacity-70',
      )}
    >
      <Avatar className="h-10 w-10 rounded-lg">
        <AvatarFallback className="rounded-lg bg-blue-500 text-white">{initial}</AvatarFallback>
      </Avatar>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{name}</p>
        <p className="truncate text-xs text-muted-foreground">{subtitle}</p>
      </div>
      {actions && <div className="flex gap-1">{actions}</div>}
    </div>
  )
}

interface EmptyStateProps {
  title: string
  description: string
}

function EmptyState({ title, description }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center px-6 py-12 text-center">
      <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-white/16 backdrop-blur-xl">
        <Users className="h-7 w-7 text-muted-foreground" />
      </div>
      <p className="text-sm font-medium">{title}</p>
      <p className="mt-2 max-w-[240px] text-sm text-muted-foreground">{description}</p>
    </div>
  )
}

interface AddFriendDialogProps {
  allUsers: User[]
  currentUser: User | null
  friends: Friend[]
  friendRequests: FriendRequest[]
  onSendRequest: (userId: number, message?: string) => Promise<void>
  onClose: () => void
}

function AddFriendDialog({
  allUsers,
  currentUser,
  friends,
  friendRequests,
  onSendRequest,
  onClose,
}: AddFriendDialogProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedUser, setSelectedUser] = useState<User | null>(null)
  const [message, setMessage] = useState('')
  const [isLoading, setIsLoading] = useState(false)

  const availableUsers = useMemo(() => {
    const friendIds = new Set(friends.map((friend) => friend.id))
    const pendingIds = new Set(
      friendRequests
        .filter((request) => request.status === 'pending')
        .map((request) => (request.direction === 'outgoing' ? request.addressee.id : request.requester.id)),
    )

    const rows = allUsers.filter((user) => {
      return (
        user.id !== currentUser?.id &&
        !friendIds.has(user.id) &&
        !pendingIds.has(user.id) &&
        (user.display_name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          user.username.toLowerCase().includes(searchQuery.toLowerCase()))
      )
    })

    return sortByDictionaryOrder(
      rows,
      (user) => user.display_name || user.username,
      (user) => user.username,
    )
  }, [allUsers, currentUser, friends, friendRequests, searchQuery])

  async function handleSend() {
    if (!selectedUser) {
      return
    }
    setIsLoading(true)
    try {
      await onSendRequest(selectedUser.id, message || undefined)
      onClose()
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <DialogContent className="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>添加好友</DialogTitle>
        <DialogDescription>搜索用户并发送好友请求。</DialogDescription>
      </DialogHeader>

      <div className="space-y-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索用户名或昵称"
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
            className="pl-9"
          />
        </div>

        {!selectedUser && (
          <ScrollArea className="h-[220px] rounded-lg border">
            {availableUsers.length === 0 ? (
              <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
                {searchQuery ? '没有找到可添加的用户' : '输入关键词搜索用户'}
              </div>
            ) : (
              <div className="p-1">
                {availableUsers.map((user) => (
                  <button
                    key={user.id}
                    onClick={() => setSelectedUser(user)}
                    className="flex w-full items-center gap-3 rounded-lg px-3 py-2 transition-colors hover:bg-accent"
                  >
                    <Avatar className="h-9 w-9 rounded-lg">
                      <AvatarFallback className="rounded-lg bg-wechat-green text-sm text-white">
                        {user.display_name.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1 text-left">
                      <p className="truncate text-sm font-medium">{user.display_name}</p>
                      <p className="truncate text-xs text-muted-foreground">@{user.username}</p>
                    </div>
                  </button>
                ))}
              </div>
            )}
          </ScrollArea>
        )}

        {selectedUser && (
          <div className="space-y-4">
            <div className="flex items-center gap-3 rounded-lg bg-muted/50 p-3">
              <Avatar className="h-12 w-12 rounded-lg">
                <AvatarFallback className="rounded-lg bg-wechat-green text-white">
                  {selectedUser.display_name.charAt(0)}
                </AvatarFallback>
              </Avatar>
              <div className="min-w-0 flex-1">
                <p className="truncate font-medium">{selectedUser.display_name}</p>
                <p className="truncate text-sm text-muted-foreground">@{selectedUser.username}</p>
              </div>
              <Button variant="ghost" size="sm" onClick={() => setSelectedUser(null)}>
                更换
              </Button>
            </div>

            <div className="space-y-2">
              <Label>验证消息（可选）</Label>
              <Textarea
                placeholder="介绍一下自己吧..."
                value={message}
                onChange={(event) => setMessage(event.target.value)}
                rows={3}
              />
            </div>
          </div>
        )}
      </div>

      <DialogFooter>
        <DialogClose asChild>
          <Button variant="ghost">取消</Button>
        </DialogClose>
        <Button
          onClick={handleSend}
          disabled={!selectedUser || isLoading}
          className="bg-wechat-green hover:bg-wechat-green-dark"
        >
          {isLoading ? '发送中...' : '发送请求'}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}

interface CreateGroupDialogProps {
  friends: Friend[]
  onCreateGroup: (name: string, memberIds: number[]) => Promise<void>
  onClose: () => void
}

function CreateGroupDialog({ friends, onCreateGroup, onClose }: CreateGroupDialogProps) {
  const [groupName, setGroupName] = useState('')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedIds, setSelectedIds] = useState<number[]>([])
  const [isLoading, setIsLoading] = useState(false)

  const selectableFriends = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    const rows = !query
      ? friends
      : friends.filter(
          (friend) =>
            friend.display_name.toLowerCase().includes(query) ||
            friend.username.toLowerCase().includes(query),
        )

    return sortByDictionaryOrder(
      rows,
      (friend) => friend.display_name || friend.username,
      (friend) => friend.username,
    )
  }, [friends, searchQuery])

  function toggleFriend(userId: number) {
    setSelectedIds((current) => {
      if (current.includes(userId)) {
        return current.filter((id) => id !== userId)
      }
      return [...current, userId]
    })
  }

  async function handleCreateGroup() {
    if (groupName.trim().length === 0 || selectedIds.length < 2) {
      return
    }
    setIsLoading(true)
    try {
      await onCreateGroup(groupName.trim(), selectedIds)
      onClose()
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <DialogContent className="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>拉群组</DialogTitle>
        <DialogDescription>先填写群名称，再从好友列表中至少选择 2 位成员。</DialogDescription>
      </DialogHeader>

      <div className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="group-name">群名称</Label>
          <Input
            id="group-name"
            placeholder="例如：项目协作群"
            value={groupName}
            onChange={(event) => setGroupName(event.target.value)}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="group-search">选择好友</Label>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              id="group-search"
              placeholder="搜索好友"
              value={searchQuery}
              onChange={(event) => setSearchQuery(event.target.value)}
              className="pl-9"
            />
          </div>
          <p className="text-xs text-muted-foreground">
            已选择 {selectedIds.length} 人，连同你自己共 {selectedIds.length + 1} 人。
          </p>
        </div>

        <ScrollArea className="h-[260px] rounded-lg border">
          {selectableFriends.length === 0 ? (
            <div className="flex h-full items-center justify-center text-sm text-muted-foreground">
              {friends.length === 0 ? '当前还没有好友，无法创建群组。' : '没有匹配的好友'}
            </div>
          ) : (
            <div className="space-y-1 p-1">
              {selectableFriends.map((friend) => {
                const selected = selectedIds.includes(friend.id)
                return (
                  <button
                    key={friend.id}
                    onClick={() => toggleFriend(friend.id)}
                    className={cn(
                      'flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left transition-colors',
                      selected ? 'bg-wechat-green-light' : 'hover:bg-accent',
                    )}
                  >
                    <Avatar className="h-9 w-9 rounded-lg">
                      <AvatarFallback className="rounded-lg bg-wechat-green text-sm text-white">
                        {friend.display_name.charAt(0)}
                      </AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{friend.display_name}</p>
                      <p className="truncate text-xs text-muted-foreground">@{friend.username}</p>
                    </div>
                    <span
                      className={cn(
                        'flex h-5 w-5 items-center justify-center rounded-full border text-xs',
                        selected
                          ? 'border-wechat-green bg-wechat-green text-white'
                          : 'border-border text-transparent',
                      )}
                    >
                      ✓
                    </span>
                  </button>
                )
              })}
            </div>
          )}
        </ScrollArea>
      </div>

      <DialogFooter>
        <DialogClose asChild>
          <Button variant="ghost">取消</Button>
        </DialogClose>
        <Button
          onClick={handleCreateGroup}
          disabled={groupName.trim().length === 0 || selectedIds.length < 2 || isLoading}
          className="bg-wechat-green hover:bg-wechat-green-dark"
        >
          {isLoading ? '创建中...' : '创建群组'}
        </Button>
      </DialogFooter>
    </DialogContent>
  )
}
