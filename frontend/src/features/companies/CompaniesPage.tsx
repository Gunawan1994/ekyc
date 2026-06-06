import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { PlusIcon, PencilIcon, TrashIcon } from 'lucide-react'
import { companiesApi } from '../../api/companies'
import type { CreateCompanyDto, UpdateCompanyDto } from '../../api/companies'
import type { Company } from '../../types'
import { Badge } from '../../components/ui/Badge'
import type { BadgeVariant } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'
import { Select } from '../../components/ui/Select'
import { Table } from '../../components/ui/Table'
import type { Column } from '../../components/ui/Table'
import { Pagination } from '../../components/ui/Pagination'
import { Modal } from '../../components/ui/Modal'
import { CompanyForm } from './CompanyForm'

const STATUS_OPTIONS = [
  { value: '', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]

const PAGE_SIZE = 10

export default function CompaniesPage() {
  const queryClient = useQueryClient()

  // List state
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('')

  // Modal state
  const [formOpen, setFormOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Company | undefined>(undefined)
  const [deleteTarget, setDeleteTarget] = useState<Company | undefined>(
    undefined
  )

  const { data, isLoading } = useQuery({
    queryKey: ['companies', page, search, status],
    queryFn: () =>
      companiesApi
        .list({
          page,
          pageSize: PAGE_SIZE,
          search: search || undefined,
          status: status || undefined,
        })
        .then((res) => res.data),
  })

  const companies: Company[] = data?.data ?? []
  const totalPages = data?.meta
    ? Math.ceil(data.meta.total / data.meta.pageSize)
    : 1

  const createMutation = useMutation({
    mutationFn: (dto: CreateCompanyDto) => companiesApi.create(dto),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['companies'] })
      setFormOpen(false)
    },
  })

  const updateMutation = useMutation({
    mutationFn: ({ id, dto }: { id: string; dto: UpdateCompanyDto }) =>
      companiesApi.update(id, dto),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['companies'] })
      setFormOpen(false)
      setEditTarget(undefined)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => companiesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['companies'] })
      setDeleteTarget(undefined)
    },
  })

  function handleOpenCreate() {
    setEditTarget(undefined)
    setFormOpen(true)
  }

  function handleOpenEdit(company: Company) {
    setEditTarget(company)
    setFormOpen(true)
  }

  function handleFormSubmit(payload: CreateCompanyDto | UpdateCompanyDto) {
    if (editTarget) {
      updateMutation.mutate({ id: editTarget.id, dto: payload as UpdateCompanyDto })
    } else {
      createMutation.mutate(payload as CreateCompanyDto)
    }
  }

  function handleSearchChange(e: React.ChangeEvent<HTMLInputElement>) {
    setSearch(e.target.value)
    setPage(1)
  }

  function handleStatusChange(e: React.ChangeEvent<HTMLSelectElement>) {
    setStatus(e.target.value)
    setPage(1)
  }

  const isFormSubmitting = createMutation.isPending || updateMutation.isPending

  const columns: Column<Company>[] = [
    { key: 'name', header: 'Name' },
    { key: 'registrationNumber', header: 'Registration Number' },
    { key: 'email', header: 'Email' },
    {
      key: 'status',
      header: 'Status',
      render: (row) => <Badge variant={row.status as BadgeVariant} />,
    },
    {
      key: 'actions',
      header: 'Actions',
      render: (row) => (
        <div className="flex items-center gap-2">
          <button
            onClick={() => handleOpenEdit(row)}
            className="inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium rounded-lg
              text-slate-600 bg-slate-100 hover:bg-slate-200 transition-colors
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
            aria-label={`Edit ${row.name}`}
          >
            <PencilIcon size={13} />
            Edit
          </button>
          <button
            onClick={() => setDeleteTarget(row)}
            className="inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-medium rounded-lg
              text-red-600 bg-red-50 hover:bg-red-100 transition-colors
              focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-red-400"
            aria-label={`Delete ${row.name}`}
          >
            <TrashIcon size={13} />
            Delete
          </button>
        </div>
      ),
    },
  ]

  return (
    <div className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">Companies</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            Manage registered companies in the platform.
          </p>
        </div>
        <Button variant="primary" size="md" onClick={handleOpenCreate}>
          <PlusIcon size={16} />
          Add Company
        </Button>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-end gap-3">
        <div className="w-64">
          <Input
            label="Search"
            placeholder="Name, email, reg. number…"
            value={search}
            onChange={handleSearchChange}
          />
        </div>
        <div className="w-44">
          <Select
            label="Status"
            options={STATUS_OPTIONS}
            value={status}
            onChange={handleStatusChange}
          />
        </div>
      </div>

      {/* Table */}
      <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
        <Table
          columns={columns}
          data={companies}
          isLoading={isLoading}
          emptyText="No companies found."
        />
        <div className="px-4 border-t border-slate-100">
          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={setPage}
          />
        </div>
      </div>

      {/* Create / Edit modal */}
      <CompanyForm
        open={formOpen}
        onClose={() => {
          setFormOpen(false)
          setEditTarget(undefined)
        }}
        onSubmit={handleFormSubmit}
        isSubmitting={isFormSubmitting}
        initialValues={editTarget}
      />

      {/* Delete confirmation modal */}
      <Modal
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(undefined)}
        title="Delete Company"
        maxWidth="max-w-sm"
      >
        <p className="text-sm text-slate-600">
          Are you sure you want to delete{' '}
          <span className="font-semibold text-slate-800">
            {deleteTarget?.name}
          </span>
          ? This action cannot be undone.
        </p>
        <div className="flex justify-end gap-3 mt-6">
          <Button
            variant="secondary"
            onClick={() => setDeleteTarget(undefined)}
            disabled={deleteMutation.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="danger"
            loading={deleteMutation.isPending}
            onClick={() =>
              deleteTarget && deleteMutation.mutate(deleteTarget.id)
            }
          >
            Delete
          </Button>
        </div>
      </Modal>
    </div>
  )
}

export { CompaniesPage }
