
import { useLocation } from 'react-router-dom'
import { LogOut, User } from 'lucide-react'
import { useAuth } from '../../auth/useAuth'

const routeTitles: Record<string, string> = {
  '/dashboard': 'Dashboard',
  '/customers': 'Customers',
  '/companies': 'Companies',
  '/ekyc': 'eKYC Verifications',
  '/ekyb': 'eKYB Verifications',
}

function getPageTitle(pathname: string): string {
  if (routeTitles[pathname]) return routeTitles[pathname]
  const match = Object.keys(routeTitles).find((key) => pathname.startsWith(key + '/'))
  return match ? routeTitles[match] : 'eKYC Platform'
}

export function Topbar() {
  const { pathname } = useLocation()
  const { user, logout } = useAuth()
  const title = getPageTitle(pathname)

  const handleLogout = () => {
    logout().catch(() => {
      // logout handles its own cleanup; ignore post-redirect errors
    })
  }

  return (
    <header className="flex items-center justify-between h-16 px-6 bg-white border-b border-slate-200 shrink-0">
      {/* Page title */}
      <h1 className="text-lg font-semibold text-slate-800">{title}</h1>

      {/* Right section */}
      <div className="flex items-center gap-4">
        {/* User info */}
        <div className="flex items-center gap-2 text-sm text-slate-600">
          <span className="flex items-center justify-center w-7 h-7 rounded-full bg-slate-100 text-slate-500">
            <User size={15} />
          </span>
          <span className="font-medium text-slate-700">
            {user?.fullName ?? user?.email ?? 'User'}
          </span>
        </div>

        {/* Divider */}
        <div className="w-px h-5 bg-slate-200" aria-hidden="true" />

        {/* Logout */}
        <button
          onClick={handleLogout}
          className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-slate-600
            rounded-lg hover:bg-slate-100 hover:text-slate-800 transition-colors
            focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
          aria-label="Log out"
        >
          <LogOut size={15} />
          Logout
        </button>
      </div>
    </header>
  )
}
