import { lazy, Suspense } from 'react'
import { Routes, Route, Navigate, useLocation } from 'react-router-dom'
import { AppShell } from './components/layout/AppShell'
import { useAuth } from './auth/useAuth'

// Lazy-load page-level components — bundle split per route.
const LoginPage = lazy(() => import('./features/auth/LoginPage'))
const DashboardPage = lazy(() => import('./features/dashboard/DashboardPage'))
const CustomersPage = lazy(() => import('./features/customers/CustomersPage'))
const CompaniesPage = lazy(() => import('./features/companies/CompaniesPage'))
const KYCPage = lazy(() => import('./features/ekyc/EKYCPage'))
const KYCDetailPage = lazy(() => import('./features/ekyc/EKYCDetailPage'))
const KYBPage = lazy(() => import('./features/ekyb/EKYBPage'))
const KYBDetailPage = lazy(() => import('./features/ekyb/EKYBDetailPage'))
const MonitoringPage = lazy(() => import('./features/monitoring/MonitoringPage'))
const UsersPage = lazy(() => import('./features/users/UsersPage'))
const NotFoundPage = lazy(() => import('./features/common/NotFoundPage'))

function PageLoader() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-sky-600 border-t-transparent" />
    </div>
  )
}

// Layout route: guards all nested routes, renders AppShell (which has <Outlet />).
function AuthLayout() {
  const { isAuthenticated, isLoading } = useAuth()
  const location = useLocation()

  if (isLoading) return null

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <AppShell />
}

// Role guard for individual routes inside the layout.
function RoleGuard({ allowedRoles, children }: { allowedRoles: string[]; children: React.ReactNode }) {
  const { user } = useAuth()
  if (user && !allowedRoles.includes(user.role)) {
    return <Navigate to="/dashboard" replace />
  }
  return <>{children}</>
}

export default function App() {
  return (
    <Suspense fallback={<PageLoader />}>
      <Routes>
        {/* Public */}
        <Route path="/login" element={<LoginPage />} />

        {/* Protected — all share AppShell layout */}
        <Route element={<AuthLayout />}>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/customers" element={<CustomersPage />} />
          <Route path="/companies" element={<CompaniesPage />} />
          <Route path="/ekyc" element={<KYCPage />} />
          <Route path="/ekyc/:id" element={<KYCDetailPage />} />
          <Route path="/ekyb" element={<KYBPage />} />
          <Route path="/ekyb/:id" element={<KYBDetailPage />} />
          <Route path="/monitoring" element={<MonitoringPage />} />
          <Route
            path="/users"
            element={
              <RoleGuard allowedRoles={['admin']}>
                <UsersPage />
              </RoleGuard>
            }
          />
        </Route>

        {/* Redirects */}
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/404" element={<NotFoundPage />} />
        <Route path="*" element={<Navigate to="/404" replace />} />
      </Routes>
    </Suspense>
  )
}
