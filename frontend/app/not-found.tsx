'use client'

import { useEffect } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { Spinner } from '@/components/ui/spinner'
import { useAuth } from '@/lib/auth-context'

export default function NotFound() {
  const router = useRouter()
  const { isAuthenticated, isLoading } = useAuth()

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace('/login')
    }
  }, [isAuthenticated, isLoading, router])

  if (isLoading || !isAuthenticated) {
    return (
      <div className="h-screen w-full flex items-center justify-center bg-background">
        <Spinner className="h-8 w-8 text-wechat-green" />
      </div>
    )
  }

  return (
    <div className="h-screen w-full flex flex-col items-center justify-center gap-4 bg-background">
      <div className="text-center">
        <h1 className="text-3xl font-semibold text-foreground">页面不存在</h1>
        <p className="mt-2 text-sm text-muted-foreground">你访问的地址不存在或没有权限访问。</p>
      </div>
      <Link
        href="/"
        className="rounded-md bg-wechat-green px-4 py-2 text-sm font-medium text-white transition hover:bg-wechat-green-dark"
      >
        返回工作台
      </Link>
    </div>
  )
}
