import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { kycApi } from '../../api/kyc'
import type { ReviewKYCDto } from '../../api/kyc'
import { riskApi } from '../../api/risk'
import type { ManualOverrideDto } from '../../api/risk'
import { Badge } from '../../components/ui/Badge'
import type { BadgeVariant } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../auth/useAuth'
import type { RiskLevel } from '../../types'
import { RiskBreakdown } from '../../components/risk/RiskBreakdown'

const RISK_COLORS: Record<RiskLevel, string> = {
  low:      'bg-emerald-100 text-emerald-800 border-emerald-200',
  medium:   'bg-amber-100 text-amber-800 border-amber-200',
  high:     'bg-orange-100 text-orange-800 border-orange-200',
  critical: 'bg-red-100 text-red-800 border-red-200',
}

const RISK_BAR_COLORS: Record<RiskLevel, string> = {
  low:      'bg-emerald-500',
  medium:   'bg-amber-500',
  high:     'bg-orange-500',
  critical: 'bg-red-500',
}

function formatDate(iso?: string) {
  if (!iso) return '—'
  return new Date(iso).toLocaleString('en-GB', {
    day: '2-digit', month: 'short', year: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}


function DocLink({ label, url }: { label: string; url?: string }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs font-medium text-slate-500 uppercase tracking-wide">{label}</span>
      {url
        ? (
          <a href={url} target="_blank" rel="noopener noreferrer"
            className="inline-flex items-center gap-1.5 text-sm text-sky-600 hover:text-sky-800 hover:underline font-medium break-all">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M13.5 6H5.25A2.25 2.25 0 003 8.25v10.5A2.25 2.25 0 005.25 21h10.5A2.25 2.25 0 0018 18.75V10.5m-10.5 6L21 3m0 0h-5.25M21 3v5.25" />
            </svg>
            View document
          </a>
        )
        : <span className="text-sm text-slate-400 italic">Not provided</span>}
    </div>
  )
}

function InfoRow({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-xs text-slate-400 uppercase tracking-wide">{label}</span>
      <span className={`text-sm text-slate-700 break-all ${mono ? 'font-mono text-xs' : ''}`}>{value}</span>
    </div>
  )
}

type ModalAction = 'approve' | 'reject' | 'in_review' | 'request_docs'

const MODAL_CFG = {
  approve:      { title: 'Approve Verification',         confirmLabel: 'Approve',       variant: 'primary' as const, notesRequired: false },
  reject:       { title: 'Reject Verification',          confirmLabel: 'Reject',        variant: 'danger' as const,  notesRequired: false },
  in_review:    { title: 'Set In Review',                 confirmLabel: 'Set In Review', variant: 'primary' as const, notesRequired: false },
  request_docs: { title: 'Request Additional Documents', confirmLabel: 'Send Request',  variant: 'primary' as const, notesRequired: true  },
}

export default function EKYCDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const isReviewer = ['admin', 'super_admin', 'reviewer', 'compliance_officer', 'risk_analyst'].includes(user?.role ?? '')
  const isAdmin = ['admin', 'super_admin'].includes(user?.role ?? '')
  const isRiskAnalyst = ['risk_analyst', 'admin', 'super_admin'].includes(user?.role ?? '')

  const [action, setAction] = useState<ModalAction | null>(null)
  const [notes, setNotes] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState(false)

  const { data, isLoading, isError } = useQuery({
    queryKey: ['kyc-detail', id],
    queryFn: () => kycApi.getById(id!).then(r => r.data),
    enabled: !!id,
  })

  const { data: riskData, isLoading: riskLoading } = useQuery({
    queryKey: ['kyc-risk', id],
    queryFn: () => riskApi.getKYCRisk(id!).then(r => r.data.data),
    enabled: !!id,
  })

  const { data: riskHistory } = useQuery({
    queryKey: ['kyc-risk-history', id],
    queryFn: () => riskApi.listKYCHistory(id!).then(r => r.data.data ?? []),
    enabled: !!id,
  })

  const overrideMutation = useMutation({
    mutationFn: (dto: ManualOverrideDto) => riskApi.overrideKYCRisk(id!, dto),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kyc-risk', id] })
      queryClient.invalidateQueries({ queryKey: ['kyc-risk-history', id] })
      queryClient.invalidateQueries({ queryKey: ['kyc-detail', id] })
      queryClient.invalidateQueries({ queryKey: ['kyc-monitoring'] })
      queryClient.invalidateQueries({ queryKey: ['kyc'] })
    },
  })

  const record = data?.data

  function invalidate() {
    queryClient.invalidateQueries({ queryKey: ['kyc-detail', id] })
    queryClient.invalidateQueries({ queryKey: ['kyc'] })
    queryClient.invalidateQueries({ queryKey: ['kyc-monitoring'] })
    queryClient.invalidateQueries({ queryKey: ['dashboard', 'stats'] })
  }

  const approveMutation = useMutation({
    mutationFn: (dto: ReviewKYCDto) => kycApi.approve(id!, dto),
    onSuccess: () => { invalidate(); setAction(null); setNotes('') },
  })
  const rejectMutation = useMutation({
    mutationFn: (dto: ReviewKYCDto) => kycApi.reject(id!, dto),
    onSuccess: () => { invalidate(); setAction(null); setNotes('') },
  })
  const setInReviewMutation = useMutation({
    mutationFn: (n?: string) => kycApi.setInReview(id!, { notes: n }),
    onSuccess: () => { invalidate(); setAction(null); setNotes('') },
  })
  const requestDocsMutation = useMutation({
    mutationFn: (n: string) => kycApi.requestDocs(id!, { notes: n }),
    onSuccess: () => { invalidate(); setAction(null); setNotes('') },
  })

  const deleteMutation = useMutation({
    mutationFn: () => kycApi.delete(id!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['kyc'] })
      queryClient.invalidateQueries({ queryKey: ['kyc-monitoring'] })
      queryClient.invalidateQueries({ queryKey: ['dashboard', 'stats'] })
      navigate('/ekyc')
    },
  })

  const isSubmitting =
    approveMutation.isPending || rejectMutation.isPending ||
    setInReviewMutation.isPending || requestDocsMutation.isPending

  function handleConfirm() {
    if (action === 'approve') approveMutation.mutate({ notes })
    else if (action === 'reject') rejectMutation.mutate({ notes })
    else if (action === 'in_review') setInReviewMutation.mutate(notes || undefined)
    else if (action === 'request_docs') requestDocsMutation.mutate(notes)
  }

  const cfg = action ? MODAL_CFG[action] : null

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-sky-600 border-t-transparent" />
      </div>
    )
  }

  if (isError || !record) {
    return (
      <div className="flex flex-col items-center justify-center h-64 gap-3">
        <p className="text-slate-500">Verification not found.</p>
        <Button variant="secondary" onClick={() => navigate('/ekyc')}>Back to list</Button>
      </div>
    )
  }

  const riskLevel = record.riskLevel as RiskLevel | undefined

  return (
    <div className="flex flex-col gap-6 max-w-4xl">
      {/* Header */}
      <div className="flex items-start justify-between gap-4 flex-wrap">
        <div className="flex items-center gap-3">
          <button
            onClick={() => navigate('/ekyc')}
            className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-800 transition-colors"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
            </svg>
            eKYC
          </button>
          <span className="text-slate-300">/</span>
          <h1 className="text-lg font-semibold text-slate-900">
            {record.customerName ?? record.customerId}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={record.status as BadgeVariant} />
          {riskLevel && (
            <span className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold border ${RISK_COLORS[riskLevel]}`}>
              Risk: {riskLevel.charAt(0).toUpperCase() + riskLevel.slice(1)}
              {record.riskScore != null && <span className="opacity-70 ml-0.5">({record.riskScore})</span>}
            </span>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Main content */}
        <div className="lg:col-span-2 flex flex-col gap-4">

          {/* Documents */}
          <section className="bg-white rounded-xl border border-slate-200 shadow-sm p-5">
            <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-4">Documents</h2>
            <div className="grid grid-cols-2 gap-6">
              <DocLink label="ID Document (KTP/Passport)" url={record.idDocumentUrl} />
              <DocLink label="Selfie with ID" url={record.selfieUrl} />
            </div>
          </section>

          {/* Notes / rejection */}
          {(record.notes || record.rejectionReason) && (
            <section className="bg-white rounded-xl border border-slate-200 shadow-sm p-5">
              <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-3">Notes</h2>
              <div className="flex flex-col gap-3">
                {record.notes && (
                  <div>
                    <p className="text-xs text-slate-500 mb-1">Submission Notes</p>
                    <p className="text-sm text-slate-700 whitespace-pre-wrap">{record.notes}</p>
                  </div>
                )}
                {record.rejectionReason && (
                  <div className="p-3 rounded-lg bg-red-50 border border-red-100">
                    <p className="text-xs text-red-600 font-medium mb-1">Rejection Reason</p>
                    <p className="text-sm text-red-700 whitespace-pre-wrap">{record.rejectionReason}</p>
                  </div>
                )}
              </div>
            </section>
          )}

          {/* Risk Breakdown */}
          <RiskBreakdown
            riskAssessment={riskData}
            history={riskHistory ?? []}
            isLoading={riskLoading}
            entityType="kyc"
            isRiskAnalyst={isRiskAnalyst}
            onOverride={(level, notes) => overrideMutation.mutate({ risk_level: level, notes })}
            isOverriding={overrideMutation.isPending}
          />
        </div>

        {/* Sidebar */}
        <div className="flex flex-col gap-4">
          {/* Timeline */}
          <section className="bg-white rounded-xl border border-slate-200 shadow-sm p-5">
            <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-4">Info</h2>
            <div className="flex flex-col gap-3 text-sm">
              <InfoRow label="Submitted" value={formatDate(record.submittedAt)} />
              <InfoRow label="Reviewed" value={formatDate(record.reviewedAt)} />
              <InfoRow label="Customer ID" value={record.customerId} mono />
              <InfoRow label="Submitted By" value={record.submittedBy} mono />
              {record.reviewerId && <InfoRow label="Reviewer ID" value={record.reviewerId} mono />}
            </div>
          </section>

          {/* Risk card */}
          {riskLevel && (
            <section className={`rounded-xl border p-5 ${RISK_COLORS[riskLevel]}`}>
              <h2 className="text-xs font-semibold uppercase tracking-wide opacity-70 mb-3">Risk Assessment</h2>
              <div className="flex items-end justify-between mb-2">
                <span className="text-2xl font-bold">{record.riskScore ?? 0}</span>
                <span className="text-xs font-semibold uppercase opacity-70">{riskLevel}</span>
              </div>
              <div className="h-2 rounded-full bg-black/10 overflow-hidden">
                <div className={`h-full rounded-full ${RISK_BAR_COLORS[riskLevel]}`}
                  style={{ width: `${record.riskScore ?? 0}%` }} />
              </div>
            </section>
          )}

          {/* Actions */}
          {isReviewer && (
            <section className="bg-white rounded-xl border border-slate-200 shadow-sm p-5">
              <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide mb-3">Actions</h2>
              <div className="flex flex-col gap-2">
                {record.status === 'pending' && (
                  <Button variant="secondary" onClick={() => { setNotes(''); setAction('in_review') }}>
                    Set In Review
                  </Button>
                )}
                {(record.status === 'pending' || record.status === 'in_review') && (
                  <>
                    <Button variant="primary" onClick={() => { setNotes(''); setAction('approve') }}>
                      Approve
                    </Button>
                    <Button variant="danger" onClick={() => { setNotes(''); setAction('reject') }}>
                      Reject
                    </Button>
                    <Button variant="secondary" onClick={() => { setNotes(''); setAction('request_docs') }}>
                      Request Documents
                    </Button>
                  </>
                )}
                {(record.status === 'approved' || record.status === 'rejected') && (
                  <p className="text-sm text-slate-400 text-center py-1">Verification completed.</p>
                )}
                {isAdmin && (
                  <div className="pt-2 mt-1 border-t border-slate-100">
                    <Button variant="danger" onClick={() => setDeleteConfirm(true)}>
                      Delete Record
                    </Button>
                  </div>
                )}
              </div>
            </section>
          )}
        </div>
      </div>

      {/* Delete confirm modal */}
      <Modal open={deleteConfirm} onClose={() => { if (!deleteMutation.isPending) setDeleteConfirm(false) }} title="Delete Verification" maxWidth="max-w-md">
        <div className="flex flex-col gap-4">
          <p className="text-sm text-slate-600">
            Permanently delete eKYC record for{' '}
            <span className="font-semibold text-slate-800">{record?.customerName ?? record?.customerId}</span>?
            This cannot be undone.
          </p>
          {deleteMutation.isError && <p className="text-sm text-red-600">Failed to delete. Please try again.</p>}
          <div className="flex justify-end gap-3 pt-2 border-t border-slate-100">
            <Button variant="secondary" onClick={() => setDeleteConfirm(false)} disabled={deleteMutation.isPending}>Cancel</Button>
            <Button variant="danger" loading={deleteMutation.isPending} onClick={() => deleteMutation.mutate()}>Delete</Button>
          </div>
        </div>
      </Modal>

      {/* Action modal */}
      {cfg && action && (
        <Modal open={!!action} onClose={() => { if (!isSubmitting) setAction(null) }} title={cfg.title} maxWidth="max-w-md">
          <div className="flex flex-col gap-4">
            <p className="text-sm text-slate-600">
              {action === 'in_review' ? 'Mark this submission as under review for ' :
               action === 'request_docs' ? 'Request additional documents from ' :
               action === 'approve' ? 'Approve eKYC verification for ' :
               'Reject eKYC verification for '}
              <span className="font-semibold text-slate-800">{record.customerName ?? record.customerId}</span>.
            </p>
            <div className="flex flex-col gap-1">
              <label htmlFor="detail-notes" className="text-sm font-medium text-slate-700">
                Notes {cfg.notesRequired
                  ? <span className="text-red-500 ml-1">*</span>
                  : <span className="text-slate-400 font-normal ml-1">(optional)</span>}
              </label>
              <textarea id="detail-notes" rows={4} value={notes} onChange={(e) => setNotes(e.target.value)}
                placeholder={action === 'request_docs' ? 'Describe which documents are needed…' : 'Add notes…'}
                disabled={isSubmitting}
                className="w-full px-3 py-2 text-sm rounded-lg border border-slate-300 bg-white
                  text-slate-800 placeholder:text-slate-400 resize-none
                  focus:outline-none focus:ring-2 focus:ring-sky-500 disabled:bg-slate-50 disabled:cursor-not-allowed" />
            </div>
            <div className="flex justify-end gap-3 pt-2 border-t border-slate-100">
              <Button variant="secondary" onClick={() => setAction(null)} disabled={isSubmitting}>Cancel</Button>
              <Button variant={cfg.variant} loading={isSubmitting} onClick={handleConfirm}
                disabled={cfg.notesRequired && !notes.trim()}>
                {cfg.confirmLabel}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}
