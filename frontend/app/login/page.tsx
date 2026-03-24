'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'
import { useAuth } from '@/lib/auth-context'
import { InteractiveLoginScene } from '@/components/auth/interactive-login-scene'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { FieldGroup, Field, FieldLabel } from '@/components/ui/field'
import { Spinner } from '@/components/ui/spinner'
import { Eye, EyeOff, LockKeyhole, MessageCircle, Sparkles, UserRound } from 'lucide-react'

export default function LoginPage() {
  const router = useRouter()
  const { login, register, isAuthenticated, isLoading: authLoading } = useAuth()
  const [activeTab, setActiveTab] = useState<'login' | 'register'>('login')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')
  const [focusedField, setFocusedField] = useState<string | null>(null)
  const [showLoginPassword, setShowLoginPassword] = useState(false)
  const [showRegisterPassword, setShowRegisterPassword] = useState(false)
  const [showRegisterConfirmPassword, setShowRegisterConfirmPassword] = useState(false)

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
      <div className="flex min-h-screen items-center justify-center bg-[radial-gradient(circle_at_top,#ecfdf5,transparent_32%),linear-gradient(160deg,#f7faf8,#edf7f2_55%,#e8f5ef)]">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  return (
    <div className="relative min-h-screen overflow-hidden bg-[radial-gradient(circle_at_top,#ecfdf5,transparent_22%),linear-gradient(160deg,#f7faf8,#edf7f2_55%,#e8f5ef)] px-4 py-6 md:px-6">
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_12%_14%,rgba(34,197,94,0.1),transparent_18%),radial-gradient(circle_at_88%_10%,rgba(125,211,252,0.16),transparent_20%),radial-gradient(circle_at_50%_100%,rgba(16,185,129,0.08),transparent_24%)]" />
      <div className="relative mx-auto flex min-h-[calc(100vh-3rem)] w-full max-w-5xl items-center justify-center">
        <div className="flex items-center justify-center">
          <Card className="wechat-login-card w-full max-w-5xl border-white/70 bg-white/82 shadow-[0_24px_80px_rgba(15,23,42,0.12)] backdrop-blur-2xl">
            <div className="grid xl:grid-cols-[0.92fr_1.08fr]">
              <div className="border-b border-black/5 p-5 xl:border-b-0 xl:border-r xl:p-6">
                <InteractiveLoginScene
                  activeTab={activeTab}
                  focusedField={focusedField}
                  passwordLength={
                    activeTab === 'login'
                      ? loginPassword.length
                      : Math.max(registerPassword.length, registerConfirmPassword.length)
                  }
                  showPassword={
                    activeTab === 'login'
                      ? showLoginPassword
                      : showRegisterPassword || showRegisterConfirmPassword
                  }
                />
              </div>

              <div className="p-2">
                <CardHeader className="space-y-4 pb-3">
                  <div className="flex items-center gap-3">
                    <div className="flex h-12 w-12 items-center justify-center rounded-[18px] bg-[linear-gradient(145deg,var(--color-wechat-green),#0f9d58)] shadow-[0_16px_32px_rgba(22,163,74,0.22)]">
                      <MessageCircle className="h-6 w-6 text-white" />
                    </div>
                    <div>
                      <p className="text-[11px] font-medium uppercase tracking-[0.22em] text-muted-foreground">WeHi</p>
                      <CardTitle className="mt-1 text-[26px] font-semibold tracking-[-0.04em]">
                        {activeTab === 'login' ? '登录' : '创建账号'}
                      </CardTitle>
                    </div>
                  </div>

                  <CardDescription className="max-w-md text-sm leading-6 text-muted-foreground">
                    {activeTab === 'login' ? '输入用户名与密码继续会话。' : '创建一个新账号并立即进入消息列表。'}
                  </CardDescription>
                </CardHeader>

                <CardContent>
                  <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as 'login' | 'register')}>
                    <TabsList className="mb-6 grid h-12 w-full grid-cols-2 rounded-2xl bg-black/[0.04] p-1">
                      <TabsTrigger value="login" className="rounded-xl data-[state=active]:bg-white data-[state=active]:shadow-sm">
                        登录
                      </TabsTrigger>
                      <TabsTrigger value="register" className="rounded-xl data-[state=active]:bg-white data-[state=active]:shadow-sm">
                        注册
                      </TabsTrigger>
                    </TabsList>

                    {error && (
                      <div className="mb-4 rounded-2xl border border-destructive/20 bg-destructive/8 px-4 py-3 text-sm text-destructive shadow-sm">
                        {error}
                      </div>
                    )}

                    <TabsContent value="login" className="mt-0">
                      <form onSubmit={handleLogin}>
                        <FieldGroup>
                          <Field>
                            <FieldLabel>用户名</FieldLabel>
                            <div className="relative">
                              <UserRound className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                              <Input
                                type="text"
                                placeholder="请输入用户名"
                                value={loginUsername}
                                onChange={(e) => setLoginUsername(e.target.value)}
                                onFocus={() => setFocusedField('login-username')}
                                onBlur={() => setFocusedField(null)}
                                required
                                autoComplete="username"
                                className="h-12 rounded-2xl border-black/6 bg-white/88 pl-11 pr-4"
                              />
                            </div>
                          </Field>
                          <Field>
                            <FieldLabel>密码</FieldLabel>
                            <div className="relative">
                              <LockKeyhole className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                              <Input
                                type={showLoginPassword ? 'text' : 'password'}
                                placeholder="请输入密码"
                                value={loginPassword}
                                onChange={(e) => setLoginPassword(e.target.value)}
                                onFocus={() => setFocusedField('login-password')}
                                onBlur={() => setFocusedField(null)}
                                required
                                autoComplete="current-password"
                                className="h-12 rounded-2xl border-black/6 bg-white/88 pl-11 pr-12"
                              />
                              <button
                                type="button"
                                className="absolute right-3 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-black/[0.05] hover:text-foreground"
                                onClick={() => setShowLoginPassword((value) => !value)}
                                aria-label={showLoginPassword ? '隐藏密码' : '显示密码'}
                              >
                                {showLoginPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                              </button>
                            </div>
                          </Field>
                        </FieldGroup>
                        <Button type="submit" className="mt-6 h-12 w-full rounded-2xl bg-wechat-green hover:bg-wechat-green-dark" disabled={isLoading}>
                          {isLoading ? <Spinner className="mr-2" /> : null}
                          进入 WeHi
                        </Button>
                      </form>
                    </TabsContent>

                    <TabsContent value="register" className="mt-0">
                      <form onSubmit={handleRegister}>
                        <FieldGroup>
                          <Field>
                            <FieldLabel>用户名</FieldLabel>
                            <div className="relative">
                              <UserRound className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                              <Input
                                type="text"
                                placeholder="设置登录用户名"
                                value={registerUsername}
                                onChange={(e) => setRegisterUsername(e.target.value)}
                                onFocus={() => setFocusedField('register-username')}
                                onBlur={() => setFocusedField(null)}
                                required
                                autoComplete="username"
                                className="h-12 rounded-2xl border-black/6 bg-white/88 pl-11 pr-4"
                              />
                            </div>
                          </Field>
                          <Field>
                            <FieldLabel>昵称</FieldLabel>
                            <div className="relative">
                              <Sparkles className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                              <Input
                                type="text"
                                placeholder="设置显示昵称"
                                value={registerDisplayName}
                                onChange={(e) => setRegisterDisplayName(e.target.value)}
                                onFocus={() => setFocusedField('register-display-name')}
                                onBlur={() => setFocusedField(null)}
                                required
                                className="h-12 rounded-2xl border-black/6 bg-white/88 pl-11 pr-4"
                              />
                            </div>
                          </Field>
                          <Field>
                            <FieldLabel>密码</FieldLabel>
                            <div className="relative">
                              <LockKeyhole className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                              <Input
                                type={showRegisterPassword ? 'text' : 'password'}
                                placeholder="设置密码（至少6位）"
                                value={registerPassword}
                                onChange={(e) => setRegisterPassword(e.target.value)}
                                onFocus={() => setFocusedField('register-password')}
                                onBlur={() => setFocusedField(null)}
                                required
                                autoComplete="new-password"
                                className="h-12 rounded-2xl border-black/6 bg-white/88 pl-11 pr-12"
                              />
                              <button
                                type="button"
                                className="absolute right-3 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-black/[0.05] hover:text-foreground"
                                onClick={() => setShowRegisterPassword((value) => !value)}
                                aria-label={showRegisterPassword ? '隐藏密码' : '显示密码'}
                              >
                                {showRegisterPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                              </button>
                            </div>
                          </Field>
                          <Field>
                            <FieldLabel>确认密码</FieldLabel>
                            <div className="relative">
                              <LockKeyhole className="pointer-events-none absolute left-4 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                              <Input
                                type={showRegisterConfirmPassword ? 'text' : 'password'}
                                placeholder="再次输入密码"
                                value={registerConfirmPassword}
                                onChange={(e) => setRegisterConfirmPassword(e.target.value)}
                                onFocus={() => setFocusedField('register-confirm-password')}
                                onBlur={() => setFocusedField(null)}
                                required
                                autoComplete="new-password"
                                className="h-12 rounded-2xl border-black/6 bg-white/88 pl-11 pr-12"
                              />
                              <button
                                type="button"
                                className="absolute right-3 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-full text-muted-foreground transition-colors hover:bg-black/[0.05] hover:text-foreground"
                                onClick={() => setShowRegisterConfirmPassword((value) => !value)}
                                aria-label={showRegisterConfirmPassword ? '隐藏密码' : '显示密码'}
                              >
                                {showRegisterConfirmPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                              </button>
                            </div>
                          </Field>
                        </FieldGroup>
                        <Button type="submit" className="mt-6 h-12 w-full rounded-2xl bg-wechat-green hover:bg-wechat-green-dark" disabled={isLoading}>
                          {isLoading ? <Spinner className="mr-2" /> : null}
                          创建并进入
                        </Button>
                      </form>
                    </TabsContent>
                  </Tabs>

                  <p className="mt-6 text-sm text-muted-foreground">继续即表示您同意我们的服务条款与隐私政策。</p>
                </CardContent>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
