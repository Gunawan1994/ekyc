
import { useQuery } from '@tanstack/react-query'
import {
  Users,
  Building2,
  Clock,
  CheckCircle,
  XCircle,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { dashboardApi } from '../../api/dashboard'
import type { DashboardStats } from '../../types'

// ---------------------------------------------------------------------------
// Stat card
// ---------------------------------------------------------------------------

interface StatCardProps {
  icon: LucideIcon
  iconColor: string
  iconBg: string
  label: string
  value: number | undefined
  isLoading: boolean
}

function StatCard({
  icon: Icon,
  iconColor,
  iconBg,
  label,
  value,
  isLoading,
}: StatCardProps) {
  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-5 flex items-center gap-4">
      <div
        className={['flex items-center justify-center w-11 h-11 rounded-xl shrink-0', iconBg].join(' ')}
        aria-hidden="true"
      >
        <Icon size={22} className={iconColor} />
      </div>
      <div className="min-w-0">
        {isLoading ? (
          <div className="h-6 w-16 bg-slate-200 rounded animate-pulse mb-1" />
        ) : (
          <p className="text-2xl font-bold text-slate-900 leading-none">
            {value?.toLocaleString() ?? '—'}
          </p>
        )}
        <p className="text-xs text-slate-500 mt-1 truncate">{label}</p>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Section heading
// ---------------------------------------------------------------------------

function SectionHeading({ children }: { children: React.ReactNode }) {
  return (
    <h2 className="text-xs font-semibold uppercase tracking-wider text-slate-400 mb-3">
      {children}
    </h2>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function DashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard', 'stats'],
    queryFn: () => dashboardApi.getStats().then((res) => res.data.data),
    refetchInterval: 60_000,
  })

  const stats: DashboardStats | undefined = data

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-slate-900">Dashboard</h1>
        <p className="text-sm text-slate-500 mt-0.5">
          Platform overview — refreshes every minute.
        </p>
      </div>

      {/* Entities */}
      <section aria-labelledby="section-entities" className="mb-6">
        <SectionHeading>
          <span id="section-entities">Entities</span>
        </SectionHeading>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <StatCard
            icon={Users}
            iconColor="text-sky-600"
            iconBg="bg-sky-50"
            label="Total Customers"
            value={stats?.totalCustomers}
            isLoading={isLoading}
          />
          <StatCard
            icon={Building2}
            iconColor="text-purple-600"
            iconBg="bg-purple-50"
            label="Total Companies"
            value={stats?.totalCompanies}
            isLoading={isLoading}
          />
        </div>
      </section>

      {/* KYC */}
      <section aria-labelledby="section-kyc" className="mb-6">
        <SectionHeading>
          <span id="section-kyc">KYC Verifications</span>
        </SectionHeading>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <StatCard
            icon={Clock}
            iconColor="text-amber-600"
            iconBg="bg-amber-50"
            label="Pending"
            value={stats?.totalKycPending}
            isLoading={isLoading}
          />
          <StatCard
            icon={CheckCircle}
            iconColor="text-green-600"
            iconBg="bg-green-50"
            label="Approved"
            value={stats?.totalKycApproved}
            isLoading={isLoading}
          />
          <StatCard
            icon={XCircle}
            iconColor="text-red-600"
            iconBg="bg-red-50"
            label="Rejected"
            value={stats?.totalKycRejected}
            isLoading={isLoading}
          />
        </div>
      </section>

      {/* KYB */}
      <section aria-labelledby="section-kyb">
        <SectionHeading>
          <span id="section-kyb">KYB Verifications</span>
        </SectionHeading>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          <StatCard
            icon={Clock}
            iconColor="text-amber-600"
            iconBg="bg-amber-50"
            label="Pending"
            value={stats?.totalKybPending}
            isLoading={isLoading}
          />
          <StatCard
            icon={CheckCircle}
            iconColor="text-green-600"
            iconBg="bg-green-50"
            label="Approved"
            value={stats?.totalKybApproved}
            isLoading={isLoading}
          />
          <StatCard
            icon={XCircle}
            iconColor="text-red-600"
            iconBg="bg-red-50"
            label="Rejected"
            value={stats?.totalKybRejected}
            isLoading={isLoading}
          />
        </div>
      </section>
    </div>
  )
}

export default DashboardPage
