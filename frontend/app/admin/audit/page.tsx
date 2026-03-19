'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { getAIAuditLogDetail, getAIAuditLogs, getAdminMe, initAdminToken } from '@/lib/admin-api'
import type { AIAuditLog, AIAuditLogDetail } from '@/lib/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'

export default function AdminAuditPage() {
  const [rows, setRows] = useState<AIAuditLog[]>([])
  const [selected, setSelected] = useState<AIAuditLogDetail | null>(null)
  const [provider, setProvider] = useState('')
  const [status, setStatus] = useState('')
  const [userId, setUserId] = useState('')
  const [conversationId, setConversationId] = useState('')
  const [isLoading, setIsLoading] = useState(true)

  async function load(detailId?: number) {
    const nextRows = await getAIAuditLogs({
      provider: provider || undefined,
      status: status || undefined,
      userId: userId ? Number(userId) : undefined,
      conversationId: conversationId ? Number(conversationId) : undefined,
      limit: 50,
    })
    setRows(nextRows)
    const targetId = detailId ?? nextRows[0]?.id
    if (targetId) {
      setSelected(await getAIAuditLogDetail(targetId))
    } else {
      setSelected(null)
    }
  }

  useEffect(() => {
    void (async () => {
      initAdminToken()
      try {
        const searchParams = new URLSearchParams(window.location.search)
        setProvider(searchParams.get('provider') || '')
        setStatus(searchParams.get('status') || '')
        setUserId(searchParams.get('user_id') || '')
        setConversationId(searchParams.get('conversation_id') || '')
        const profile = await getAdminMe()
        if (profile.must_change_password) {
          window.location.replace('/admin/force-change-password')
          return
        }
        const nextProvider = searchParams.get('provider') || ''
        const nextStatus = searchParams.get('status') || ''
        const nextUserId = searchParams.get('user_id') || ''
        const nextConversationId = searchParams.get('conversation_id') || ''
        const nextRows = await getAIAuditLogs({
          provider: nextProvider || undefined,
          status: nextStatus || undefined,
          userId: nextUserId ? Number(nextUserId) : undefined,
          conversationId: nextConversationId ? Number(nextConversationId) : undefined,
          limit: 50,
        })
        setRows(nextRows)
        const targetId = nextRows[0]?.id
        if (targetId) {
          setSelected(await getAIAuditLogDetail(targetId))
        } else {
          setSelected(null)
        }
      } catch {
        window.location.replace('/admin/login')
      } finally {
        setIsLoading(false)
      }
    })()
  }, [])

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  return (
    <main className="min-h-screen bg-[linear-gradient(180deg,#f8fbf9,#eef3f0)] p-6">
      <div className="mx-auto max-w-7xl space-y-6">
        <div>
          <Link href="/admin" className="text-sm text-muted-foreground hover:text-foreground">
            返回后台
          </Link>
          <h1 className="mt-2 text-3xl font-semibold">AI 审计</h1>
          <p className="mt-2 text-muted-foreground">筛选模型调用记录，查看请求载荷、响应摘要和失败原因。</p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>筛选条件</CardTitle>
            <CardDescription>支持按 Provider 和状态查看最近 50 条调用。</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 md:flex-row">
            <Input placeholder="Provider，例如 zhipu" value={provider} onChange={(event) => setProvider(event.target.value)} />
            <Input placeholder="状态，例如 success / error" value={status} onChange={(event) => setStatus(event.target.value)} />
            <Input placeholder="用户 ID" value={userId} onChange={(event) => setUserId(event.target.value)} />
            <Input placeholder="会话 ID" value={conversationId} onChange={(event) => setConversationId(event.target.value)} />
            <Button className="bg-wechat-green hover:bg-wechat-green-dark" onClick={() => void load(selected?.id)}>
              刷新列表
            </Button>
          </CardContent>
        </Card>

        <div className="grid gap-6 lg:grid-cols-[1.2fr_1fr]">
          <Card className="min-h-[560px]">
            <CardHeader>
              <CardTitle>调用列表</CardTitle>
              <CardDescription>{rows.length} 条记录</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {rows.length === 0 ? (
                <p className="text-sm text-muted-foreground">暂无符合条件的审计记录。</p>
              ) : (
                rows.map((row) => (
                  <button
                    key={row.id}
                    onClick={() => void load(row.id)}
                    className={`w-full rounded-xl border p-4 text-left transition hover:border-wechat-green ${selected?.id === row.id ? 'border-wechat-green bg-wechat-green/5' : 'border-border/60 bg-white/70'}`}
                  >
                    <div className="flex items-center justify-between gap-4">
                      <div>
                        <p className="font-medium">{row.provider} / {row.model}</p>
                        <p className="text-xs text-muted-foreground">用户 #{row.user_id} · 会话 #{row.conversation_id}</p>
                      </div>
                      <span className={`rounded-full px-2 py-1 text-xs ${row.status === 'success' ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-700'}`}>
                        {row.status}
                      </span>
                    </div>
                    <p className="mt-3 line-clamp-2 text-sm text-muted-foreground">{row.input_preview || '无输入摘要'}</p>
                    <div className="mt-3 flex gap-4 text-xs text-muted-foreground">
                      <span>{row.duration_ms} ms</span>
                      <span>{row.total_tokens} tokens</span>
                      <span>{new Date(row.created_at).toLocaleString('zh-CN')}</span>
                    </div>
                  </button>
                ))
              )}
            </CardContent>
          </Card>

          <Card className="min-h-[560px]">
            <CardHeader>
              <CardTitle>详情</CardTitle>
              <CardDescription>查看摘要、错误信息和原始 JSON。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {!selected ? (
                <p className="text-sm text-muted-foreground">选择左侧记录查看详情。</p>
              ) : (
                <>
                  <div className="grid gap-3 md:grid-cols-2">
                    <DetailItem label="请求 ID" value={selected.request_id || '-'} />
                    <DetailItem label="耗时" value={`${selected.duration_ms} ms`} />
                    <DetailItem label="输入 Tokens" value={String(selected.input_tokens)} />
                    <DetailItem label="输出 Tokens" value={String(selected.output_tokens)} />
                    <DetailItem label="总 Tokens" value={String(selected.total_tokens)} />
                    <DetailItem label="状态" value={selected.status} />
                  </div>
                  <DetailBlock title="输入摘要" value={selected.input_preview || '-'} />
                  <DetailBlock title="输出摘要" value={selected.output_preview || '-'} />
                  {selected.error_message ? <DetailBlock title="错误信息" value={`${selected.error_code || 'AI_PROVIDER_ERROR'}: ${selected.error_message}`} /> : null}
                  <DetailBlock title="请求 JSON" value={selected.request_payload_json} pre />
                  <DetailBlock title="响应 JSON" value={selected.response_payload_json} pre />
                </>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </main>
  )
}

function DetailItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border/60 bg-white/70 p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 text-sm font-medium">{value}</p>
    </div>
  )
}

function DetailBlock({ title, value, pre = false }: { title: string; value: string; pre?: boolean }) {
  return (
    <div className="rounded-xl border border-border/60 bg-white/70 p-3">
      <p className="text-xs text-muted-foreground">{title}</p>
      {pre ? (
        <pre className="mt-2 max-h-64 overflow-auto whitespace-pre-wrap break-all text-xs text-foreground">{value}</pre>
      ) : (
        <p className="mt-2 whitespace-pre-wrap break-words text-sm text-foreground">{value}</p>
      )}
    </div>
  )
}
