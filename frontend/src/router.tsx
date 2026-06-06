
import { Routes, Route, Navigate } from 'react-router-dom'
import { LoginPage } from './features/auth/LoginPage'
import { DashboardPage } from './features/dashboard/DashboardPage'
import { CustomersPage } from './features/customers/CustomersPage'
import { CompaniesPage } from './features/companies/CompaniesPage'
import { EKYCPage } from './features/ekyc/EKYCPage'
import { EKYBPage } from './features/ekyb/EKYBPage'
import { AppShell } from './components/layout/AppShell'
import { ProtectedRoute } from './auth/ProtectedRoute'

export function AppRouter() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <AppShell />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="customers" element={<CustomersPage />} />
        <Route path="companies" element={<CompaniesPage />} />
        <Route path="ekyc" element={<EKYCPage />} />
        <Route path="ekyb" element={<EKYBPage />} />
      </Route>
    </Routes>
  )
}
