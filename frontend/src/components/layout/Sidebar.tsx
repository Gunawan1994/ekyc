
import { NavLink } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  LayoutDashboard,
  Users,
  Building2,
  ShieldCheck,
  ShieldAlert,
  Bell,
} from 'lucide-react'
import { dashboardApi } from '../../api/dashboard'

interface NavItem {
  label: string
  to: string
  icon: React.ReactNode
  badge?: number
}

export function Sidebar() {
  const { data: stats } = useQuery({
    queryKey: ['dashboard', 'stats'],
    queryFn: () => dashboardApi.getStats().then(r => r.data.data),
    refetchInterval: 60_000,
    staleTime: 30_000,
  })

  const alertCount = (stats?.totalKycPending ?? 0) + (stats?.totalKybPending ?? 0)

  const navItems: NavItem[] = [
    { label: 'Dashboard',  to: '/dashboard',  icon: <LayoutDashboard size={18} /> },
    { label: 'Customers',  to: '/customers',  icon: <Users size={18} /> },
    { label: 'Companies',  to: '/companies',  icon: <Building2 size={18} /> },
    { label: 'eKYC',       to: '/ekyc',       icon: <ShieldCheck size={18} /> },
    { label: 'eKYB',       to: '/ekyb',       icon: <ShieldAlert size={18} /> },
    {
      label: 'Monitoring',
      to: '/monitoring',
      icon: <Bell size={18} />,
      badge: alertCount > 0 ? alertCount : undefined,
    },
  ]

  return (
    <aside className="flex flex-col w-64 shrink-0 h-screen bg-slate-800 text-slate-100">
      {/* Logo */}
      <div className="flex items-center gap-3 px-6 h-16 border-b border-slate-700 shrink-0">
        <div className="flex items-center justify-center w-8 h-8 rounded-lg bg-sky-500 text-white">
          <ShieldCheck size={18} strokeWidth={2.5} />
        </div>
        <span className="font-bold text-white tracking-tight text-base">
          eKYC Platform
        </span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-3 py-4 overflow-y-auto" aria-label="Main navigation">
        <ul role="list" className="flex flex-col gap-0.5">
          {navItems.map((item) => (
            <li key={item.to}>
              <NavLink
                to={item.to}
                className={({ isActive }) =>
                  [
                    'flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-sky-500 text-white'
                      : 'text-slate-300 hover:bg-slate-700 hover:text-white',
                  ].join(' ')
                }
              >
                <span className="shrink-0">{item.icon}</span>
                <span className="flex-1">{item.label}</span>
                {item.badge != null && (
                  <span className="ml-auto inline-flex items-center justify-center min-w-[1.25rem] h-5 px-1.5 rounded-full bg-red-500 text-white text-[10px] font-bold">
                    {item.badge > 99 ? '99+' : item.badge}
                  </span>
                )}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      {/* Footer */}
      <div className="px-6 py-4 border-t border-slate-700 text-xs text-slate-500">
        &copy; {new Date().getFullYear()} PT Sun Energy
      </div>
    </aside>
  )
}
