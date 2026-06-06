import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { kycApi } from '../../api/kyc'
import { kybApi } from '../../api/kyb'
import type { KYCVerification, KYBVerification, RiskLevel } from '../../types'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type AlertSeverity = 'critical' | 'high' | 'info'
type EntityType = 'kyc' | 'kyb'

interface AlertItem {
  id: string
  entityType: EntityType
  entityName: string
  status: string
  riskLevel?: RiskLevel
  riskScore?: number
  submittedAt: string
  severity: AlertSeverity
  link: string
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const RESOLVED = new Set(['approved', 'rejected'])
const HIGH_RISK = new Set<string>(['high', 'critical'])

const SEVERITY_BORDER: Record<AlertSeverity, string> = {
  critical: 'border-l-red-500',
  high:     'border-l-orange-500',
  info:     'border-l-sky-400',
}

const SEVERITY_BADGE: Record<AlertSeverity, string> = {
  critical: 'bg-red-100 text-red-700',
  high:     'bg-orange-100 text-orange-700',
  info:     'bg-sky-100 text-sky-700',
}

const RISK_COLOR: Record<string, string> = {
  low:      'text-emerald-600',
  medium:   'text-amber-600',
  high:     'text-orange-600',
  critical: 'text-red-600',
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function severity(riskLevel?: string): AlertSeverity {
  if (riskLevel === 'critical') return 'critical'
  if (riskLevel === 'high') return 'high'
  return 'info'
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString('en-GB', {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

function timeAgo(iso: string) {
  const diff = Date.now() - new Date(iso).getTime()
  const h = Math.floor(diff / 3_600_000)
  const d = Math.floor(h / 24)
  if (d > 0) return `${d}d ago`
  if (h > 0) return `${h}h ago`
  const m = Math.floor(diff / 60_000)
  return m <= 0 ? 'just now' : `${m}m ago`
}

function kycToAlert(r: KYCVerification): AlertItem {
  return {
    id: r.id,
    entityType: 'kyc',
    entityName: r.customerName ?? r.customerId,
    status: r.status,
    riskLevel: r.riskLevel,
    riskScore: r.riskScore,
    submittedAt: r.submittedAt,
    severity: severity(r.riskLevel),
    link: `/ekyc/${r.id}`,
  }
}

function kybToAlert(r: KYBVerification): AlertItem {
  const kyb = r as KYBVerification & { companyName?: string }
  return {
    id: r.id,
    entityType: 'kyb',
    entityName: kyb.companyName ?? r.companyId,
    status: r.status,
    riskLevel: r.riskLevel,
    riskScore: r.riskScore,
    submittedAt: r.submittedAt,
    severity: severity(r.riskLevel),
    link: `/ekyb/${r.id}`,
  }
}

function sortByOldest(items: AlertItem[]): AlertItem[] {
  return [...items].sort((a, b) => new Date(a.submittedAt).getTime() - new Date(b.submittedAt).getTime())
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function SectionHeader({ title, count, color }: { title: string; count: number; color: string }) {
  return (
    <div className="flex items-center gap-3 mb-3">
      <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">{title}</h2>
      <span className={`inline-flex items-center justify-center min-w-[1.5rem] h-6 px-2 rounded-full text-xs font-bold ${color}`}>
        {count}
      </span>
    </div>
  )
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex items-center justify-center h-14 rounded-xl border border-dashed border-slate-200 bg-slate-50">
      <p className="text-sm text-slate-400">{message}</p>
    </div>
  )
}

function AlertCard({ item, onView }: { item: AlertItem; onView: (link: string) => void }) {
  return (
    <div className={`bg-white rounded-xl border border-slate-200 border-l-4 ${SEVERITY_BORDER[item.severity]} shadow-sm p-4 flex items-start gap-4`}>
      <div className="shrink-0 pt-0.5">
        <span className={`inline-flex px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wide ${
          item.entityType === 'kyc' ? 'bg-sky-100 text-sky-700' : 'bg-purple-100 text-purple-700'
        }`}>
          {item.entityType.toUpperCase()}
        </span>
      </div>

      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap mb-1">
          <span className="text-sm font-semibold text-slate-800 truncate">{item.entityName}</span>
          {item.severity !== 'info' && (
            <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wide ${SEVERITY_BADGE[item.severity]}`}>
              {item.severity}
            </span>
          )}
        </div>
        <div className="flex items-center gap-3 flex-wrap text-xs text-slate-500">
          <span>
            Status: <span className="font-medium text-slate-700 capitalize">{item.status.replace(/_/g, ' ')}</span>
          </span>
          {item.riskLevel && (
            <span>
              Risk:{' '}
              <span className={`font-medium capitalize ${RISK_COLOR[item.riskLevel] ?? ''}`}>
                {item.riskLevel}{item.riskScore != null ? ` (${item.riskScore})` : ''}
              </span>
            </span>
          )}
          <span title={formatDate(item.submittedAt)}>{timeAgo(item.submittedAt)}</span>
        </div>
      </div>

      <button
        onClick={() => onView(item.link)}
        className="shrink-0 px-3 py-1.5 text-xs font-medium rounded-lg bg-slate-100 text-slate-700 hover:bg-sky-50 hover:text-sky-700 transition-colors"
      >
        View
      </button>
    </div>
  )
}

function LoadingSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      {[1, 2, 3].map(i => (
        <div key={i} className="bg-white rounded-xl border border-slate-200 border-l-4 border-l-slate-200 shadow-sm p-4 animate-pulse">
          <div className="flex items-start gap-4">
            <div className="w-10 h-5 rounded bg-slate-200 shrink-0" />
            <div className="flex-1">
              <div className="h-4 w-40 bg-slate-200 rounded mb-2" />
              <div className="h-3 w-64 bg-slate-100 rounded" />
            </div>
            <div className="w-12 h-7 bg-slate-100 rounded-lg shrink-0" />
          </div>
        </div>
      ))}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function MonitoringPage() {
  const navigate = useNavigate()

  const { data: kycData, isLoading: kycLoading, dataUpdatedAt: kycUpdatedAt } = useQuery({
    queryKey: ['kyc-monitoring'],
    queryFn: () => kycApi.list({ pageSize: 100 }).then(r => r.data.data ?? []),
    refetchInterval: 60_000,
  })

  const { data: kybData, isLoading: kybLoading, dataUpdatedAt: kybUpdatedAt } = useQuery({
    queryKey: ['kyb-monitoring'],
    queryFn: () => kybApi.list({ pageSize: 100 }).then(r => r.data.data ?? []),
    refetchInterval: 60_000,
  })

  const isLoading = kycLoading || kybLoading
  const kycRecords: KYCVerification[] = kycData ?? []
  const kybRecords: KYBVerification[] = kybData ?? []

  const highRiskAlerts = sortByOldest([
    ...kycRecords.filter(r => !RESOLVED.has(r.status) && HIGH_RISK.has(r.riskLevel ?? '')).map(kycToAlert),
    ...kybRecords.filter(r => !RESOLVED.has(r.status) && HIGH_RISK.has(r.riskLevel ?? '')).map(kybToAlert),
  ])

  const pendingAlerts = sortByOldest([
    ...kycRecords.filter(r => r.status === 'pending').map(kycToAlert),
    ...kybRecords.filter(r => r.status === 'pending').map(kybToAlert),
  ])

  const inReviewAlerts = sortByOldest([
    ...kycRecords.filter(r => r.status === 'in_review').map(kycToAlert),
    ...kybRecords.filter(r => r.status === 'in_review').map(kybToAlert),
  ])

  const totalAlerts = highRiskAlerts.length + pendingAlerts.length
  const lastUpdated = Math.max(kycUpdatedAt, kybUpdatedAt)

  return (
    <div className="flex flex-col gap-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Monitoring</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            Active alerts and verifications requiring attention — refreshes every minute.
          </p>
        </div>
        <div className="flex items-center gap-3">
          {totalAlerts > 0 && (
            <span className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-red-50 text-red-700 text-sm font-semibold border border-red-200">
              <span className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
              {totalAlerts} open alert{totalAlerts !== 1 ? 's' : ''}
            </span>
          )}
          {lastUpdated > 0 && (
            <span className="text-xs text-slate-400">
              Updated {timeAgo(new Date(lastUpdated).toISOString())}
            </span>
          )}
        </div>
      </div>

      {isLoading ? (
        <LoadingSkeleton />
      ) : (
        <div className="flex flex-col gap-8">
          {/* High Risk */}
          <section>
            <SectionHeader title="High Risk" count={highRiskAlerts.length} color="bg-red-100 text-red-700" />
            {highRiskAlerts.length === 0
              ? <EmptyState message="No high-risk verifications" />
              : (
                <div className="flex flex-col gap-2">
                  {highRiskAlerts.map(item => (
                    <AlertCard key={item.id} item={item} onView={navigate} />
                  ))}
                </div>
              )}
          </section>

          {/* Pending Review */}
          <section>
            <SectionHeader title="Pending Review" count={pendingAlerts.length} color="bg-amber-100 text-amber-700" />
            {pendingAlerts.length === 0
              ? <EmptyState message="No pending verifications" />
              : (
                <div className="flex flex-col gap-2">
                  {pendingAlerts.map(item => (
                    <AlertCard key={item.id} item={item} onView={navigate} />
                  ))}
                </div>
              )}
          </section>

          {/* In Review */}
          <section>
            <SectionHeader title="In Review" count={inReviewAlerts.length} color="bg-sky-100 text-sky-700" />
            {inReviewAlerts.length === 0
              ? <EmptyState message="No verifications currently in review" />
              : (
                <div className="flex flex-col gap-2">
                  {inReviewAlerts.map(item => (
                    <AlertCard key={item.id} item={item} onView={navigate} />
                  ))}
                </div>
              )}
          </section>
        </div>
      )}
    </div>
  )
}
