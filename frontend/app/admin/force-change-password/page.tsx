'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { adminChangePassword, getAdminMe, initAdminToken } from '@/lib/admin-api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Field, FieldGroup, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'

export default function AdminForceChangePasswordPage() {
  const router = useRouter()
  const [currentPassword, setCurrentPassword] = useState('123456')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [error, setError] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [isBooting, setIsBooting] = useState(true)

  useEffect(() => {
    void (async () => {
      initAdminToken()
      try {
        const profile = await getAdminMe()
        if (!profile.must_change_password) {
          router.replace('/admin')
          return
        }
        setIsBooting(false)
      } catch {
        router.replace('/admin/login')
      }
    })()
  }, [router])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    if (newPassword !== confirmPassword) {
      setError('两次输入的新密码不一致')
      return
    }
    setIsLoading(true)
    try {
      await adminChangePassword({ current_password: currentPassword, new_password: newPassword })
      router.replace('/admin')
    } catch (err) {
      setError(err instanceof Error ? err.message : '修改密码失败')
    } finally {
      setIsLoading(false)
    }
  }

  if (isBooting) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-[linear-gradient(135deg,#fffef8,#f0f7f2)] p-4">
      <Card className="w-full max-w-md border-0 shadow-xl">
        <CardHeader>
          <CardTitle>首次登录需修改密码</CardTitle>
          <CardDescription>修改完成后才可进入管理后台。</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit}>
            <FieldGroup>
              <Field>
                <FieldLabel>当前密码</FieldLabel>
                <Input type="password" value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} required />
              </Field>
              <Field>
                <FieldLabel>新密码</FieldLabel>
                <Input type="password" value={newPassword} onChange={(e) => setNewPassword(e.target.value)} required />
              </Field>
              <Field>
                <FieldLabel>确认新密码</FieldLabel>
                <Input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} required />
              </Field>
            </FieldGroup>
            {error ? <div className="mt-4 rounded-lg bg-destructive/10 px-3 py-2 text-sm text-destructive">{error}</div> : null}
            <Button type="submit" className="mt-6 w-full bg-wechat-green hover:bg-wechat-green-dark" disabled={isLoading}>
              {isLoading ? <Spinner className="mr-2" /> : null}
              保存新密码
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
