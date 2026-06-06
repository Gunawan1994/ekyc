import { useContext } from 'react'
import { AuthContext } from './AuthContext'

/**
 * Consumes AuthContext. Must be used inside <AuthProvider>.
 * Throws if called outside the provider tree.
 */
export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return ctx
}
