'use client'

import { useAuth } from '@/lib/auth-context'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import {
  User,
  Bell,
  Shield,
  HelpCircle,
  Info,
  LogOut,
  ChevronRight,
  Moon,
} from 'lucide-react'

export function SettingsPanel() {
  const { user, logout } = useAuth()

  const settingsGroups = [
    {
      title: '账号',
      items: [
        { icon: User, label: '个人信息', onClick: () => {} },
        { icon: Shield, label: '账号与安全', onClick: () => {} },
      ],
    },
    {
      title: '通用',
      items: [
        {
          icon: Bell,
          label: '消息通知',
          onClick: () => {},
        },
        {
          icon: Moon,
          label: '深色模式',
          toggle: true,
        },
      ],
    },
    {
      title: '其他',
      items: [
        { icon: HelpCircle, label: '帮助与反馈', onClick: () => {} },
        { icon: Info, label: '关于', onClick: () => {} },
      ],
    },
  ]

  return (
    <div className="h-full flex flex-col">
      {/* 头部 */}
      <div className="p-4 border-b border-panel-border">
        <h2 className="font-semibold text-lg">设置</h2>
      </div>

      <ScrollArea className="flex-1">
        {/* 用户信息卡片 */}
        <div className="p-4">
          <button className="w-full flex items-center gap-4 p-4 rounded-xl bg-card hover:bg-accent/50 transition-colors">
            <Avatar className="h-14 w-14 rounded-xl">
              <AvatarFallback className="rounded-xl bg-wechat-green text-white text-xl">
                {user?.display_name?.charAt(0) || 'U'}
              </AvatarFallback>
            </Avatar>
            <div className="flex-1 text-left">
              <p className="font-semibold text-lg">{user?.display_name}</p>
              <p className="text-sm text-muted-foreground">@{user?.username}</p>
            </div>
            <ChevronRight className="h-5 w-5 text-muted-foreground" />
          </button>
        </div>

        {/* 设置列表 */}
        <div className="px-4 pb-4 space-y-4">
          {settingsGroups.map((group) => (
            <div key={group.title} className="rounded-xl bg-card overflow-hidden">
              <div className="px-4 py-2 text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {group.title}
              </div>
              {group.items.map((item, index) => (
                <div key={item.label}>
                  {index > 0 && <Separator className="mx-4" />}
                  <button
                    className="w-full flex items-center gap-3 px-4 py-3 hover:bg-accent/50 transition-colors"
                    onClick={item.onClick}
                  >
                    <item.icon className="h-5 w-5 text-muted-foreground" />
                    <span className="flex-1 text-left text-sm">{item.label}</span>
                    {item.toggle ? (
                      <Switch />
                    ) : (
                      <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    )}
                  </button>
                </div>
              ))}
            </div>
          ))}

          {/* 退出登录按钮 */}
          <Button
            variant="outline"
            className="w-full text-destructive hover:text-destructive hover:bg-destructive/10"
            onClick={logout}
          >
            <LogOut className="mr-2 h-4 w-4" />
            退出登录
          </Button>
        </div>
      </ScrollArea>
    </div>
  )
}
