import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { kybApi } from '../../api/kyb'
import type { ReviewKYBDto, SubmitKYBDto } from '../../api/kyb'
import { companiesApi } from '../../api/companies'
import { uploadFile } from '../../api/upload'
import type { KYBVerification } from '../../types'
import { Badge } from '../../components/ui/Badge'
import type { BadgeVariant } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Table } from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import { Pagination } from '../../components/ui/Pagination'
import { Modal } from '../../components/ui/Modal'
import { useAuth } from '../../auth/useAuth'
import type { RiskLevel } from '../../types'

const RISK_COLORS: Record<RiskLevel, string> = {
  low:      'bg-emerald-100 text-emerald-800',
  medium:   'bg-amber-100 text-amber-800',
  high:     'bg-orange-100 text-orange-800',
  critical: 'bg-red-100 text-red-800',
}

function RiskBadge({ level, score }: { level?: RiskLevel; score?: number }) {
  if (!level) return <span className="text-slate-400 text-xs">—</span>
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${RISK_COLORS[level]}`}>
      {level.charAt(0).toUpperCase() + level.slice(1)}
      {score !== undefined && <span className="opacity-70">({score})</span>}
    </span>
  )
}

const STATUS_TABS = [
  { label: 'All', value: '' },
  { label: 'Pending', value: 'pending' },
  { label: 'In Review', value: 'in_review' },
  { label: 'Approved', value: 'approved' },
  { label: 'Rejected', value: 'rejected' },
]

const PAGE_SIZE = 10

type ModalAction = 'approve' | 'reject' | 'in_review' | 'request_docs'

interface ActionModal {
  open: boolean
  action: ModalAction
  record: KYBVerification | null
}

const closedModal: ActionModal = { open: false, action: 'approve', record: null }

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

export default function EKYBPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const isReviewer = ['admin', 'super_admin', 'reviewer', 'compliance_officer', 'risk_analyst'].includes(user?.role ?? '')

  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState('')
  const [actionModal, setActionModal] = useState<ActionModal>(closedModal)
  const [notes, setNotes] = useState('')

  const [submitOpen, setSubmitOpen] = useState(false)
  const [submitCompanyId, setSubmitCompanyId] = useState('')
  const [submitNotes, setSubmitNotes] = useState('')
  const [submitUploading, setSubmitUploading] = useState(false)
  const [bizDocFile, setBizDocFile] = useState<File | null>(null)
  const [taxDocFile, setTaxDocFile] = useState<File | null>(null)
  const [directorIdFile, setDirectorIdFile] = useState<File | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['kyb', page, statusFilter],
    queryFn: () =>
      kybApi.list({ page, pageSize: PAGE_SIZE, status: statusFilter || undefined }).then(r => r.data),
  })

  const { data: companiesData } = useQuery({
    queryKey: ['companies-all'],
    queryFn: () => companiesApi.list({ page: 1, pageSize: 200 }).then(r => r.data),
    enabled: submitOpen,
  })

  const records: KYBVerification[] = data?.data ?? []
  const companies = companiesData?.data ?? []
  const totalPages = data?.meta ? Math.ceil(data.meta.total / data.meta.pageSize) : 1

  function invalidate() { queryClient.invalidateQueries({ queryKey: ['kyb'] }) }

  const submitMutation = useMutation({
    mutationFn: (dto: SubmitKYBDto) => kybApi.submit(dto),
    onSuccess: () => {
      invalidate()
      setSubmitOpen(false)
      setSubmitCompanyId('')
      setSubmitNotes('')
      setBizDocFile(null)
      setTaxDocFile(null)
      setDirectorIdFile(null)
    },
  })

  async function handleKYBSubmit() {
    if (!submitCompanyId) return
    setSubmitUploading(true)
    try {
      const [businessDocUrl, taxDocUrl, directorIdDocUrl] = await Promise.all([
        bizDocFile ? uploadFile(bizDocFile) : Promise.resolve(undefined),
        taxDocFile ? uploadFile(taxDocFile) : Promise.resolve(undefined),
        directorIdFile ? uploadFile(directorIdFile) : Promise.resolve(undefined),
      ])
      submitMutation.mutate({
        companyId: submitCompanyId,
        notes: submitNotes || undefined,
        businessDocUrl,
        taxDocUrl,
        directorIdDocUrl,
      })
    } finally {
      setSubmitUploading(false)
    }
  }

  const approveMutation = useMutation({
    mutationFn: ({ id, dto }: { id: string; dto: ReviewKYBDto }) => kybApi.approve(id, dto),
    onSuccess: () => { invalidate(); setActionModal(closedModal); setNotes('') },
  })

  const rejectMutation = useMutation({
    mutationFn: ({ id, dto }: { id: string; dto: ReviewKYBDto }) => kybApi.reject(id, dto),
    onSuccess: () => { invalidate(); setActionModal(closedModal); setNotes('') },
  })

  const setInReviewMutation = useMutation({
    mutationFn: ({ id, notes }: { id: string; notes?: string }) => kybApi.setInReview(id, { notes }),
    onSuccess: () => { invalidate(); setActionModal(closedModal); setNotes('') },
  })

  const requestDocsMutation = useMutation({
    mutationFn: ({ id, notes }: { id: string; notes: string }) => kybApi.requestDocs(id, { notes }),
    onSuccess: () => { invalidate(); setActionModal(closedModal); setNotes('') },
  })

  function openAction(record: KYBVerification, action: ModalAction) {
    setNotes('')
    setActionModal({ open: true, action, record })
  }

  function handleConfirm() {
    if (!actionModal.record) return
    const id = actionModal.record.id
    if (actionModal.action === 'approve') approveMutation.mutate({ id, dto: { notes } })
    else if (actionModal.action === 'reject') rejectMutation.mutate({ id, dto: { notes } })
    else if (actionModal.action === 'in_review') setInReviewMutation.mutate({ id, notes: notes || undefined })
    else if (actionModal.action === 'request_docs') requestDocsMutation.mutate({ id, notes })
  }

  const isSubmitting =
    approveMutation.isPending || rejectMutation.isPending ||
    setInReviewMutation.isPending || requestDocsMutation.isPending

  const modalConfig = {
    approve:       { title: 'Approve Verification',          confirmLabel: 'Approve',          variant: 'primary' as const,    notesRequired: false },
    reject:        { title: 'Reject Verification',           confirmLabel: 'Reject',           variant: 'danger' as const,     notesRequired: false },
    in_review:     { title: 'Set In Review',                  confirmLabel: 'Set In Review',    variant: 'primary' as const,    notesRequired: false },
    request_docs:  { title: 'Request Additional Documents',  confirmLabel: 'Send Request',     variant: 'primary' as const,    notesRequired: true  },
  }

  const columns: Column<KYBVerification>[] = [
    {
      key: 'companyName',
      header: 'Company',
      render: (row) => <span className="font-medium text-slate-800">{row.companyName ?? '—'}</span>,
    },
    {
      key: 'submittedBy',
      header: 'Submitted By',
      render: (row) => <span className="text-slate-600 text-sm">{row.submittedBy ?? '—'}</span>,
    },
    {
      key: 'riskLevel',
      header: 'Risk',
      render: (row) => <RiskBadge level={row.riskLevel} score={row.riskScore} />,
    },
    {
      key: 'status',
      header: 'Status',
      render: (row) => <Badge variant={row.status as BadgeVariant} />,
    },
    {
      key: 'submittedAt',
      header: 'Submitted At',
      render: (row) => <span className="text-slate-500 text-sm">{formatDate(row.submittedAt)}</span>,
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row) => (
        <div className="flex items-center gap-1.5 flex-wrap">
          <ActionBtn color="slate" onClick={() => navigate(`/ekyb/${row.id}`)}>View</ActionBtn>
          {isReviewer && row.status === 'pending' && (
            <ActionBtn color="sky" onClick={() => openAction(row, 'in_review')}>In Review</ActionBtn>
          )}
          {isReviewer && (row.status === 'pending' || row.status === 'in_review') && (
            <>
              <ActionBtn color="emerald" onClick={() => openAction(row, 'approve')}>Approve</ActionBtn>
              <ActionBtn color="red" onClick={() => openAction(row, 'reject')}>Reject</ActionBtn>
              <ActionBtn color="amber" onClick={() => openAction(row, 'request_docs')}>Req. Docs</ActionBtn>
            </>
          )}
        </div>
      ),
    },
  ]

  const cfg = actionModal.action ? modalConfig[actionModal.action] : null

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">eKYB Verifications</h1>
          <p className="text-sm text-slate-500 mt-0.5">Business identity verification requests.</p>
        </div>
        <Button variant="primary" onClick={() => setSubmitOpen(true)}>+ Submit KYB</Button>
      </div>

      <div role="tablist" className="flex gap-1 bg-slate-100 p-1 rounded-lg w-fit flex-wrap">
        {STATUS_TABS.map((tab) => (
          <button
            key={tab.value}
            role="tab"
            aria-selected={statusFilter === tab.value}
            onClick={() => { setStatusFilter(tab.value); setPage(1) }}
            className={[
              'px-4 py-1.5 text-sm font-medium rounded-md transition-colors',
              'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500',
              statusFilter === tab.value ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-800',
            ].join(' ')}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <Table columns={columns} data={records} isLoading={isLoading} emptyText="No business verification requests found." />
        <div className="px-4 border-t border-slate-100">
          <Pagination page={page} totalPages={totalPages} onPageChange={setPage} />
        </div>
      </div>

      {/* Submit modal */}
      <Modal open={submitOpen} onClose={() => { if (!submitMutation.isPending && !submitUploading) setSubmitOpen(false) }} title="Submit KYB Verification" maxWidth="max-w-lg">
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <label htmlFor="kyb-company" className="text-sm font-medium text-slate-700">
              Company <span className="text-red-500">*</span>
            </label>
            <select id="kyb-company" value={submitCompanyId} onChange={(e) => setSubmitCompanyId(e.target.value)}
              disabled={submitMutation.isPending || submitUploading}
              className="w-full px-3 py-2 text-sm rounded-lg border border-slate-300 bg-white text-slate-800
                focus:outline-none focus:ring-2 focus:ring-sky-500 disabled:bg-slate-50 disabled:cursor-not-allowed">
              <option value="">— Select company —</option>
              {companies.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </select>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <KYBFileField id="kyb-biz-doc" label="Business Doc (SIUP/NIB)" file={bizDocFile} onChange={setBizDocFile} disabled={submitMutation.isPending || submitUploading} />
            <KYBFileField id="kyb-tax-doc" label="Tax Doc (NPWP)" file={taxDocFile} onChange={setTaxDocFile} disabled={submitMutation.isPending || submitUploading} />
            <KYBFileField id="kyb-dir-id" label="Director ID (KTP)" file={directorIdFile} onChange={setDirectorIdFile} disabled={submitMutation.isPending || submitUploading} />
          </div>
          <div className="flex flex-col gap-1">
            <label htmlFor="kyb-sub-notes" className="text-sm font-medium text-slate-700">
              Notes <span className="text-slate-400 font-normal">(optional)</span>
            </label>
            <textarea id="kyb-sub-notes" rows={2} value={submitNotes} onChange={(e) => setSubmitNotes(e.target.value)}
              placeholder="Additional notes…" disabled={submitMutation.isPending || submitUploading}
              className="w-full px-3 py-2 text-sm rounded-lg border border-slate-300 bg-white
                text-slate-800 placeholder:text-slate-400 resize-none
                focus:outline-none focus:ring-2 focus:ring-sky-500 disabled:bg-slate-50 disabled:cursor-not-allowed" />
          </div>
          {submitMutation.isError && <p className="text-sm text-red-600">Failed to submit. Please try again.</p>}
          <div className="flex justify-end gap-3 pt-2 border-t border-slate-100">
            <Button variant="secondary" onClick={() => setSubmitOpen(false)} disabled={submitMutation.isPending || submitUploading}>Cancel</Button>
            <Button variant="primary" loading={submitMutation.isPending || submitUploading} onClick={handleKYBSubmit} disabled={!submitCompanyId}>
              {submitUploading ? 'Uploading…' : 'Submit'}
            </Button>
          </div>
        </div>
      </Modal>

      {/* Action modal */}
      {cfg && (
        <Modal open={actionModal.open} onClose={() => { if (!isSubmitting) setActionModal(closedModal) }} title={cfg.title} maxWidth="max-w-md">
          <div className="flex flex-col gap-4">
            <p className="text-sm text-slate-600">
              {actionModal.action === 'in_review'
                ? 'Mark this submission as under review for '
                : actionModal.action === 'request_docs'
                ? 'Request additional documents from '
                : actionModal.action === 'approve'
                ? 'Approve eKYB verification for '
                : 'Reject eKYB verification for '}
              <span className="font-semibold text-slate-800">
                {actionModal.record?.companyName ?? actionModal.record?.companyId}
              </span>.
            </p>
            <div className="flex flex-col gap-1">
              <label htmlFor="kyb-action-notes" className="text-sm font-medium text-slate-700">
                Notes
                {cfg.notesRequired
                  ? <span className="text-red-500 ml-1">*</span>
                  : <span className="text-slate-400 font-normal ml-1">(optional)</span>}
              </label>
              <textarea id="kyb-action-notes" rows={4} value={notes} onChange={(e) => setNotes(e.target.value)}
                placeholder={actionModal.action === 'request_docs' ? 'Describe which documents are needed…' : 'Add notes…'}
                disabled={isSubmitting}
                className="w-full px-3 py-2 text-sm rounded-lg border border-slate-300 bg-white
                  text-slate-800 placeholder:text-slate-400 resize-none
                  focus:outline-none focus:ring-2 focus:ring-sky-500
                  disabled:bg-slate-50 disabled:cursor-not-allowed" />
            </div>
            <div className="flex justify-end gap-3 pt-2 border-t border-slate-100">
              <Button variant="secondary" onClick={() => setActionModal(closedModal)} disabled={isSubmitting}>Cancel</Button>
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

interface KYBFileFieldProps {
  id: string
  label: string
  file: File | null
  onChange: (f: File | null) => void
  disabled?: boolean
}

function KYBFileField({ id, label, file, onChange, disabled }: KYBFileFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={id} className="text-xs font-medium text-slate-700 leading-tight">{label}</label>
      <label htmlFor={id}
        className={[
          'flex flex-col items-center justify-center gap-1 px-2 py-3 rounded-lg border-2 border-dashed cursor-pointer transition-colors text-center min-h-[72px]',
          disabled ? 'border-slate-200 bg-slate-50 cursor-not-allowed' : 'border-slate-300 hover:border-sky-400 hover:bg-sky-50',
        ].join(' ')}>
        {file
          ? <span className="text-xs text-sky-700 font-medium break-all line-clamp-2 px-1">{file.name}</span>
          : <span className="text-xs text-slate-400">Upload<br/><span className="text-slate-300 text-[10px]">JPG PNG PDF WEBP</span></span>}
        <input
          id={id}
          type="file"
          accept=".jpg,.jpeg,.png,.pdf,.webp"
          disabled={disabled}
          className="sr-only"
          onChange={(e) => onChange(e.target.files?.[0] ?? null)}
        />
      </label>
      {file && !disabled && (
        <button type="button" onClick={() => onChange(null)} className="text-xs text-red-500 hover:underline text-left">Remove</button>
      )}
    </div>
  )
}

function ActionBtn({ color, onClick, children }: { color: string; onClick: () => void; children: React.ReactNode }) {
  const colors: Record<string, string> = {
    emerald: 'text-emerald-700 bg-emerald-50 hover:bg-emerald-100 focus-visible:ring-emerald-400',
    red:     'text-red-600 bg-red-50 hover:bg-red-100 focus-visible:ring-red-400',
    sky:     'text-sky-700 bg-sky-50 hover:bg-sky-100 focus-visible:ring-sky-400',
    amber:   'text-amber-700 bg-amber-50 hover:bg-amber-100 focus-visible:ring-amber-400',
    slate:   'text-slate-600 bg-slate-50 hover:bg-slate-100 focus-visible:ring-slate-400',
  }
  return (
    <button onClick={onClick}
      className={`inline-flex items-center px-2.5 py-1.5 text-xs font-medium rounded-lg transition-colors
        focus-visible:outline-none focus-visible:ring-2 ${colors[color] ?? colors.sky}`}>
      {children}
    </button>
  )
}

export { EKYBPage }
