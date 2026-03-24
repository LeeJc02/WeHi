'use client'

import { useState } from 'react'
import { cn } from '@/lib/utils'

interface InteractiveLoginSceneProps {
  activeTab: 'login' | 'register'
  focusedField: string | null
  passwordLength: number
  showPassword: boolean
}

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max)
}

export function InteractiveLoginScene({
  activeTab,
  focusedField,
  passwordLength,
  showPassword,
}: InteractiveLoginSceneProps) {
  const [pointer, setPointer] = useState({ x: 0, y: 0 })

  const focusMode = focusedField?.includes('password') ? 'password' : focusedField ? 'text' : 'idle'

  const lookX = clamp((pointer.x - 0.5) * 14, -8, 8)
  const lookY = clamp((pointer.y - 0.5) * 10, -6, 6)
  const peekOffset = showPassword && passwordLength > 0 ? 6 : 0

  return (
    <div
      className="auth-scene relative overflow-hidden rounded-[28px] border border-black/5 p-6 shadow-[inset_0_1px_0_rgba(255,255,255,0.55)]"
      onMouseMove={(event) => {
        const rect = event.currentTarget.getBoundingClientRect()
        setPointer({
          x: (event.clientX - rect.left) / rect.width,
          y: (event.clientY - rect.top) / rect.height,
        })
      }}
      onMouseLeave={() => setPointer({ x: 0.5, y: 0.5 })}
    >
      <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_18%_18%,rgba(255,255,255,0.28),transparent_20%),radial-gradient(circle_at_82%_20%,rgba(134,239,172,0.26),transparent_24%),radial-gradient(circle_at_50%_100%,rgba(255,255,255,0.12),transparent_28%)]" />
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(140deg,rgba(255,255,255,0.08),transparent_42%,rgba(15,23,42,0.08))]" />
      <div className="pointer-events-none absolute inset-0 auth-grid opacity-40" />

      <div className="relative z-10 flex min-h-[320px] flex-col justify-between">
        <div className="flex items-center justify-between">
          <div className="rounded-full border border-black/6 bg-white/72 px-3 py-1 text-[11px] font-medium tracking-[0.18em] text-slate-600">
            WeHi
          </div>
          <div className="rounded-full bg-white/56 px-3 py-1 text-[11px] font-medium text-slate-600 backdrop-blur-sm">
            {focusMode === 'password' ? '密码聚焦' : focusMode === 'text' ? '输入中' : '待输入'}
          </div>
        </div>

        <div className="relative mx-auto h-[250px] w-full max-w-[420px]">
          <div className="absolute inset-x-10 bottom-0 h-24 rounded-full bg-black/12 blur-2xl" />

          <div
            className={cn(
              'auth-character absolute bottom-0 left-[62px] h-[214px] w-[120px] rounded-[28px_28px_16px_16px] bg-[#1f2937] shadow-[0_18px_28px_rgba(15,23,42,0.16)]',
              focusMode === 'password' ? 'translate-x-5 -rotate-2' : '-rotate-1',
            )}
          >
            <div className="absolute left-6 top-10 flex gap-5">
              <EyeBall x={lookX - 1 + peekOffset} y={lookY - 1 + peekOffset} />
              <EyeBall x={lookX - 1 + peekOffset} y={lookY - 1 + peekOffset} />
            </div>
          </div>

          <div
            className={cn(
              'auth-character absolute bottom-0 left-[6px] h-[154px] w-[168px] rounded-[84px_84px_20px_20px] bg-[#d9b98f] shadow-[0_18px_28px_rgba(180,142,93,0.16)]',
              focusMode === 'text' ? '-translate-x-1 rotate-1' : '',
            )}
          >
            <div className="absolute left-[44px] top-[56px] flex gap-7">
              <DotEye x={lookX} y={lookY} />
              <DotEye x={lookX} y={lookY} />
            </div>
          </div>

          <div
            className={cn(
              'auth-character absolute bottom-0 right-[74px] h-[246px] w-[144px] rounded-[74px_74px_18px_18px] bg-[#78c4b0] shadow-[0_18px_28px_rgba(16,185,129,0.18)]',
              showPassword && passwordLength > 0 ? '-translate-x-4 rotate-[4deg]' : 'rotate-[1deg]',
            )}
          >
            <div className="absolute left-[38px] top-[54px] flex gap-6">
              <EyeBall x={lookX + 2 - peekOffset} y={lookY - 1} />
              <EyeBall x={lookX + 2 - peekOffset} y={lookY - 1} />
            </div>
          </div>

          <div
            className={cn(
              'auth-character absolute bottom-0 right-[6px] h-[178px] w-[120px] rounded-[60px_60px_16px_16px] bg-[#b3d4a0] shadow-[0_18px_28px_rgba(22,163,74,0.14)]',
              focusMode === 'password' ? 'translate-x-2 -rotate-[3deg]' : '',
            )}
          >
            <div className="absolute left-[30px] top-[42px] flex gap-5">
              <DotEye x={lookX - 1 - peekOffset} y={lookY} />
              <DotEye x={lookX - 1 - peekOffset} y={lookY} />
            </div>
            <div
              className={cn(
                'absolute left-[26px] top-[92px] h-1 rounded-full bg-slate-900/72 transition-all duration-300',
                showPassword && passwordLength > 0 ? 'w-16' : 'w-12',
              )}
            />
          </div>
        </div>

        <div className="flex items-center justify-between rounded-2xl border border-black/5 bg-white/58 px-4 py-3 text-xs text-slate-600 backdrop-blur-md">
          <span>{showPassword ? '密码可见' : '密码隐藏'}</span>
          <span>{passwordLength > 0 ? `${passwordLength} 位输入` : '等待输入'}</span>
        </div>
      </div>
    </div>
  )
}

function EyeBall({ x, y }: { x: number; y: number }) {
  return (
    <div className="flex h-[18px] w-[18px] items-center justify-center rounded-full bg-white shadow-[inset_0_-2px_0_rgba(15,23,42,0.08)]">
      <div
        className="h-[7px] w-[7px] rounded-full bg-slate-900 transition-transform duration-150"
        style={{ transform: `translate(${x}px, ${y}px)` }}
      />
    </div>
  )
}

function DotEye({ x, y }: { x: number; y: number }) {
  return (
    <div
      className="h-3 w-3 rounded-full bg-slate-900 transition-transform duration-150"
      style={{ transform: `translate(${x}px, ${y}px)` }}
    />
  )
}
