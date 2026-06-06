import { useState } from 'react'
import type { RiskAssessment, ManualOverrideDto } from '../../api/risk'

// ─── Label maps ──────────────────────────────────────────────────────────────

const KYC_LABELS: Record<string, string> = {
  missing_id_document: 'ID Document (KTP/Passport)',
  missing_selfie: 'Selfie with ID',
  liveness_risk: 'Liveness Check',
  liveness_score: 'Liveness Score',
  face_match_risk: 'Face Match Check',
  face_match_score: 'Face Match Score',
  manual_override: 'Manual Override',
}

const KYB_LABELS: Record<string, string> = {
  missing_business_doc: 'Business Document (NIB/SIUP)',
  missing_tax_doc: 'Tax Document (NPWP)',
  missing_director_id: 'Director ID (KTP)',
  multiple_docs_missing: 'Multiple Documents Missing',
  company_age_months: 'Company Age (months)',
  company_age_risk: 'Company Age Risk',
  industry: 'Industry',
  industry_risk: 'Industry Risk Level',
  company_status: 'Company Status',
  company_status_risk: 'Company Status Risk',
  manual_override: 'Manual Override',
}

// Keys where the value is just displayed with no colour indicator
const DISPLAY_ONLY_KEYS = new Set([
  'liveness_score',
  'face_match_score',
  'company_age_months',
  'industry',
  'company_status',
  'set_by',
])

// ─── Colour helpers ───────────────────────────────────────────────────────────

const RED_STRINGS = new Set(['high', 'very_new', 'not_checked', 'low_score', 'inactive'])
const AMBER_STRINGS = new Set(['medium', 'new', 'moderate', 'moderate_score', 'pending'])
const GREEN_STRINGS = new Set(['low', 'good', 'established', 'mature', 'active'])

function getFactorColor(
  key: string,
  value: boolean | number | string,
): 'red' | 'amber' | 'green' | 'neutral' {
  if (DISPLAY_ONLY_KEYS.has(key)) return 'neutral'

  if (typeof value === 'boolean') {
    if (key.startsWith('missing') || key === 'multiple_docs_missing') {
      return value ? 'red' : 'green'
    }
    return 'neutral'
  }

  if (typeof value === 'string') {
    if (RED_STRINGS.has(value)) return 'red'
    if (AMBER_STRINGS.has(value)) return 'amber'
    if (GREEN_STRINGS.has(value)) return 'green'
  }

  return 'neutral'
}

const COLOR_CLASSES: Record<'red' | 'amber' | 'green' | 'neutral', string> = {
  red: 'text-red-600 font-medium',
  amber: 'text-amber-600 font-medium',
  green: 'text-emerald-600 font-medium',
  neutral: 'text-slate-600',
}

const INDICATOR_CLASSES: Record<'red' | 'amber' | 'green' | 'neutral', string> = {
  red: 'bg-red-500',
  amber: 'bg-amber-500',
  green: 'bg-emerald-500',
  neutral: '',
}

// ─── Utility ─────────────────────────────────────────────────────────────────

