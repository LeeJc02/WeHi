'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useParams } from 'next/navigation'
import { getAdminMe, getMessageJourney, initAdminToken } from '@/lib/admin-api'
import type { MessageJourney } from '@/lib/types'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'

export default function AdminMessageJourneyPage() {
  const params = useParams<{ id: string }>()
  const [journey, setJourney] = useState<MessageJourney | null>(null)
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
        setJourney(await getMessageJourney(Number(params.id)))
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
      <div className="mx-auto max-w-5xl space-y-6">
        <div>
          <Link href="/admin" className="text-sm text-muted-foreground hover:text-foreground">返回后台</Link>
          <h1 className="mt-2 text-3xl font-semibold">消息旅程</h1>
          <p className="mt-2 text-muted-foreground">消息 #{params.id} 的服务端阶段轨迹。</p>
        </div>

        {journey ? (
          <>
            <Card>
              <CardHeader>
                <CardTitle>消息摘要</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-3 md:grid-cols-3">
                <Stat label="会话 ID" value={String(journey.conversation_id)} />
                <Stat label="发送者" value={String(journey.sender_id)} />
                <Stat label="投递状态" value={journey.delivery_status} />
                <Stat label="客户端消息 ID" value={journey.client_msg_id || '-'} />
                <Stat label="类型" value={journey.message_type} />
                <Stat label="创建时间" value={new Date(journey.created_at).toLocaleString('zh-CN')} />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>阶段时间线</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {journey.stages.map((stage, index) => (
                  <div key={`${stage.name}-${index}`} className="rounded-xl border border-border/60 bg-white/70 p-4">
                    <div className="flex items-center justify-between gap-4">
                      <div>
                        <p className="font-medium">{stage.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {stage.recipient_id ? `接收方 #${stage.recipient_id}` : '系统阶段'}
                        </p>
                      </div>
                      <span className="text-xs text-muted-foreground">{new Date(stage.occurred_at).toLocaleString('zh-CN')}</span>
                    </div>
                    {stage.note ? <p className="mt-2 text-sm text-muted-foreground">{stage.note}</p> : null}
                  </div>
                ))}
              </CardContent>
            </Card>
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
