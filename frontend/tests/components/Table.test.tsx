import { describe, it, expect } from 'vitest'
import { render, screen, within } from '@testing-library/react'


// ---------------------------------------------------------------------------
// Inline Table component definition
// ---------------------------------------------------------------------------
// Source component lives at src/components/ui/Table.tsx.
// Replace with real import once the source file exists:
//   import Table from '@/components/ui/Table'
// ---------------------------------------------------------------------------

interface Column<T> {
  key: string
  header: string
  render?: (value: unknown, row: T) => React.ReactNode
}

interface TableProps<T extends Record<string, unknown>> {
  columns: Column<T>[]
  data: T[]
  isLoading?: boolean
  emptyMessage?: string
}

const SKELETON_ROWS = 5

function Table<T extends Record<string, unknown>>({
  columns,
  data,
  isLoading = false,
  emptyMessage = 'No data available',
}: TableProps<T>) {
  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200" role="table">
        <thead className="bg-gray-50">
          <tr>
            {columns.map((col) => (
              <th
                key={col.key}
                scope="col"
                className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
              >
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-200 bg-white">
          {isLoading ? (
            Array.from({ length: SKELETON_ROWS }).map((_, rowIdx) => (
              <tr key={`skeleton-${rowIdx}`} aria-label="loading-row">
                {columns.map((col) => (
                  <td key={col.key} className="px-6 py-4">
                    <div
                      className="h-4 animate-pulse rounded bg-gray-200"
                      aria-hidden="true"
                      data-testid="skeleton-cell"
                    />
                  </td>
                ))}
              </tr>
            ))
          ) : data.length === 0 ? (
            <tr>
              <td
                colSpan={columns.length}
                className="px-6 py-10 text-center text-sm text-gray-500"
                data-testid="empty-state"
              >
                {emptyMessage}
              </td>
            </tr>
          ) : (
            data.map((row, rowIdx) => (
              <tr key={rowIdx} data-testid="table-row">
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className="whitespace-nowrap px-6 py-4 text-sm text-gray-900"
                    data-testid={`cell-${col.key}`}
                  >
                    {col.render
                      ? col.render(row[col.key], row)
                      : String(row[col.key] ?? '')}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

interface CustomerRow extends Record<string, unknown> {
  id: string
  name: string
  email: string
  status: string
}

const columns: Column<CustomerRow>[] = [
  { key: 'id', header: 'ID' },
  { key: 'name', header: 'Name' },
  { key: 'email', header: 'Email' },
  { key: 'status', header: 'Status' },
]

const sampleData: CustomerRow[] = [
  {
    id: '1',
    name: 'Budi Santoso',
    email: 'budi@example.com',
    status: 'pending',
  },
  {
    id: '2',
    name: 'Sari Dewi',
    email: 'sari@example.com',
    status: 'approved',
  },
]

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('Table', () => {
  it('renders table headers from columns config', () => {
    render(<Table columns={columns} data={sampleData} />)

    expect(screen.getByText('ID')).toBeInTheDocument()
    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByText('Email')).toBeInTheDocument()
    expect(screen.getByText('Status')).toBeInTheDocument()
  })

  it('renders row data correctly', () => {
    render(<Table columns={columns} data={sampleData} />)

    const rows = screen.getAllByTestId('table-row')
    expect(rows).toHaveLength(2)

    // First row
    const firstRow = rows[0]
    expect(within(firstRow).getByTestId('cell-name')).toHaveTextContent(
      'Budi Santoso',
    )
    expect(within(firstRow).getByTestId('cell-email')).toHaveTextContent(
      'budi@example.com',
    )
    expect(within(firstRow).getByTestId('cell-status')).toHaveTextContent(
      'pending',
    )

    // Second row
    const secondRow = rows[1]
    expect(within(secondRow).getByTestId('cell-name')).toHaveTextContent(
      'Sari Dewi',
    )
    expect(within(secondRow).getByTestId('cell-status')).toHaveTextContent(
      'approved',
    )
  })

  it('renders custom cell via render function', () => {
    const columnsWithRender: Column<CustomerRow>[] = [
      ...columns.filter((c) => c.key !== 'status'),
      {
        key: 'status',
        header: 'Status',
        render: (value) => (
          <span data-testid="custom-status">{String(value).toUpperCase()}</span>
        ),
      },
    ]

    render(<Table columns={columnsWithRender} data={[sampleData[0]]} />)

    expect(screen.getByTestId('custom-status')).toHaveTextContent('PENDING')
  })

  it('shows loading skeleton when isLoading=true', () => {
    render(<Table columns={columns} data={[]} isLoading />)

    // Should not show empty state
    expect(screen.queryByTestId('empty-state')).not.toBeInTheDocument()

    // Should show skeleton rows (SKELETON_ROWS * columns.length = 5 * 4 = 20 cells)
    const skeletonCells = screen.getAllByTestId('skeleton-cell')
    expect(skeletonCells.length).toBe(SKELETON_ROWS * columns.length)

    // Skeleton rows should have aria-label for accessibility
    const loadingRows = screen.getAllByLabelText('loading-row')
    expect(loadingRows).toHaveLength(SKELETON_ROWS)
  })

  it('shows empty state when data is empty array', () => {
    render(<Table columns={columns} data={[]} />)

    const emptyState = screen.getByTestId('empty-state')
    expect(emptyState).toBeInTheDocument()
    expect(emptyState).toHaveTextContent('No data available')

    // No rows rendered
    expect(screen.queryAllByTestId('table-row')).toHaveLength(0)
  })

  it('shows custom empty message when provided', () => {
    render(
      <Table
        columns={columns}
        data={[]}
        emptyMessage="No customers found matching your search."
      />,
    )

    expect(screen.getByTestId('empty-state')).toHaveTextContent(
      'No customers found matching your search.',
    )
  })
})
