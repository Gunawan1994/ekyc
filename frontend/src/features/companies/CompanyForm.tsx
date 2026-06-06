import { useEffect, useState } from 'react'
import { Modal } from '../../components/ui/Modal'
import { Input } from '../../components/ui/Input'
import { Button } from '../../components/ui/Button'
import type { Company } from '../../types'
import type { CreateCompanyDto, UpdateCompanyDto } from '../../api/companies'

interface CompanyFormProps {
  open: boolean
  onClose: () => void
  onSubmit: (data: CreateCompanyDto | UpdateCompanyDto) => void
  isSubmitting: boolean
  initialValues?: Company
}

interface FormFields {
  name: string
  registrationNumber: string
  address: string
  phone: string
  email: string
}

interface FormErrors {
  name?: string
  registrationNumber?: string
  address?: string
  phone?: string
  email?: string
}

function validateForm(fields: FormFields): FormErrors {
  const errors: FormErrors = {}

  if (!fields.name.trim()) {
    errors.name = 'Company name is required.'
  }
  if (!fields.registrationNumber.trim()) {
    errors.registrationNumber = 'Registration number is required.'
  }
  if (!fields.address.trim()) {
    errors.address = 'Address is required.'
  }
  if (!fields.phone.trim()) {
    errors.phone = 'Phone is required.'
  }
  if (!fields.email.trim()) {
    errors.email = 'Email is required.'
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(fields.email)) {
    errors.email = 'Enter a valid email address.'
  }

  return errors
}

const emptyFields: FormFields = {
  name: '',
  registrationNumber: '',
  address: '',
  phone: '',
  email: '',
}

export function CompanyForm({
  open,
  onClose,
  onSubmit,
  isSubmitting,
  initialValues,
}: CompanyFormProps) {
  const isEdit = !!initialValues

  const [fields, setFields] = useState<FormFields>(emptyFields)
  const [errors, setErrors] = useState<FormErrors>({})

  // Sync form state when the modal opens or the target record changes.
  useEffect(() => {
    if (!open) return
    if (initialValues) {
      setFields({
        name: initialValues.name,
        registrationNumber: initialValues.registrationNumber,
        address: initialValues.address,
        phone: initialValues.phone,
        email: initialValues.email,
      })
    } else {
      setFields(emptyFields)
    }
    setErrors({})
  }, [open, initialValues])

  function handleChange(
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) {
    const { name, value } = e.target
    setFields((prev) => ({ ...prev, [name]: value }))
    if (errors[name as keyof FormErrors]) {
      setErrors((prev) => ({ ...prev, [name]: undefined }))
    }
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const validation = validateForm(fields)
    if (Object.keys(validation).length > 0) {
      setErrors(validation)
      return
    }
    onSubmit({ ...fields })
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? 'Edit Company' : 'Add Company'}
      maxWidth="max-w-lg"
    >
      <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-4">
        <Input
          label="Company Name"
          name="name"
          value={fields.name}
          onChange={handleChange}
          error={errors.name}
          placeholder="PT. Example Indonesia"
          autoFocus
          disabled={isSubmitting}
        />

        <Input
          label="Registration Number"
          name="registrationNumber"
          value={fields.registrationNumber}
          onChange={handleChange}
          error={errors.registrationNumber}
          placeholder="e.g. 1234567890"
          disabled={isSubmitting}
        />

        <div className="flex flex-col gap-1">
          <label
            htmlFor="company-address"
            className="text-sm font-medium text-slate-700"
          >
            Address
          </label>
          <textarea
            id="company-address"
            name="address"
            value={fields.address}
            onChange={handleChange}
            rows={3}
            placeholder="Full business address"
            disabled={isSubmitting}
            aria-invalid={errors.address ? 'true' : undefined}
            aria-describedby={
              errors.address ? 'company-address-error' : undefined
            }
            className={[
              'w-full px-3 py-2 text-sm rounded-lg border bg-white text-slate-800 placeholder:text-slate-400 resize-none',
              'focus:outline-none focus:ring-2 focus:ring-offset-0',
              'disabled:bg-slate-50 disabled:text-slate-400 disabled:cursor-not-allowed',
              'transition-colors',
              errors.address
                ? 'border-red-400 focus:ring-red-400 focus:border-red-400'
                : 'border-slate-300 focus:ring-sky-500 focus:border-sky-500',
            ].join(' ')}
          />
          {errors.address && (
            <p
              id="company-address-error"
              role="alert"
              className="text-xs text-red-600"
            >
              {errors.address}
            </p>
          )}
        </div>

        <Input
          label="Phone"
          name="phone"
          type="tel"
          value={fields.phone}
          onChange={handleChange}
          error={errors.phone}
          placeholder="+62 21 1234 5678"
          disabled={isSubmitting}
        />

        <Input
          label="Email"
          name="email"
          type="email"
          value={fields.email}
          onChange={handleChange}
          error={errors.email}
          placeholder="contact@company.com"
          disabled={isSubmitting}
        />

        <div className="flex justify-end gap-3 pt-2 border-t border-slate-100">
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isSubmitting}
          >
            Cancel
          </Button>
          <Button type="submit" variant="primary" loading={isSubmitting}>
            {isEdit ? 'Save Changes' : 'Create Company'}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
