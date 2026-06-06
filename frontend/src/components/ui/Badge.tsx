

export type BadgeVariant = 'pending' | 'in_review' | 'approved' | 'rejected' | 'active' | 'inactive'

interface BadgeProps {
  variant: BadgeVariant
  label?: string
  className?: string
}

const variantClasses: Record<BadgeVariant, string> = {
  pending: 'bg-amber-100 text-amber-800',
  in_review: 'bg-sky-100 text-sky-800',
  approved: 'bg-emerald-100 text-emerald-800',
  rejected: 'bg-red-100 text-red-800',
  active: 'bg-emerald-100 text-emerald-800',
  inactive: 'bg-slate-100 text-slate-600',
}

const defaultLabels: Record<BadgeVariant, string> = {
  pending: 'Pending',
  in_review: 'In Review',
  approved: 'Approved',
  rejected: 'Rejected',
  active: 'Active',
  inactive: 'Inactive',
}

export function Badge({ variant, label, className = '' }: BadgeProps) {
  return (
    <span
      className={[
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
        variantClasses[variant],
        className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      {label ?? defaultLabels[variant]}
    </span>
  )
}
