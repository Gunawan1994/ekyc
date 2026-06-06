import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { Plus, Search, Pencil, Trash2, AlertTriangle } from 'lucide-react'
import toast from 'react-hot-toast'
import { customersApi } from '../../api/customers'
import type { Customer } from '../../types'
import { Button } from '../../components/ui/Button'
import { Table } from '../../components/ui/Table'
import { Pagination } from '../../components/ui/Pagination'
import { Modal } from '../../components/ui/Modal'
import { CustomerForm } from './CustomerForm'
import type { Column } from '../../components/ui/Table'

const PAGE_SIZE = 10

const ID_TYPE_LABELS: Record<Customer['idType'], string> = {
  ktp: 'KTP',
  passport: 'Passport',
  sim: 'SIM',
}

// ---------------------------------------------------------------------------
// Delete confirmation dialog
// ---------------------------------------------------------------------------

interface DeleteDialogProps {
  customer: Customer | null
  onConfirm: () => void
  onCancel: () => void
  isDeleting: boolean
}

function DeleteDialog({
  customer,
  onConfirm,
  onCancel,
  isDeleting,
}: DeleteDialogProps) {
  return (
    <Modal
      open={!!customer}
      onClose={onCancel}
      title="Delete Customer"
      maxWidth="max-w-sm"
    >
      <div className="flex flex-col gap-4">
        <div className="flex items-start gap-3">
          <div className="flex items-center justify-center w-10 h-10 rounded-full bg-red-100 shrink-0">
            <AlertTriangle size={18} className="text-red-600" aria-hidden="true" />
          </div>
          <p className="text-sm text-slate-700 pt-2">
            Are you sure you want to delete{' '}
            <span className="font-semibold">{customer?.fullName}</span>? This action
            cannot be undone.
          </p>
        </div>
        <div className="flex justify-end gap-2 pt-1">
          <Button variant="secondary" onClick={onCancel} disabled={isDeleting}>
            Cancel
          </Button>
          <Button variant="danger" onClick={onConfirm} loading={isDeleting}>
            Delete
          </Button>
        </div>
      </div>
    </Modal>
  )
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export function CustomersPage() {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  const page = Number(searchParams.get('page') ?? '1')
  const search = searchParams.get('search') ?? ''

  const [searchInput, setSearchInput] = useState(search)
  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Customer | null>(null)
  const [deleteTarget, setDeleteTarget] = useState<Customer | null>(null)

  // ---- data ----
  const { data, isLoading } = useQuery({
    queryKey: ['customers', { page, search }],
    queryFn: () =>
      customersApi
        .list({ page, pageSize: PAGE_SIZE, search: search || undefined })
        .then((res) => res.data),
  })

  const customers: Customer[] = data?.data ?? []
  const totalPages = data?.meta
    ? Math.max(1, Math.ceil(data.meta.total / data.meta.pageSize))
    : 1

  // ---- delete ----
  const deleteMutation = useMutation({
    mutationFn: (id: string) => customersApi.delete(id),
    onSuccess: () => {
      toast.success('Customer deleted.')
      setDeleteTarget(null)
      queryClient.invalidateQueries({ queryKey: ['customers'] })
    },
    onError: () => {
      toast.error('Failed to delete customer.')
    },
  })

  // ---- helpers ----
  function applySearch(value: string) {
    setSearchParams({ page: '1', ...(value ? { search: value } : {}) })
  }

  function handleSearchKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') applySearch(searchInput)
  }

  function handleSearchBlur() {
    if (searchInput !== search) applySearch(searchInput)
  }

  function handlePageChange(next: number) {
    setSearchParams({ page: String(next), ...(search ? { search } : {}) })
  }

  function openCreate() {
    setEditTarget(null)
    setFormOpen(true)
  }

  function openEdit(customer: Customer) {
    setEditTarget(customer)
    setFormOpen(true)
  }

  function closeForm() {
    setFormOpen(false)
    setEditTarget(null)
  }

  // ---- table columns ----
  const columns: Column<Customer>[] = [
    {
      key: 'fullName',
      header: 'Name',
      render: (row) => (
        <span className="font-medium text-slate-800">{row.fullName}</span>
      ),
    },
    { key: 'idNumber', header: 'ID Number' },
    {
      key: 'idType',
      header: 'ID Type',
      render: (row) => (
        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700">
          {ID_TYPE_LABELS[row.idType]}
        </span>
      ),
    },
    { key: 'email', header: 'Email' },
    { key: 'phone', header: 'Phone' },
    {
      key: 'actions',
      header: 'Actions',
      render: (row) => (
        <div className="flex items-center gap-1">
          <button
            onClick={() => openEdit(row)}
            className="p-1.5 rounded-lg text-slate-400 hover:text-sky-600 hover:bg-sky-50
              transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
            aria-label={`Edit ${row.fullName}`}
          >
            <Pencil size={15} />
          </button>
          <button
            onClick={() => setDeleteTarget(row)}
            className="p-1.5 rounded-lg text-slate-400 hover:text-red-600 hover:bg-red-50
              transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-500"
            aria-label={`Delete ${row.fullName}`}
          >
            <Trash2 size={15} />
          </button>
        </div>
      ),
    },
  ]

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6 gap-4 flex-wrap">
        <h1 className="text-2xl font-bold text-slate-900">Customers</h1>
        <Button onClick={openCreate}>
          <Plus size={16} aria-hidden="true" />
          Add Customer
        </Button>
      </div>

      {/* Search */}
      <div className="relative mb-4 max-w-sm">
        <Search
          size={15}
          className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400 pointer-events-none"
          aria-hidden="true"
        />
        <input
          type="search"
          placeholder="Search by name, email, or ID..."
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          onKeyDown={handleSearchKeyDown}
          onBlur={handleSearchBlur}
          className="w-full pl-9 pr-3 py-2 text-sm rounded-lg border border-slate-300 bg-white
            text-slate-800 placeholder:text-slate-400
            focus:outline-none focus:ring-2 focus:ring-sky-500 focus:border-sky-500 transition-colors"
          aria-label="Search customers"
        />
      </div>

      {/* Table */}
      <Table
        columns={columns}
        data={customers}
        isLoading={isLoading}
        emptyText="No customers found."
      />

      {/* Pagination */}
      <Pagination
        page={page}
        totalPages={totalPages}
        onPageChange={handlePageChange}
      />

      {/* Add / Edit form modal */}
      <CustomerForm
        open={formOpen}
        onClose={closeForm}
        customer={editTarget}
        onSuccess={() => {
          closeForm()
          queryClient.invalidateQueries({ queryKey: ['customers'] })
        }}
      />

      {/* Delete confirmation */}
      <DeleteDialog
        customer={deleteTarget}
        onConfirm={() => {
          if (deleteTarget) deleteMutation.mutate(deleteTarget.id)
        }}
        onCancel={() => setDeleteTarget(null)}
        isDeleting={deleteMutation.isPending}
      />
    </div>
  )
}

export default CustomersPage
