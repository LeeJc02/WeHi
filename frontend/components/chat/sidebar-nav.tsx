'use client'

import { cn } from '@/lib/utils'
import { useAuth } from '@/lib/auth-context'
import { useChatStore } from '@/lib/chat-store'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
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
  MessageCircle,
  Users,
  Settings,
  LogOut,
  User,
  Bell,
} from 'lucide-react'

export function SidebarNav() {
  const { user, logout } = useAuth()
  const { activeView, setActiveView, conversations, friendRequests, groupRequests, isConnected } = useChatStore()

  const pendingRequests = friendRequests.filter(
    (r) => r.status === 'pending' && r.direction === 'incoming'
  ).length + groupRequests.filter((r) => r.status === 'pending' && r.direction === 'incoming').length

  const unreadMessages = conversations.reduce((total, conversation) => {
    return total + Math.max(0, conversation.unread_count)
  }, 0)

  const navItems = [
    {
      id: 'chat' as const,
      icon: MessageCircle,
      label: '聊天',
      badge: unreadMessages > 0 ? unreadMessages : undefined,
    },
    {
      id: 'contacts' as const,
      icon: Users,
      label: '通讯录',
      badge: pendingRequests > 0 ? pendingRequests : undefined,
    },
  ]

  return (
    <TooltipProvider delayDuration={100}>
      <div className="flex h-full w-16 flex-col items-center gap-2 border-r border-sidebar-border/50 bg-sidebar/72 py-4 backdrop-blur-xl">
        {/* 用户头像 */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="ghost"
              className="relative w-10 h-10 rounded-lg p-0 hover:bg-sidebar-accent"
            >
              <Avatar className="w-10 h-10 rounded-lg">
                <AvatarFallback className="bg-wechat-green text-white rounded-lg font-medium">
                  {user?.display_name?.charAt(0) || user?.username?.charAt(0) || 'U'}
                </AvatarFallback>
              </Avatar>
              {/* 在线状态指示器 */}
              <span
                className={cn(
                  'absolute bottom-0 right-0 w-3 h-3 rounded-full border-2 border-sidebar',
                  isConnected ? 'bg-green-500' : 'bg-gray-400'
                )}
              />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" side="right" className="w-56">
            <div className="px-3 py-2">
              <p className="font-medium">{user?.display_name}</p>
              <p className="text-sm text-muted-foreground">@{user?.username}</p>
            </div>
            <DropdownMenuSeparator />
            <DropdownMenuItem>
              <User className="mr-2 h-4 w-4" />
              个人资料
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Bell className="mr-2 h-4 w-4" />
              通知设置
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className="text-destructive focus:text-destructive"
              onClick={logout}
            >
              <LogOut className="mr-2 h-4 w-4" />
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>

        {/* 分隔线 */}
        <div className="w-8 h-px bg-sidebar-border my-2" />

        {/* 导航项 */}
        <nav className="flex-1 flex flex-col gap-1">
          {navItems.map((item) => (
            <Tooltip key={item.id}>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className={cn(
                    'relative w-10 h-10 rounded-lg transition-colors',
                    activeView === item.id
                      ? 'bg-sidebar-accent text-wechat-green'
                      : 'text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
                  )}
                  onClick={() => setActiveView(item.id)}
                >
                  <item.icon className="h-5 w-5" />
                  {item.badge && (
                    <span className="absolute -top-1 -right-1 min-w-[18px] h-[18px] rounded-full bg-red-500 text-white text-xs flex items-center justify-center px-1">
                      {item.badge > 99 ? '99+' : item.badge}
                    </span>
                  )}
                </Button>
              </TooltipTrigger>
              <TooltipContent side="right">
                <p>{item.label}</p>
              </TooltipContent>
            </Tooltip>
          ))}
        </nav>

        {/* 底部设置 */}
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant="ghost"
              size="icon"
              className={cn(
                'w-10 h-10 rounded-lg transition-colors',
                activeView === 'settings'
                  ? 'bg-sidebar-accent text-wechat-green'
                  : 'text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground'
              )}
              onClick={() => setActiveView('settings')}
            >
              <Settings className="h-5 w-5" />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="right">
            <p>设置</p>
          </TooltipContent>
        </Tooltip>
      </div>
    </TooltipProvider>
  )
}