function formatDate(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('en-GB', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatValue(value: boolean | number | string): string {
  if (typeof value === 'boolean') return value ? 'Yes' : 'No'
  return String(value)
}

const RISK_LEVEL_COLORS: Record<string, string> = {
  low: 'bg-emerald-100 text-emerald-800 border-emerald-200',
  medium: 'bg-amber-100 text-amber-800 border-amber-200',
  high: 'bg-orange-100 text-orange-800 border-orange-200',
  critical: 'bg-red-100 text-red-800 border-red-200',
}

// ─── Props ────────────────────────────────────────────────────────────────────

interface RiskBreakdownProps {
  riskAssessment: RiskAssessment | undefined
  history: RiskAssessment[]
  isLoading: boolean
  entityType: 'kyc' | 'kyb'
  isRiskAnalyst: boolean
  onOverride: (level: ManualOverrideDto['risk_level'], notes: string) => void
  isOverriding: boolean
}

// ─── Component ────────────────────────────────────────────────────────────────

export function RiskBreakdown({
  riskAssessment,
  history,
  isLoading,
  entityType,
  isRiskAnalyst,
  onOverride,
  isOverriding,
}: RiskBreakdownProps) {
  const [overrideLevel, setOverrideLevel] = useState<ManualOverrideDto['risk_level']>('low')
  const [overrideNotes, setOverrideNotes] = useState('')
  const [historyOpen, setHistoryOpen] = useState(false)

  const labelMap = entityType === 'kyc' ? KYC_LABELS : KYB_LABELS

  function handleOverrideSubmit(e: React.FormEvent) {
    e.preventDefault()
    onOverride(overrideLevel, overrideNotes)
    setOverrideNotes('')
  }

  const recentHistory = history.slice(0, 5)

  return (
    <div className="flex flex-col gap-4">
      {/* ── Risk Factors ───────────────────────────────────────────────── */}
      <section className="bg-white rounded-xl border border-slate-200 shadow-sm p-5">
        <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-4">
          Risk Factors
        </h2>

        {isLoading ? (
          <div className="flex items-center gap-2 py-4">
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-sky-500 border-t-transparent" />
            <span className="text-sm text-slate-400">Loading risk data…</span>
          </div>
        ) : !riskAssessment ? (
          <p className="text-sm text-slate-400 italic py-2">No risk assessment available.</p>
        ) : (
          <>
            {/* Factor list */}
            <ul className="divide-y divide-slate-100">
              {Object.entries(riskAssessment.riskFactors).map(([key, value]) => {
                const label = labelMap[key] ?? key.replace(/_/g, ' ')
                const color = getFactorColor(key, value)
                const displayVal = formatValue(value)

                return (
                  <li key={key} className="flex items-center justify-between py-2.5 gap-4">
                    <span className="text-sm text-slate-600 shrink-0">{label}</span>
                    <span className={`flex items-center gap-1.5 text-sm text-right ${COLOR_CLASSES[color]}`}>
                      {color !== 'neutral' && (
                        <span className={`inline-block h-2 w-2 rounded-full flex-shrink-0 ${INDICATOR_CLASSES[color]}`} />
                      )}
                      {displayVal}
                    </span>
                  </li>
                )
              })}
            </ul>

            {/* Assessed by / time */}
            <div className="mt-4 pt-3 border-t border-slate-100 flex flex-wrap gap-x-6 gap-y-1">
              {riskAssessment.assessedBy && (
                <span className="text-xs text-slate-400">
                  Assessed by:{' '}
                  <span className="text-slate-600 font-medium">{riskAssessment.assessedBy}</span>
                </span>
              )}
              <span className="text-xs text-slate-400">{formatDate(riskAssessment.assessedAt)}</span>
            </div>

            {/* Manual override form */}
            {isRiskAnalyst && (
              <form onSubmit={handleOverrideSubmit} className="mt-5 pt-4 border-t border-slate-200">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-wide mb-3">
                  Manual Override
                </p>
                <div className="flex flex-col gap-3">
                  <div className="flex flex-col gap-1">
                    <label htmlFor="override-level" className="text-sm font-medium text-slate-700">
                      Risk Level
                    </label>
                    <select
                      id="override-level"
                      value={overrideLevel}
                      onChange={(e) =>
                        setOverrideLevel(e.target.value as ManualOverrideDto['risk_level'])
                      }
                      disabled={isOverriding}
                      className="w-full px-3 py-2 text-sm rounded-lg border border-slate-300 bg-white
                        text-slate-800 focus:outline-none focus:ring-2 focus:ring-sky-500
                        disabled:bg-slate-50 disabled:cursor-not-allowed"
                    >
                      <option value="low">Low</option>
                      <option value="medium">Medium</option>
                      <option value="high">High</option>
                      <option value="critical">Critical</option>
                    </select>
                  </div>

                  <div className="flex flex-col gap-1">
                    <label htmlFor="override-notes" className="text-sm font-medium text-slate-700">
                      Notes{' '}
                      <span className="text-slate-400 font-normal">(optional)</span>
                    </label>
                    <textarea
                      id="override-notes"
                      rows={3}
                      value={overrideNotes}
                      onChange={(e) => setOverrideNotes(e.target.value)}
                      placeholder="Reason for override…"
                      disabled={isOverriding}
                      className="w-full px-3 py-2 text-sm rounded-lg border border-slate-300 bg-white
                        text-slate-800 placeholder:text-slate-400 resize-none
                        focus:outline-none focus:ring-2 focus:ring-sky-500
                        disabled:bg-slate-50 disabled:cursor-not-allowed"
                    />
                  </div>

                  <button
                    type="submit"
                    disabled={isOverriding}
                    className="self-end inline-flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium
                      bg-sky-600 text-white hover:bg-sky-700 active:bg-sky-800
                      disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                  >
                    {isOverriding && (
                      <span className="h-3.5 w-3.5 animate-spin rounded-full border-2 border-white border-t-transparent" />
                    )}
                    Apply Override
                  </button>
                </div>
              </form>
            )}
          </>
        )}
      </section>

      {/* ── Risk History ───────────────────────────────────────────────── */}
      {history.length > 0 && (
        <section className="bg-white rounded-xl border border-slate-200 shadow-sm p-5">
          <button
            type="button"
            onClick={() => setHistoryOpen((v) => !v)}
            className="flex w-full items-center justify-between gap-2"
          >
            <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">
              Risk History
              <span className="ml-2 text-xs font-normal text-slate-400 normal-case tracking-normal">
                ({history.length} assessment{history.length !== 1 ? 's' : ''})
              </span>
            </h2>
            <svg
              className={`w-4 h-4 text-slate-400 transition-transform ${historyOpen ? 'rotate-180' : ''}`}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>

          {historyOpen && (
            <ul className="mt-4 divide-y divide-slate-100">
              {recentHistory.map((item) => {
                const levelColor =
                  RISK_LEVEL_COLORS[item.riskLevel] ??
                  'bg-slate-100 text-slate-600 border-slate-200'
                return (
                  <li key={item.id} className="py-3 flex flex-col gap-1">
                    <div className="flex items-center justify-between gap-3 flex-wrap">
                      <span className="text-xs text-slate-400">{formatDate(item.assessedAt)}</span>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold text-slate-700">{item.riskScore}</span>
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold border ${levelColor}`}
                        >
                          {item.riskLevel.charAt(0).toUpperCase() + item.riskLevel.slice(1)}
                        </span>
                      </div>
                    </div>
                    {item.assessedBy && (
                      <span className="text-xs text-slate-400">
                        By: <span className="text-slate-600">{item.assessedBy}</span>
                      </span>
                    )}
                    {item.notes && (
                      <p className="text-xs text-slate-500 italic">{item.notes}</p>
                    )}
                  </li>
                )
              })}
            </ul>
          )}
        </section>
      )}
    </div>
  )
}
