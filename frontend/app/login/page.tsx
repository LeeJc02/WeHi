'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/lib/auth-context'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { FieldGroup, Field, FieldLabel } from '@/components/ui/field'
import { Spinner } from '@/components/ui/spinner'
import { MessageCircle } from 'lucide-react'

export default function LoginPage() {
  const router = useRouter()
  const { login, register, isAuthenticated, isLoading: authLoading } = useAuth()
  const [activeTab, setActiveTab] = useState<'login' | 'register'>('login')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  // 登录表单状态
  const [loginUsername, setLoginUsername] = useState('')
  const [loginPassword, setLoginPassword] = useState('')

  // 注册表单状态
  const [registerUsername, setRegisterUsername] = useState('')
  const [registerDisplayName, setRegisterDisplayName] = useState('')
  const [registerPassword, setRegisterPassword] = useState('')
  const [registerConfirmPassword, setRegisterConfirmPassword] = useState('')

  useEffect(() => {
    if (!authLoading && isAuthenticated) {
      router.replace('/')
    }
  }, [authLoading, isAuthenticated, router])

  async function handleLogin(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setIsLoading(true)

    try {
      await login(loginUsername, loginPassword)
      router.push('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请检查用户名和密码')
    } finally {
      setIsLoading(false)
    }
  }

  async function handleRegister(e: React.FormEvent) {
    e.preventDefault()
    setError('')

    if (registerPassword !== registerConfirmPassword) {
      setError('两次输入的密码不一致')
      return
    }

    if (registerPassword.length < 6) {
      setError('密码长度至少6位')
      return
    }

    setIsLoading(true)

    try {
      await register(registerUsername, registerDisplayName, registerPassword)
      router.push('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : '注册失败，请稍后再试')
    } finally {
      setIsLoading(false)
    }
  }

  if (authLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-wechat-green/5 via-background to-wechat-green/10">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-wechat-green/5 via-background to-wechat-green/10 p-4">
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="flex flex-col items-center mb-8">
          <div className="w-20 h-20 rounded-2xl bg-wechat-green flex items-center justify-center mb-4 shadow-lg shadow-wechat-green/30">
            <MessageCircle className="w-12 h-12 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-foreground">WeChat Web</h1>
          <p className="text-muted-foreground mt-1">连接你我，沟通世界</p>
        </div>

        <Card className="border-0 shadow-xl">
          <CardHeader className="space-y-1 pb-4">
            <CardTitle className="text-xl text-center">欢迎使用</CardTitle>
            <CardDescription className="text-center">
              登录或注册以开始使用
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'login' | 'register')}>
              <TabsList className="grid w-full grid-cols-2 mb-6">
                <TabsTrigger value="login">登录</TabsTrigger>
                <TabsTrigger value="register">注册</TabsTrigger>
              </TabsList>

              {error && (
                <div className="mb-4 p-3 rounded-lg bg-destructive/10 text-destructive text-sm">
                  {error}
                </div>
              )}

              <TabsContent value="login" className="mt-0">
                <form onSubmit={handleLogin}>
                  <FieldGroup>
                    <Field>
                      <FieldLabel>用户名</FieldLabel>
                      <Input
                        type="text"
                        placeholder="请输入用户名"
                        value={loginUsername}
                        onChange={(e) => setLoginUsername(e.target.value)}
                        required
                        autoComplete="username"
                      />
                    </Field>
                    <Field>
                      <FieldLabel>密码</FieldLabel>
                      <Input
                        type="password"
                        placeholder="请输入密码"
                        value={loginPassword}
                        onChange={(e) => setLoginPassword(e.target.value)}
                        required
                        autoComplete="current-password"
                      />
                    </Field>
                  </FieldGroup>
                  <Button
                    type="submit"
                    className="w-full mt-6 bg-wechat-green hover:bg-wechat-green-dark"
                    disabled={isLoading}
                  >
                    {isLoading ? <Spinner className="mr-2" /> : null}
                    登录
                  </Button>
                </form>
              </TabsContent>

              <TabsContent value="register" className="mt-0">
                <form onSubmit={handleRegister}>
                  <FieldGroup>
                    <Field>
                      <FieldLabel>用户名</FieldLabel>
                      <Input
                        type="text"
                        placeholder="设置登录用户名"
                        value={registerUsername}
                        onChange={(e) => setRegisterUsername(e.target.value)}
                        required
                        autoComplete="username"
                      />
                    </Field>
                    <Field>
                      <FieldLabel>昵称</FieldLabel>
                      <Input
                        type="text"
                        placeholder="设置显示昵称"
                        value={registerDisplayName}
                        onChange={(e) => setRegisterDisplayName(e.target.value)}
                        required
                      />
                    </Field>
                    <Field>
                      <FieldLabel>密码</FieldLabel>
                      <Input
                        type="password"
                        placeholder="设置密码（至少6位）"
                        value={registerPassword}
                        onChange={(e) => setRegisterPassword(e.target.value)}
                        required
                        autoComplete="new-password"
                      />
                    </Field>
                    <Field>
                      <FieldLabel>确认密码</FieldLabel>
                      <Input
                        type="password"
                        placeholder="再次输入密码"
                        value={registerConfirmPassword}
                        onChange={(e) => setRegisterConfirmPassword(e.target.value)}
                        required
                        autoComplete="new-password"
                      />
                    </Field>
                  </FieldGroup>
                  <Button
                    type="submit"
                    className="w-full mt-6 bg-wechat-green hover:bg-wechat-green-dark"
                    disabled={isLoading}
                  >
                    {isLoading ? <Spinner className="mr-2" /> : null}
                    注册
                  </Button>
                </form>
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>

        <p className="text-center text-muted-foreground text-sm mt-6">
          继续即表示您同意我们的服务条款和隐私政策
        </p>
      </div>
    </div>
  )
}
