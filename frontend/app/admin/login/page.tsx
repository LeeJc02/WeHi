'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { adminLogin, getAdminMe, initAdminToken } from '@/lib/admin-api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { Shield } from 'lucide-react'

export default function AdminLoginPage() {
  const router = useRouter()
  const [username, setUsername] = useState('root')
  const [password, setPassword] = useState('123456')
  const [isLoading, setIsLoading] = useState(false)
  const [isBooting, setIsBooting] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    void (async () => {
      initAdminToken()
      try {
        const profile = await getAdminMe()
        router.replace(profile.must_change_password ? '/admin/force-change-password' : '/admin')
      } catch {
        setIsBooting(false)
      }
    })()
  }, [router])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setIsLoading(true)
    try {
      const result = await adminLogin({ username, password })
      router.replace(result.admin.must_change_password ? '/admin/force-change-password' : '/admin')
    } catch (err) {
      setError(err instanceof Error ? err.message : '管理端登录失败')
    } finally {
      setIsLoading(false)
    }
  }

  if (isBooting) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[radial-gradient(circle_at_top,_rgba(7,193,96,0.12),_transparent_55%),linear-gradient(135deg,#f6fbf7,#eef4f2)]">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-[radial-gradient(circle_at_top,_rgba(7,193,96,0.12),_transparent_55%),linear-gradient(135deg,#f6fbf7,#eef4f2)] p-4">
      <Card className="w-full max-w-md border-0 shadow-2xl">
        <CardHeader className="space-y-3">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-wechat-green text-white">
            <Shield className="h-7 w-7" />
          </div>
          <CardTitle>管理后台登录</CardTitle>
          <CardDescription>默认账户为 `root / 123456`，首次登录后必须修改密码。</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel>管理员账号</FieldLabel>
                <Input value={username} onChange={(e) => setUsername(e.target.value)} required />
              </Field>
              <Field>
                <FieldLabel>密码</FieldLabel>
                <Input type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
              </Field>
            </FieldGroup>
            {error ? (
              <div className="mt-4 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</div>
            ) : null}
            <Button type="submit" className="mt-6 w-full bg-wechat-green hover:bg-wechat-green-dark" disabled={isLoading}>
              {isLoading ? <Spinner className="mr-2" /> : null}
              登录
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
