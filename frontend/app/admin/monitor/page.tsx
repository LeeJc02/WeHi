'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { getAdminMe, getMonitorOverview, getMonitorTimeseries, initAdminToken } from '@/lib/admin-api'
import type { MonitorOverview, MonitorTimeseries } from '@/lib/types'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Spinner } from '@/components/ui/spinner'

export default function AdminMonitorPage() {
  const router = useRouter()
  const [overview, setOverview] = useState<MonitorOverview | null>(null)
  const [timeseries, setTimeseries] = useState<MonitorTimeseries | null>(null)

  useEffect(() => {
    let cancelled = false
    let intervalId: number | undefined

    void (async () => {
      initAdminToken()
      try {
        const profile = await getAdminMe()
        if (profile.must_change_password) {
          router.replace('/admin/force-change-password')
          return
        }
        const load = async () => {
          const [nextOverview, nextTimeseries] = await Promise.all([getMonitorOverview(), getMonitorTimeseries()])
          if (!cancelled) {
            setOverview(nextOverview)
            setTimeseries(nextTimeseries)
          }
        }
        await load()
        intervalId = window.setInterval(() => {
          void load()
        }, 5000)
      } catch {
        router.replace('/admin/login')
      }
    })()

    return () => {
      cancelled = true
      if (intervalId) {
        window.clearInterval(intervalId)
      }
    }
  }, [router])

  if (!overview || !timeseries) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  const latestPoint = timeseries.points.at(-1)

  return (
    <main className="min-h-screen bg-background p-6">
      <div className="mx-auto max-w-6xl space-y-6">
        <div>
          <Link href="/admin" className="text-sm text-muted-foreground hover:text-foreground">
            返回后台
          </Link>
          <h1 className="mt-2 text-3xl font-semibold">监控总览</h1>
          <p className="mt-2 text-muted-foreground">每 5 秒轮询三项服务的健康与 metrics，保留最近 1 小时数据。</p>
        </div>

        <div className="grid gap-4 md:grid-cols-4 xl:grid-cols-7">
          <MetricCard title="总请求数" value={overview.total_requests.toFixed(0)} />
          <MetricCard title="4xx 错误" value={overview.client_errors.toFixed(0)} />
          <MetricCard title="5xx 错误" value={overview.server_errors.toFixed(0)} />
          <MetricCard title="平均延迟" value={`${overview.average_latency_ms.toFixed(1)} ms`} />
          <MetricCard title="AI 待重试" value={overview.ai_retry_pending.toFixed(0)} />
          <MetricCard title="AI 已完成" value={overview.ai_retry_completed.toFixed(0)} />
          <MetricCard title="AI 已耗尽" value={overview.ai_retry_exhausted.toFixed(0)} />
        </div>

        <div className="grid gap-4 md:grid-cols-2">
          <Card>
            <CardHeader>
              <CardTitle>服务状态</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              {overview.services.map((service) => (
                <div key={service.name} className="flex items-center justify-between rounded-lg border px-3 py-2">
                  <div>
                    <p className="font-medium">{service.name}</p>
                    <p className="text-xs text-muted-foreground">{service.checked_at}</p>
                  </div>
                  <span className={service.healthy ? 'text-green-600' : 'text-red-600'}>
                    {service.status}
                  </span>
                </div>
              ))}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>实时摘要</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <p>WebSocket 连接数：{overview.websocket_connections.toFixed(0)}</p>
              <p>AI 待重试作业：{overview.ai_retry_pending.toFixed(0)}</p>
              <p>AI 已耗尽作业：{overview.ai_retry_exhausted.toFixed(0)}</p>
              <p>最新快照时间：{overview.snapshot_at}</p>
              <p>时间序列点数：{timeseries.points.length}</p>
              {latestPoint ? <p>最近点延迟：{latestPoint.average_latency_ms.toFixed(1)} ms</p> : null}
            </CardContent>
          </Card>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>最近时间序列</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {timeseries.points.slice(-12).map((point) => (
              <div key={point.timestamp} className="grid grid-cols-7 gap-3 rounded-lg border px-3 py-2 text-sm">
                <span>{new Date(point.timestamp).toLocaleTimeString('zh-CN')}</span>
                <span>请求 {point.total_requests.toFixed(0)}</span>
                <span>4xx {point.client_errors.toFixed(0)}</span>
                <span>5xx {point.server_errors.toFixed(0)}</span>
                <span>延迟 {point.average_latency_ms.toFixed(1)} ms</span>
                <span>待重试 {point.ai_retry_pending.toFixed(0)}</span>
                <span>耗尽 {point.ai_retry_exhausted.toFixed(0)}</span>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </main>
  )
}

function MetricCard({ title, value }: { title: string; value: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold">{value}</p>
      </CardContent>
    </Card>
  )
}
