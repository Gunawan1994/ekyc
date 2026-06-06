
import { ChevronLeft, ChevronRight } from 'lucide-react'

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

function getPageNumbers(current: number, total: number): (number | '...')[] {
  if (total <= 7) {
    return Array.from({ length: total }, (_, i) => i + 1)
  }

  const pages: (number | '...')[] = [1]

  if (current > 3) pages.push('...')

  const start = Math.max(2, current - 1)
  const end = Math.min(total - 1, current + 1)

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }

  if (current < total - 2) pages.push('...')

  pages.push(total)

  return pages
}

export function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null

  const pages = getPageNumbers(page, totalPages)

  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <p className="text-sm text-slate-500">
        Page <span className="font-medium text-slate-700">{page}</span> of{' '}
        <span className="font-medium text-slate-700">{totalPages}</span>
      </p>

      <nav aria-label="Pagination" className="flex items-center gap-1">
        {/* Previous */}
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-slate-600 rounded-lg
            hover:bg-slate-100 disabled:opacity-40 disabled:cursor-not-allowed
            focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 transition-colors"
          aria-label="Previous page"
        >
          <ChevronLeft size={16} />
          Previous
        </button>

        {/* Page numbers */}
        <div className="hidden sm:flex items-center gap-1">
          {pages.map((p, idx) =>
            p === '...' ? (
              <span
                key={`ellipsis-${idx}`}
                className="px-3 py-1.5 text-sm text-slate-400 select-none"
                aria-hidden="true"
              >
                &hellip;
              </span>
            ) : (
              <button
                key={p}
                onClick={() => onPageChange(p as number)}
                aria-current={p === page ? 'page' : undefined}
                className={[
                  'min-w-[2.25rem] px-3 py-1.5 text-sm font-medium rounded-lg transition-colors',
                  'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500',
                  p === page
                    ? 'bg-sky-600 text-white'
                    : 'text-slate-600 hover:bg-slate-100',
                ].join(' ')}
              >
                {p}
              </button>
            )
          )}
        </div>

        {/* Next */}
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className="flex items-center gap-1 px-3 py-1.5 text-sm font-medium text-slate-600 rounded-lg
            hover:bg-slate-100 disabled:opacity-40 disabled:cursor-not-allowed
            focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 transition-colors"
          aria-label="Next page"
        >
          Next
          <ChevronRight size={16} />
        </button>
      </nav>
    </div>
  )
}
