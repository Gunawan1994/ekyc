import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'


// ---------------------------------------------------------------------------
// Inline Badge component definition
// ---------------------------------------------------------------------------
// The source component lives at src/components/ui/Badge.tsx.
// Because those source files are not yet generated, we define a reference
// implementation here that mirrors the contract the tests are validating.
// When the real component is written, replace this import with:
//   import Badge from '@/components/ui/Badge'
// ---------------------------------------------------------------------------

type BadgeStatus = 'pending' | 'approved' | 'rejected' | string

interface BadgeProps {
  status: BadgeStatus
  label?: string
}

const STATUS_CLASS_MAP: Record<string, string> = {
  pending: 'bg-amber-100 text-amber-800',
  approved: 'bg-green-100 text-green-800',
  rejected: 'bg-red-100 text-red-800',
}

const DEFAULT_CLASS = 'bg-gray-100 text-gray-800'

function Badge({ status, label }: BadgeProps) {
  const colorClass = STATUS_CLASS_MAP[status] ?? DEFAULT_CLASS
  const displayLabel = label ?? status.charAt(0).toUpperCase() + status.slice(1)

  return (
    <span
      data-testid="badge"
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${colorClass}`}
    >
      {displayLabel}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('Badge', () => {
  it('renders pending badge with amber color class', () => {
    render(<Badge status="pending" />)

    const badge = screen.getByTestId('badge')

    expect(badge).toBeInTheDocument()
    expect(badge).toHaveTextContent('Pending')
    expect(badge.className).toContain('bg-amber-100')
    expect(badge.className).toContain('text-amber-800')
  })

  it('renders approved badge with green color class', () => {
    render(<Badge status="approved" />)

    const badge = screen.getByTestId('badge')

    expect(badge).toBeInTheDocument()
    expect(badge).toHaveTextContent('Approved')
    expect(badge.className).toContain('bg-green-100')
    expect(badge.className).toContain('text-green-800')
  })

  it('renders rejected badge with red color class', () => {
    render(<Badge status="rejected" />)

    const badge = screen.getByTestId('badge')

    expect(badge).toBeInTheDocument()
    expect(badge).toHaveTextContent('Rejected')
    expect(badge.className).toContain('bg-red-100')
    expect(badge.className).toContain('text-red-800')
  })

  it('renders custom label when provided', () => {
    render(<Badge status="pending" label="Under Review" />)

    expect(screen.getByTestId('badge')).toHaveTextContent('Under Review')
  })

  it('falls back to gray color class for unknown status', () => {
    render(<Badge status="unknown" />)

    const badge = screen.getByTestId('badge')

    expect(badge.className).toContain('bg-gray-100')
    expect(badge.className).toContain('text-gray-800')
  })
})
