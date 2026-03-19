'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { getAdminMe, getConversationConsistency, getConversationEvents, initAdminToken } from '@/lib/admin-api'
import type { ConversationConsistency, SyncEvent } from '@/lib/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'

export default function AdminConversationDiagnosticsPage() {
  const params = useParams<{ id: string }>()
  const [consistency, setConsistency] = useState<ConversationConsistency | null>(null)
  const [events, setEvents] = useState<SyncEvent[]>([])
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    void (async () => {
      initAdminToken()
      try {
        const profile = await getAdminMe()
        if (profile.must_change_password) {
          window.location.replace('/admin/force-change-password')
          return
        }
        const conversationId = Number(params.id)
        const [nextConsistency, nextEvents] = await Promise.all([
          getConversationConsistency(conversationId),
          getConversationEvents(conversationId, 100),
        ])
        setConsistency(nextConsistency)
        setEvents(nextEvents)
      } catch {
        window.location.replace('/admin/login')
      } finally {
        setIsLoading(false)
      }
    })()
  }, [params.id])

  if (isLoading) {
    return <div className="flex min-h-screen items-center justify-center bg-background"><Spinner className="h-8 w-8 text-wechat-green" /></div>
  }

  return (
    <main className="min-h-screen bg-[linear-gradient(180deg,#f8fbf9,#eef3f0)] p-6">
      <div className="mx-auto max-w-6xl space-y-6">
        <div>
          <Link href="/admin" className="text-sm text-muted-foreground hover:text-foreground">返回后台</Link>
          <h1 className="mt-2 text-3xl font-semibold">会话一致性诊断</h1>
          <p className="mt-2 text-muted-foreground">会话 #{params.id} 的已读游标、在线状态和最近事件。</p>
        </div>

        {consistency ? (
          <>
            <Card>
              <CardHeader>
                <CardTitle>会话状态</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-3 md:grid-cols-4">
                <Stat label="最后消息 Seq" value={String(consistency.last_message_seq)} />
                <Stat label="最后消息时间" value={consistency.last_message_at ? new Date(consistency.last_message_at).toLocaleString('zh-CN') : '-'} />
                <Stat label="在线成员数" value={String(consistency.online_count)} />
                <Stat label="当前事件游标" value={String(consistency.current_event_lag)} />
              </CardContent>
            </Card>

            <div className="grid gap-6 lg:grid-cols-[1.2fr_1fr]">
              <Card>
                <CardHeader>
                  <CardTitle>成员一致性</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {consistency.members.map((member) => (
                    <div key={member.user_id} className="rounded-xl border border-border/60 bg-white/70 p-4">
                      <div className="flex items-center justify-between gap-4">
                        <div>
                          <p className="font-medium">{member.display_name} <span className="text-xs text-muted-foreground">@{member.username}</span></p>
                          <p className="text-xs text-muted-foreground">#{member.user_id} · {member.role}</p>
                        </div>
                        <span className={`rounded-full px-2 py-1 text-xs ${member.online ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-600'}`}>
                          {member.online ? '在线' : '离线'}
                        </span>
                      </div>
                      <div className="mt-3 grid gap-2 md:grid-cols-3">
                        <Stat label="Last Read Seq" value={String(member.last_read_seq)} />
                        <Stat label="未读数" value={String(member.unread_count)} />
                        <Stat label="Sync Cursor" value={String(member.current_cursor)} />
                      </div>
                    </div>
                  ))}
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle>事件时间线</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  {events.length === 0 ? (
                    <p className="text-sm text-muted-foreground">暂无事件。</p>
                  ) : (
                    events.map((event) => (
                      <div key={event.event_id} className="rounded-xl border border-border/60 bg-white/70 p-3">
                        <div className="flex items-center justify-between gap-3">
                          <p className="font-medium">{event.event_type}</p>
                          <span className="text-xs text-muted-foreground">#{event.cursor}</span>
                        </div>
                        <p className="mt-1 text-xs text-muted-foreground">{new Date(event.created_at).toLocaleString('zh-CN')}</p>
                        <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-all text-xs text-foreground">
                          {JSON.stringify(event.payload, null, 2)}
                        </pre>
                      </div>
                    ))
                  )}
                </CardContent>
              </Card>
            </div>
          </>
        ) : null}
      </div>
    </main>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border/60 bg-white/70 p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 text-sm font-medium break-all">{value}</p>
    </div>
  )
}
