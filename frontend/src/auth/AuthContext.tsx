import {
  createContext,
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react'
import api from '../lib/axios'
import type { AuthTokens, User } from '../types'

interface AuthContextValue {
  user: User | null
  tokens: AuthTokens | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthContextValue | null>(null)

const TOKEN_KEYS = {
  access: 'access_token',
  refresh: 'refresh_token',
} as const

function readTokens(): AuthTokens | null {
  const accessToken = localStorage.getItem(TOKEN_KEYS.access)
  const refreshToken = localStorage.getItem(TOKEN_KEYS.refresh)
  if (accessToken && refreshToken) return { accessToken, refreshToken }
  return null
}

function writeTokens(tokens: AuthTokens): void {
  localStorage.setItem(TOKEN_KEYS.access, tokens.accessToken)
  localStorage.setItem(TOKEN_KEYS.refresh, tokens.refreshToken)
}

function clearTokens(): void {
  localStorage.removeItem(TOKEN_KEYS.access)
  localStorage.removeItem(TOKEN_KEYS.refresh)
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [tokens, setTokens] = useState<AuthTokens | null>(readTokens)
  const [isLoading, setIsLoading] = useState<boolean>(!!readTokens())

  // On mount, if persisted tokens exist, fetch the current user profile.
  useEffect(() => {
    if (!tokens) {
      setIsLoading(false)
      return
    }
    let cancelled = false
    ;(async () => {
      try {
        const res = await api.get<{ data: User }>('/auth/me')
        if (!cancelled) setUser(res.data.data)
      } catch {
        // Tokens invalid or expired beyond refresh — clear state.
        if (!cancelled) {
          clearTokens()
          setTokens(null)
          setUser(null)
        }
      } finally {
        if (!cancelled) setIsLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
    // Run only once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const res = await api.post<{
      data: { user: User; accessToken: string; refreshToken: string }
    }>('/auth/login', { email, password })
    const { user: fetchedUser, accessToken, refreshToken } = res.data.data
    const newTokens: AuthTokens = { accessToken, refreshToken }
    writeTokens(newTokens)
    setTokens(newTokens)
    setUser(fetchedUser)
  }, [])

  const logout = useCallback(async () => {
    try {
      await api.post('/auth/logout')
    } catch {
      // Best-effort — proceed with local cleanup regardless.
    } finally {
      clearTokens()
      setTokens(null)
      setUser(null)
    }
  }, [])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      tokens,
      isAuthenticated: !!user,
      isLoading,
      login,
      logout,
    }),
    [user, tokens, isLoading, login, logout]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
