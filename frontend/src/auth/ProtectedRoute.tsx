
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from './useAuth'

interface ProtectedRouteProps {
  children: React.ReactNode
  /** Optional: restrict access to specific roles */
  allowedRoles?: string[]
}

/**
 * Guards a route behind authentication.
 * - While the auth state is loading (token rehydration), renders nothing.
 * - Unauthenticated visitors are redirected to /login, preserving the
 *   intended destination in location state so the login page can redirect
 *   back after a successful sign-in.
 * - If allowedRoles is provided, users whose role is not in the list are
 *   redirected to / (dashboard).
 */
export function ProtectedRoute({
  children,
  allowedRoles,
}: ProtectedRouteProps) {
  const { isAuthenticated, isLoading, user } = useAuth()
  const location = useLocation()

  if (isLoading) {
    // Avoid a flash of the login page while tokens are being validated.
    return null
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  if (allowedRoles && user && !allowedRoles.includes(user.role)) {
    return <Navigate to="/dashboard" replace />
  }

  return <>{children}</>
}
