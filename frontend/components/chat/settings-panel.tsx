'use client'

import { useEffect, useState } from 'react'
import { getSessions, logoutOthers, updateProfile } from '@/lib/api'
import { useAuth } from '@/lib/auth-context'
import type { Session } from '@/lib/types'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
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
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { useTheme } from 'next-themes'
import {
  Bell,
  ChevronRight,
  Info,
  LogOut,
  MonitorSmartphone,
  Moon,
  Shield,
  User,
} from 'lucide-react'

const NOTIFICATION_PREF_KEY = 'chat_notifications_enabled'

type SettingsActionItem = {
  icon: typeof User
  label: string
  onClick?: () => void
  description?: string
  toggle?: false
}

type SettingsToggleItem = {
  icon: typeof User
  label: string
  checked: boolean
  onToggle: (checked: boolean) => void
  toggle: true
  description?: string
}

type SettingsItem = SettingsActionItem | SettingsToggleItem

export function SettingsPanel() {
  const { user, logout, refreshUser } = useAuth()
  const { resolvedTheme, setTheme } = useTheme()
  const [profileOpen, setProfileOpen] = useState(false)
  const [sessionsOpen, setSessionsOpen] = useState(false)
  const [displayName, setDisplayName] = useState(user?.display_name || '')
  const [avatarUrl, setAvatarUrl] = useState(user?.avatar_url || '')
  const [notificationsEnabled, setNotificationsEnabled] = useState(true)
  const [sessions, setSessions] = useState<Session[]>([])
  const [isSaving, setIsSaving] = useState(false)
  const [isLoadingSessions, setIsLoadingSessions] = useState(false)
  const [isRevokingOthers, setIsRevokingOthers] = useState(false)

  useEffect(() => {
    setDisplayName(user?.display_name || '')
    setAvatarUrl(user?.avatar_url || '')
  }, [user?.avatar_url, user?.display_name])

  useEffect(() => {
    if (typeof window === 'undefined') {
      return
    }
    setNotificationsEnabled(window.localStorage.getItem(NOTIFICATION_PREF_KEY) !== 'false')
  }, [])

  async function openSessions() {
    setSessionsOpen(true)
    setIsLoadingSessions(true)
    try {
      setSessions(await getSessions())
    } finally {
      setIsLoadingSessions(false)
    }
  }

  async function handleSaveProfile() {
    setIsSaving(true)
    try {
      await updateProfile({ display_name: displayName.trim(), avatar_url: avatarUrl.trim() })
      await refreshUser()
      setProfileOpen(false)
    } finally {
      setIsSaving(false)
    }
  }

  async function handleLogoutOthers() {
    setIsRevokingOthers(true)
    try {
      await logoutOthers()
      setSessions(await getSessions())
    } finally {
      setIsRevokingOthers(false)
    }
  }

  const settingsGroups: { title: string; items: SettingsItem[] }[] = [
    {
      title: '账号',
      items: [
        { icon: User, label: '个人信息', onClick: () => setProfileOpen(true) },
        { icon: Shield, label: '设备会话管理', onClick: () => void openSessions() },
      ],
    },
    {
      title: '通用',
      items: [
        {
          icon: Bell,
          label: '消息通知',
          toggle: true,
          checked: notificationsEnabled,
          onToggle: (checked: boolean) => {
            setNotificationsEnabled(checked)
            if (typeof window !== 'undefined') {
              window.localStorage.setItem(NOTIFICATION_PREF_KEY, String(checked))
            }
          },
        },
        {
          icon: Moon,
          label: '深色模式',
          toggle: true,
          checked: resolvedTheme === 'dark',
          onToggle: (checked: boolean) => setTheme(checked ? 'dark' : 'light'),
        },
      ],
    },
    {
      title: '其他',
      items: [
        { icon: Info, label: '关于', description: '当前重点展示会话、同步和实时链路。' },
      ],
    },
  ]

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b border-panel-border">
        <h2 className="font-semibold text-lg">设置</h2>
      </div>

      <ScrollArea className="flex-1">
        <div className="p-4">
          <button
            className="w-full flex items-center gap-4 p-4 rounded-xl bg-card hover:bg-accent/50 transition-colors"
            onClick={() => setProfileOpen(true)}
          >
            <Avatar className="h-14 w-14 rounded-xl">
              {user?.avatar_url ? (
                <img src={user.avatar_url} alt={user.display_name} className="h-full w-full rounded-xl object-cover" />
              ) : null}
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

        <div className="px-4 pb-4 space-y-4">
          {settingsGroups.map((group) => (
            <div key={group.title} className="rounded-xl bg-card overflow-hidden">
              <div className="px-4 py-2 text-xs font-medium text-muted-foreground uppercase tracking-wider">
                {group.title}
              </div>
              {group.items.map((item, index) => (
                <div key={item.label}>
                  {index > 0 && <Separator className="mx-4" />}
                  <div className="w-full flex items-center gap-3 px-4 py-3">
                    <item.icon className="h-5 w-5 text-muted-foreground" />
                    <button
                      className="flex-1 text-left"
                      onClick={'onClick' in item ? item.onClick : undefined}
                      disabled={!('onClick' in item && item.onClick)}
                    >
                      <p className="text-sm">{item.label}</p>
                      {'description' in item && item.description ? (
                        <p className="text-xs text-muted-foreground mt-1">{item.description}</p>
                      ) : null}
                    </button>
                    {'toggle' in item && item.toggle ? (
                      <Switch checked={item.checked} onCheckedChange={item.onToggle} />
                    ) : (
                      <ChevronRight className="h-4 w-4 text-muted-foreground" />
                    )}
                  </div>
                </div>
              ))}
            </div>
          ))}

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

      <Dialog open={profileOpen} onOpenChange={setProfileOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>个人资料</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">昵称</label>
              <Input value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">头像 URL</label>
              <Input
                value={avatarUrl}
                onChange={(event) => setAvatarUrl(event.target.value)}
                placeholder="https://example.com/avatar.png"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setProfileOpen(false)}>
              取消
            </Button>
            <Button
              onClick={() => void handleSaveProfile()}
              disabled={isSaving || !displayName.trim()}
              className="bg-wechat-green hover:bg-wechat-green-dark"
            >
              {isSaving ? '保存中...' : '保存'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={sessionsOpen} onOpenChange={setSessionsOpen}>
        <DialogContent className="sm:max-w-2xl">
          <DialogHeader>
            <DialogTitle>设备会话管理</DialogTitle>
            <DialogDescription>查看当前登录设备，并可一键退出其他设备。</DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="rounded-lg border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
              当前共 {sessions.length} 个活跃设备会话
            </div>

            <ScrollArea className="h-72 rounded-lg border">
              <div className="space-y-2 p-3">
                {isLoadingSessions ? (
                  <p className="text-sm text-muted-foreground">正在加载设备会话...</p>
                ) : sessions.length === 0 ? (
                  <p className="text-sm text-muted-foreground">暂无活跃设备</p>
                ) : (
                  sessions.map((session) => (
                    <div key={session.id} className="rounded-lg border bg-white/60 px-3 py-3">
                      <div className="flex items-start gap-3">
                        <div className="mt-0.5 rounded-lg bg-wechat-green/10 p-2 text-wechat-green">
                          <MonitorSmartphone className="h-4 w-4" />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2">
                            <p className="truncate text-sm font-medium">{session.device_id || 'browser'}</p>
                            {session.current && <Badge>当前设备</Badge>}
                          </div>
                          <p className="truncate text-xs text-muted-foreground mt-1">{session.user_agent || '未知 UA'}</p>
                          <p className="text-xs text-muted-foreground mt-1">
                            最近活跃：{new Date(session.last_seen_at).toLocaleString('zh-CN')}
                          </p>
                          <p className="text-xs text-muted-foreground mt-1">
                            过期时间：{new Date(session.expires_at).toLocaleString('zh-CN')}
                          </p>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </ScrollArea>
          </div>

          <DialogFooter className="sm:justify-between">
            <Button
              variant="destructive"
              onClick={() => void handleLogoutOthers()}
              disabled={isRevokingOthers || sessions.filter((item) => !item.current).length === 0}
            >
              {isRevokingOthers ? '处理中...' : '退出其他设备'}
            </Button>
            <Button variant="ghost" onClick={() => setSessionsOpen(false)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
