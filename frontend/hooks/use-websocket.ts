'use client'

import { useEffect, useRef, useState, useCallback } from 'react'
import type { WsEvent } from '@/lib/types'
import { getWebSocketUrl, getAccessToken } from '@/lib/api'

type WsEventHandler = (event: WsEvent) => void

interface UseWebSocketReturn {
  isConnected: boolean
  lastEvent: WsEvent | null
  subscribe: (handler: WsEventHandler) => () => void
}

export function useWebSocket(enabled: boolean = true): UseWebSocketReturn {
  const wsRef = useRef<WebSocket | null>(null)
  const handlersRef = useRef<Set<WsEventHandler>>(new Set())
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [lastEvent, setLastEvent] = useState<WsEvent | null>(null)

  const connect = useCallback(() => {
    if (!enabled) return
    const token = getAccessToken()
    if (!token) return

    // 清除之前的重连定时器
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }

    // 关闭现有连接
    if (wsRef.current) {
      wsRef.current.close()
    }

    const wsUrl = getWebSocketUrl()
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      setIsConnected(true)
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as WsEvent
        setLastEvent(data)
        handlersRef.current.forEach((handler) => handler(data))
      } catch {
        // 忽略解析错误
      }
    }

    ws.onerror = () => {
      // WebSocket 错误
    }

    ws.onclose = () => {
      setIsConnected(false)
      // 5秒后重连
      reconnectTimeoutRef.current = setTimeout(() => {
        if (getAccessToken()) {
          connect()
        }
      }, 5000)
    }

    wsRef.current = ws
  }, [enabled])

  useEffect(() => {
    if (!enabled) {
      setIsConnected(false)
      return
    }
    
    const token = getAccessToken()
    if (token) {
      connect()
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [connect, enabled])

  const subscribe = useCallback((handler: WsEventHandler) => {
    handlersRef.current.add(handler)
    return () => {
      handlersRef.current.delete(handler)
    }
  }, [])

  return {
    isConnected,
    lastEvent,
    subscribe,
  }
}
