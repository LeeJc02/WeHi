'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { cleanupAIRetryJobs, getAIConfig, getAdminMe, getAIRetryJobDetail, getAIRetryJobs, initAdminToken, retryAIJobNow, retryAIJobs, updateAIConfig } from '@/lib/admin-api'
import type { AIConfig, AIRetryJob, AIRetryJobDetail } from '@/lib/types'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

export default function AdminAIPage() {
  const router = useRouter()
  const [config, setConfig] = useState<AIConfig | null>(null)
  const [retryJobs, setRetryJobs] = useState<AIRetryJob[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [isRefreshingJobs, setIsRefreshingJobs] = useState(false)
  const [retryStatusFilter, setRetryStatusFilter] = useState<'all' | 'pending' | 'completed' | 'exhausted'>('all')
  const [selectedRetryJob, setSelectedRetryJob] = useState<AIRetryJobDetail | null>(null)
  const [isLoadingRetryDetail, setIsLoadingRetryDetail] = useState(false)
  const [error, setError] = useState('')

  async function loadRetryJobs(status = retryStatusFilter) {
    setRetryJobs(await getAIRetryJobs({
      limit: 20,
      status: status === 'all' ? undefined : status,
    }))
  }

  useEffect(() => {
    void (async () => {
      initAdminToken()
      try {
        const profile = await getAdminMe()
        if (profile.must_change_password) {
          router.replace('/admin/force-change-password')
          return
        }
        const [nextConfig, jobs] = await Promise.all([
          getAIConfig(),
          getAIRetryJobs({ limit: 20 }),
        ])
        setConfig(nextConfig)
        setRetryJobs(jobs)
      } catch {
        router.replace('/admin/login')
      } finally {
        setIsLoading(false)
      }
    })()
  }, [router])

  async function handleSave() {
    if (!config) {
      return
    }
    setIsSaving(true)
    setError('')
    try {
      setConfig(await updateAIConfig(config))
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存 AI 配置失败')
    } finally {
      setIsSaving(false)
    }
  }

  async function handleRetryJob(id: number) {
    setIsRefreshingJobs(true)
    setError('')
    try {
      await retryAIJobNow(id)
      await loadRetryJobs()
    } catch (err) {
      setError(err instanceof Error ? err.message : '重试作业提交失败')
    } finally {
      setIsRefreshingJobs(false)
    }
  }

  async function handleRefreshJobs(status = retryStatusFilter) {
    setIsRefreshingJobs(true)
    setError('')
    try {
      await loadRetryJobs(status)
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载重试作业失败')
    } finally {
      setIsRefreshingJobs(false)
    }
  }

  async function handleRetryVisibleJobs() {
    const retryableIDs = retryJobs.filter((job) => job.status !== 'pending').map((job) => job.id)
    if (retryableIDs.length === 0) {
      return
    }
    setIsRefreshingJobs(true)
    setError('')
    try {
      await retryAIJobs(retryableIDs)
      await loadRetryJobs(retryStatusFilter)
    } catch (err) {
      setError(err instanceof Error ? err.message : '批量重试提交失败')
    } finally {
      setIsRefreshingJobs(false)
    }
  }

  async function handleCleanupJobs(statuses: string[]) {
    setIsRefreshingJobs(true)
    setError('')
    try {
      await cleanupAIRetryJobs(statuses)
      if (selectedRetryJob && statuses.includes(selectedRetryJob.status)) {
        setSelectedRetryJob(null)
      }
      await loadRetryJobs(retryStatusFilter)
    } catch (err) {
      setError(err instanceof Error ? err.message : '清理重试作业失败')
    } finally {
      setIsRefreshingJobs(false)
    }
  }

  async function handleSelectRetryJob(id: number) {
    setIsLoadingRetryDetail(true)
    setError('')
    try {
      setSelectedRetryJob(await getAIRetryJobDetail(id))
    } catch (err) {
      setError(err instanceof Error ? err.message : '加载重试作业详情失败')
    } finally {
      setIsLoadingRetryDetail(false)
    }
  }

  if (isLoading || !config) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  return (
    <main className="min-h-screen bg-background p-6">
      <div className="mx-auto max-w-5xl space-y-6">
        <div>
          <Link href="/admin" className="text-sm text-muted-foreground hover:text-foreground">
            返回后台
          </Link>
          <h1 className="mt-2 text-3xl font-semibold">AI 配置</h1>
          <p className="mt-2 text-muted-foreground">所有 AI 相关设置统一写入 `config/ai.yaml`。</p>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Bot 基础设置</CardTitle>
            <CardDescription>切换默认 Provider、模型、Prompt 和上下文长度。</CardDescription>
          </CardHeader>
          <CardContent>
            <FieldGroup>
              <Field>
                <FieldLabel>启用 Bot</FieldLabel>
                <Switch checked={config.bot.enabled} onCheckedChange={(checked) => setConfig({ ...config, bot: { ...config.bot, enabled: checked } })} />
              </Field>
              <Field>
                <FieldLabel>Bot 用户名</FieldLabel>
                <Input value={config.bot.username} onChange={(e) => setConfig({ ...config, bot: { ...config.bot, username: e.target.value } })} />
              </Field>
              <Field>
                <FieldLabel>Bot 显示名</FieldLabel>
                <Input value={config.bot.display_name} onChange={(e) => setConfig({ ...config, bot: { ...config.bot, display_name: e.target.value } })} />
              </Field>
              <Field>
                <FieldLabel>默认 Provider</FieldLabel>
                <Input value={config.bot.default_provider} onChange={(e) => setConfig({ ...config, bot: { ...config.bot, default_provider: e.target.value } })} />
              </Field>
              <Field>
                <FieldLabel>默认模型</FieldLabel>
                <Input value={config.bot.default_model} onChange={(e) => setConfig({ ...config, bot: { ...config.bot, default_model: e.target.value } })} />
              </Field>
              <Field>
                <FieldLabel>上下文消息条数</FieldLabel>
                <Input type="number" value={config.bot.context_messages} onChange={(e) => setConfig({ ...config, bot: { ...config.bot, context_messages: Number(e.target.value) } })} />
              </Field>
              <Field>
                <FieldLabel>异步超时（秒）</FieldLabel>
                <Input type="number" value={config.bot.async_timeout_seconds} onChange={(e) => setConfig({ ...config, bot: { ...config.bot, async_timeout_seconds: Number(e.target.value) } })} />
              </Field>
              <Field>
                <FieldLabel>Bot Prompt</FieldLabel>
                <Textarea value={config.bot.system_prompt} onChange={(e) => setConfig({ ...config, bot: { ...config.bot, system_prompt: e.target.value } })} rows={8} />
              </Field>
            </FieldGroup>
          </CardContent>
        </Card>

        <div className="grid gap-4 md:grid-cols-3">
          {(['zhipu', 'openai', 'anthropic'] as const).map((providerKey) => {
            const provider = config.providers[providerKey]
            return (
              <Card key={providerKey}>
                <CardHeader>
                  <CardTitle className="capitalize">{providerKey}</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <Field>
                    <FieldLabel>启用</FieldLabel>
                    <Switch checked={provider.enabled} onCheckedChange={(checked) => setConfig({
                      ...config,
                      providers: {
                        ...config.providers,
                        [providerKey]: { ...provider, enabled: checked },
                      },
                    })} />
                  </Field>
                  <Field>
                    <FieldLabel>Base URL</FieldLabel>
                    <Input value={provider.base_url} onChange={(e) => setConfig({
                      ...config,
                      providers: {
                        ...config.providers,
                        [providerKey]: { ...provider, base_url: e.target.value },
                      },
                    })} />
                  </Field>
                  <Field>
                    <FieldLabel>API Key</FieldLabel>
                    <Input value={provider.api_key} onChange={(e) => setConfig({
                      ...config,
                      providers: {
                        ...config.providers,
                        [providerKey]: { ...provider, api_key: e.target.value },
                      },
                    })} />
                  </Field>
                  <Field>
                    <FieldLabel>可用模型（逗号分隔）</FieldLabel>
                    <Textarea value={provider.models.join(', ')} onChange={(e) => setConfig({
                      ...config,
                      providers: {
                        ...config.providers,
                        [providerKey]: {
                          ...provider,
                          models: e.target.value.split(',').map((item) => item.trim()).filter(Boolean),
                        },
                      },
                    })} rows={4} />
                  </Field>
                </CardContent>
              </Card>
            )
          })}
        </div>

        <Card>
          <CardHeader>
            <CardTitle>审计配置</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <Field>
              <FieldLabel>启用 AI 审计</FieldLabel>
              <Switch checked={config.audit.enabled} onCheckedChange={(checked) => setConfig({ ...config, audit: { ...config.audit, enabled: checked } })} />
            </Field>
            <Field>
              <FieldLabel>保留天数</FieldLabel>
              <Input type="number" value={config.audit.retention_days} onChange={(e) => setConfig({ ...config, audit: { ...config.audit, retention_days: Number(e.target.value) } })} />
            </Field>
            {error ? <div className="rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</div> : null}
            <Button onClick={() => void handleSave()} className="bg-wechat-green hover:bg-wechat-green-dark" disabled={isSaving}>
              {isSaving ? '保存中...' : '保存配置'}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>AI 重试作业</CardTitle>
            <CardDescription>查看异步回复补偿队列，并手动把失败任务重新投入执行。</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
              <div className="flex flex-wrap gap-2">
                {(['all', 'pending', 'completed', 'exhausted'] as const).map((status) => (
                  <Button
                    key={status}
                    variant={retryStatusFilter === status ? 'default' : 'outline'}
                    className={retryStatusFilter === status ? 'bg-wechat-green hover:bg-wechat-green-dark' : ''}
                    onClick={() => {
                      setRetryStatusFilter(status)
                      void handleRefreshJobs(status)
                    }}
                    disabled={isRefreshingJobs}
                  >
                    {status === 'all' ? '全部' : status}
                  </Button>
                ))}
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => void handleCleanupJobs(['completed', 'exhausted'])}
                  disabled={isRefreshingJobs}
                >
                  清理已完成/已耗尽
                </Button>
                <Button
                  variant="outline"
                  onClick={() => void handleRetryVisibleJobs()}
                  disabled={isRefreshingJobs || retryJobs.every((job) => job.status === 'pending')}
                >
                  批量重试当前列表
                </Button>
                <Button variant="outline" onClick={() => void handleRefreshJobs()} disabled={isRefreshingJobs}>
                  {isRefreshingJobs ? '刷新中...' : '刷新'}
                </Button>
              </div>
            </div>
            <div className="text-sm text-muted-foreground">
              当前展示最近 20 条作业，筛选状态：{retryStatusFilter === 'all' ? '全部' : retryStatusFilter}
            </div>
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead className="text-left text-muted-foreground">
                  <tr>
                    <th className="px-2 py-2">ID</th>
                    <th className="px-2 py-2">状态</th>
                    <th className="px-2 py-2">用户</th>
                    <th className="px-2 py-2">会话</th>
                    <th className="px-2 py-2">尝试次数</th>
                    <th className="px-2 py-2">下次执行</th>
                    <th className="px-2 py-2">最近错误</th>
                    <th className="px-2 py-2 text-right">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {retryJobs.map((job) => (
                    <tr key={job.id} className="border-t border-border">
                      <td className="px-2 py-2 font-mono">{job.id}</td>
                      <td className="px-2 py-2">{job.status}</td>
                      <td className="px-2 py-2">{job.user_id}</td>
                      <td className="px-2 py-2">{job.conversation_id}</td>
                      <td className="px-2 py-2">{job.attempt_count}</td>
                      <td className="px-2 py-2">{new Date(job.next_attempt_at).toLocaleString()}</td>
                      <td className="max-w-xs px-2 py-2 text-muted-foreground">{job.last_error || '-'}</td>
                      <td className="px-2 py-2">
                        <div className="flex justify-end gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => void handleSelectRetryJob(job.id)}
                            disabled={isLoadingRetryDetail}
                          >
                            详情
                          </Button>
                          <Button variant="outline" size="sm" asChild>
                            <Link href={`/admin/conversations/${job.conversation_id}`}>会话诊断</Link>
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => void handleRetryJob(job.id)}
                            disabled={isRefreshingJobs}
                          >
                            立即重试
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                  {retryJobs.length === 0 ? (
                    <tr>
                      <td colSpan={8} className="px-2 py-6 text-center text-muted-foreground">
                        当前没有待观察的重试作业
                      </td>
                    </tr>
                  ) : null}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>

        {selectedRetryJob ? (
          <Card>
            <CardHeader>
              <CardTitle>重试作业详情</CardTitle>
              <CardDescription>查看补偿任务的最新状态、错误原因和关联会话。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 md:grid-cols-4">
                <Stat label="作业 ID" value={String(selectedRetryJob.id)} />
                <Stat label="状态" value={selectedRetryJob.status} />
                <Stat label="尝试次数" value={String(selectedRetryJob.attempt_count)} />
                <Stat label="下次执行" value={new Date(selectedRetryJob.next_attempt_at).toLocaleString()} />
                <Stat label="用户 ID" value={String(selectedRetryJob.user_id)} />
                <Stat label="会话 ID" value={String(selectedRetryJob.conversation_id)} />
                <Stat label="创建时间" value={new Date(selectedRetryJob.created_at).toLocaleString()} />
                <Stat label="更新时间" value={new Date(selectedRetryJob.updated_at).toLocaleString()} />
              </div>
              <div className="rounded-xl border border-border/60 bg-white/70 p-4">
                <p className="text-xs text-muted-foreground">最近错误</p>
                <pre className="mt-2 whitespace-pre-wrap break-all text-sm">{selectedRetryJob.last_error || '暂无错误信息'}</pre>
              </div>
              <div className="flex gap-2">
                <Button variant="outline" asChild>
                  <Link href={`/admin/conversations/${selectedRetryJob.conversation_id}`}>查看会话诊断</Link>
                </Button>
                <Button variant="outline" asChild>
                  <Link href={`/admin/audit?user_id=${selectedRetryJob.user_id}&conversation_id=${selectedRetryJob.conversation_id}`}>查看关联审计</Link>
                </Button>
                <Button
                  variant="outline"
                  onClick={() => void handleRetryJob(selectedRetryJob.id)}
                  disabled={isRefreshingJobs}
                >
                  重新执行当前作业
                </Button>
              </div>
            </CardContent>
          </Card>
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
