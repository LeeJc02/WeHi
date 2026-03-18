'use client'

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from 'react'
import {
  ApiRequestError,
  clearTokens,
  getCurrentUser,
  initTokens,
  login as apiLogin,
  logout as apiLogout,
  register as apiRegister,
} from './api'
import type { User } from './types'

interface AuthContextType {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
  login: (username: string, password: string) => Promise<void>
  register: (username: string, displayName: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    let cancelled = false

    async function bootstrap() {
      initTokens()
      try {
        const profile = await getCurrentUser()
        if (!cancelled) {
          setUser(profile)
        }
      } catch (error) {
        if (error instanceof ApiRequestError && error.status === 401) {
          clearTokens()
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false)
        }
      }
    }

    void bootstrap()

    return () => {
      cancelled = true
    }
  }, [])

  const login = useCallback(async (username: string, password: string) => {
    const result = await apiLogin({ username, password })
    setUser(result.user)
  }, [])

  const register = useCallback(async (username: string, displayName: string, password: string) => {
    await apiRegister({ username, display_name: displayName, password })
    const result = await apiLogin({ username, password })
    setUser(result.user)
  }, [])

  const logout = useCallback(async () => {
    try {
      await apiLogout()
    } finally {
      clearTokens()
      setUser(null)
    }
  }, [])

  return (
    <AuthContext.Provider
      value={{
        user,
        isLoading,
        isAuthenticated: !!user,
        login,
        register,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
