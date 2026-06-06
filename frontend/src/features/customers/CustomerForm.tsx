import { useEffect, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import toast from 'react-hot-toast'
import { customersApi } from '../../api/customers'
import type { CreateCustomerDto } from '../../api/customers'
import { companiesApi } from '../../api/companies'
import type { Customer } from '../../types'
import { Modal } from '../../components/ui/Modal'
import { Input } from '../../components/ui/Input'
import { Select } from '../../components/ui/Select'
import { Button } from '../../components/ui/Button'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface CustomerFormProps {
  open: boolean
  onClose: () => void
  /** When provided, the form operates in edit mode. */
  customer: Customer | null
  onSuccess: () => void
}

interface FormState {
  fullName: string
  idNumber: string
  idType: 'ktp' | 'passport' | 'sim' | ''
  companyId: string
  phone: string
  email: string
  address: string
}

interface FormErrors {
  fullName?: string
  idNumber?: string
  idType?: string
  companyId?: string
  phone?: string
  email?: string
  address?: string
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const EMPTY_FORM: FormState = {
  fullName: '',
  idNumber: '',
  idType: '',
  companyId: '',
  phone: '',
  email: '',
  address: '',
}

const ID_TYPE_OPTIONS = [
  { value: 'ktp', label: 'KTP' },
  { value: 'passport', label: 'Passport' },
  { value: 'sim', label: 'SIM' },
]

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

function validate(form: FormState): FormErrors {
  const errors: FormErrors = {}

  if (!form.fullName.trim()) {
    errors.fullName = 'Full name is required.'
  }
  if (!form.idNumber.trim()) {
    errors.idNumber = 'ID number is required.'
  }
  if (!form.idType) {
    errors.idType = 'ID type is required.'
  }
  if (!form.companyId) {
    errors.companyId = 'Company is required.'
  }
  if (!form.phone.trim()) {
    errors.phone = 'Phone number is required.'
  }
  if (!form.email.trim()) {
    errors.email = 'Email is required.'
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    errors.email = 'Enter a valid email address.'
  }
  if (!form.address.trim()) {
    errors.address = 'Address is required.'
  }

  return errors
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

function extractErrorMessage(error: unknown): string | undefined {
  if (error && typeof error === 'object' && 'response' in error) {
    const axiosError = error as {
      response?: { data?: { error?: { message?: string } } }
    }
    return axiosError.response?.data?.error?.message
  }
  return undefined
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function CustomerForm({
  open,
  onClose,
  customer,
  onSuccess,
}: CustomerFormProps) {
  const isEdit = !!customer

  const [form, setForm] = useState<FormState>(EMPTY_FORM)
  const [errors, setErrors] = useState<FormErrors>({})

  // Populate or reset form when modal opens
  useEffect(() => {
    if (customer) {
      setForm({
        fullName: customer.fullName,
        idNumber: customer.idNumber,
        idType: customer.idType,
        companyId: customer.companyId,
        phone: customer.phone,
        email: customer.email,
        address: customer.address,
      })
    } else {
      setForm(EMPTY_FORM)
    }
    setErrors({})
  }, [customer, open])

  // Companies for the select — fetch only when modal is open
  const { data: companiesData } = useQuery({
    queryKey: ['companies', 'all'],
    queryFn: () =>
      companiesApi.list({ pageSize: 200 }).then((res) => res.data.data ?? []),
    enabled: open,
    staleTime: 30_000,
  })

  const companyOptions =
    companiesData?.map((c) => ({ value: c.id, label: c.name })) ?? []

  // Mutations
  const createMutation = useMutation({
    mutationFn: (data: CreateCustomerDto) => customersApi.create(data),
    onSuccess: () => {
      toast.success('Customer created successfully.')
      onSuccess()
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error) ?? 'Failed to create customer.'
      toast.error(message)
    },
  })

  const updateMutation = useMutation({
    mutationFn: (data: CreateCustomerDto) =>
      customersApi.update(customer!.id, data),
    onSuccess: () => {
      toast.success('Customer updated successfully.')
      onSuccess()
    },
    onError: (error: unknown) => {
      const message = extractErrorMessage(error) ?? 'Failed to update customer.'
      toast.error(message)
    },
  })

  const isPending = createMutation.isPending || updateMutation.isPending

  // ---- handlers ----

  function handleChange(
    e: React.ChangeEvent<
      HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
    >
  ) {
    const { name, value } = e.target
    setForm((prev) => ({ ...prev, [name]: value }))
    if (errors[name as keyof FormErrors]) {
      setErrors((prev) => ({ ...prev, [name]: undefined }))
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const validationErrors = validate(form)
    if (Object.keys(validationErrors).length > 0) {
      setErrors(validationErrors)
      return
    }

    const payload: CreateCustomerDto = {
      fullName: form.fullName.trim(),
      idNumber: form.idNumber.trim(),
      idType: form.idType as CreateCustomerDto['idType'],
      companyId: form.companyId,
      phone: form.phone.trim(),
      email: form.email.trim(),
      address: form.address.trim(),
    }

    if (isEdit) {
      updateMutation.mutate(payload)
    } else {
      createMutation.mutate(payload)
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit Customer' : 'Add Customer'}
      maxWidth="max-w-xl"
    >
      <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
        {/* Full Name */}
        <Input
          label="Full Name"
          name="fullName"
          id="customer-fullName"
          placeholder="e.g. Budi Santoso"
          value={form.fullName}
          onChange={handleChange}
          error={errors.fullName}
          disabled={isPending}
          autoFocus={!isEdit}
        />

        {/* ID Number + ID Type */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Input
            label="ID Number"
            name="idNumber"
            id="customer-idNumber"
            placeholder="e.g. 3171234567890001"
            value={form.idNumber}
            onChange={handleChange}
            error={errors.idNumber}
            disabled={isPending}
          />
          <Select
            label="ID Type"
            name="idType"
            id="customer-idType"
            placeholder="Select ID type"
            options={ID_TYPE_OPTIONS}
            value={form.idType}
            onChange={handleChange}
            error={errors.idType}
            disabled={isPending}
          />
        </div>

        {/* Company */}
        <Select
          label="Company"
          name="companyId"
          id="customer-companyId"
          placeholder="Select company"
          options={companyOptions}
          value={form.companyId}
          onChange={handleChange}
          error={errors.companyId}
          disabled={isPending}
        />

        {/* Phone + Email */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Input
            label="Phone"
            name="phone"
            id="customer-phone"
            type="tel"
            placeholder="e.g. 08123456789"
            value={form.phone}
            onChange={handleChange}
            error={errors.phone}
            disabled={isPending}
          />
          <Input
            label="Email"
            name="email"
            id="customer-email"
            type="email"
            placeholder="e.g. budi@example.com"
            value={form.email}
            onChange={handleChange}
            error={errors.email}
            disabled={isPending}
          />
        </div>

        {/* Address */}
        <div className="flex flex-col gap-1">
          <label
            htmlFor="customer-address"
            className="text-sm font-medium text-slate-700"
          >
            Address
          </label>
          <textarea
            id="customer-address"
            name="address"
            rows={3}
            placeholder="Street address, city, province"
            value={form.address}
            onChange={handleChange}
            disabled={isPending}
            aria-invalid={errors.address ? 'true' : undefined}
            aria-describedby={
              errors.address ? 'customer-address-error' : undefined
            }
            className={[
              'w-full px-3 py-2 text-sm rounded-lg border bg-white text-slate-800',
              'placeholder:text-slate-400 resize-none transition-colors',
              'focus:outline-none focus:ring-2 focus:ring-offset-0',
              'disabled:bg-slate-50 disabled:text-slate-400 disabled:cursor-not-allowed',
              errors.address
                ? 'border-red-400 focus:ring-red-400 focus:border-red-400'
                : 'border-slate-300 focus:ring-sky-500 focus:border-sky-500',
            ]
              .filter(Boolean)
              .join(' ')}
          />
          {errors.address && (
            <p
              id="customer-address-error"
              role="alert"
              className="text-xs text-red-600"
            >
              {errors.address}
            </p>
          )}
        </div>

        {/* Footer actions */}
        <div className="flex justify-end gap-2 pt-2">
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isPending}
          >
            Cancel
          </Button>
          <Button type="submit" loading={isPending}>
            {isEdit ? 'Save Changes' : 'Create Customer'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
