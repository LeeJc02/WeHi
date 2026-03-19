'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { clearAdminToken, getAdminMe, initAdminToken, resolveMessageByClientMsgID, triggerSearchReindex } from '@/lib/admin-api'
import type { AdminProfile } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import Link from 'next/link'

export default function AdminHomePage() {
  const router = useRouter()
  const [profile, setProfile] = useState<AdminProfile | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [clientMsgID, setClientMsgID] = useState('')
  const [isResolving, setIsResolving] = useState(false)
  const [resolveError, setResolveError] = useState('')
  const [reindexing, setReindexing] = useState(false)

  useEffect(() => {
    void (async () => {
      initAdminToken()
      try {
        const next = await getAdminMe()
        if (next.must_change_password) {
          router.replace('/admin/force-change-password')
          return
        }
        setProfile(next)
      } catch {
        router.replace('/admin/login')
      } finally {
        setIsLoading(false)
      }
    })()
  }, [router])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  return (
    <main className="min-h-screen bg-[linear-gradient(180deg,#f8fbf9,#eef3f0)] p-6">
      <div className="mx-auto max-w-6xl">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <p className="text-sm text-muted-foreground">管理员</p>
            <h1 className="text-3xl font-semibold">后台控制台</h1>
            <p className="mt-2 text-muted-foreground">当前登录账号：{profile?.username}</p>
          </div>
          <Button
            variant="outline"
            onClick={() => {
              clearAdminToken()
              router.replace('/admin/login')
            }}
          >
            退出登录
          </Button>
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          <Card>
            <CardHeader>
              <CardTitle>AI 配置</CardTitle>
              <CardDescription>统一管理 Bot Prompt、Provider 和模型。</CardDescription>
            </CardHeader>
            <CardContent>
              <Button asChild className="w-full bg-wechat-green hover:bg-wechat-green-dark">
                <Link href="/admin/ai">进入配置</Link>
              </Button>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>监控总览</CardTitle>
              <CardDescription>查看健康状态、请求量、错误率和实时指标。</CardDescription>
            </CardHeader>
            <CardContent>
              <Button asChild className="w-full bg-wechat-green hover:bg-wechat-green-dark">
                <Link href="/admin/monitor">查看监控</Link>
              </Button>
            </CardContent>
          </Card>
          <Card>
            <CardHeader>
              <CardTitle>AI 审计</CardTitle>
              <CardDescription>审计 AI 调用记录、耗时和错误信息。</CardDescription>
            </CardHeader>
            <CardContent>
              <Button asChild className="w-full bg-wechat-green hover:bg-wechat-green-dark">
                <Link href="/admin/audit">查看审计</Link>
              </Button>
            </CardContent>
          </Card>
        </div>

        <Card className="mt-6">
          <CardHeader>
            <CardTitle>消息快速定位</CardTitle>
            <CardDescription>输入 `client_msg_id`，直接跳转到消息旅程页。</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 md:flex-row">
            <Input
              placeholder="例如 runtime-1740000000"
              value={clientMsgID}
              onChange={(event) => {
                setClientMsgID(event.target.value)
                setResolveError('')
              }}
            />
            <Button
              className="bg-wechat-green hover:bg-wechat-green-dark"
              disabled={isResolving || clientMsgID.trim().length === 0}
              onClick={() => {
                void (async () => {
                  setIsResolving(true)
                  setResolveError('')
                  try {
                    const result = await resolveMessageByClientMsgID({ clientMsgId: clientMsgID.trim() })
                    router.push(`/admin/messages/${result.message_id}`)
                  } catch (error) {
                    setResolveError(error instanceof Error ? error.message : '消息定位失败')
                  } finally {
                    setIsResolving(false)
                  }
                })()
              }}
            >
              {isResolving ? '定位中...' : '定位消息'}
            </Button>
          </CardContent>
          {resolveError ? <CardContent className="pt-0 text-sm text-red-500">{resolveError}</CardContent> : null}
        </Card>

        <Card className="mt-6">
          <CardHeader>
            <CardTitle>补偿入口</CardTitle>
            <CardDescription>当搜索索引漂移或需要全量修复时，触发一次后端重建。</CardDescription>
          </CardHeader>
          <CardContent>
            <Button
              variant="outline"
              disabled={reindexing}
              onClick={() => {
                void (async () => {
                  setReindexing(true)
                  try {
                    await triggerSearchReindex()
                  } finally {
                    setReindexing(false)
                  }
                })()
              }}
            >
              {reindexing ? '重建中...' : '重建搜索索引'}
            </Button>
          </CardContent>
        </Card>
      </div>
    </main>
  )
}
