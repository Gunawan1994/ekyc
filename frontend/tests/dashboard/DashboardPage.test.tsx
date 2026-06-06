import { describe, it, expect } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

import {
  QueryClient,
  QueryClientProvider,
  useQuery,
} from '@tanstack/react-query'
import { server } from '../mocks/server'
import { http, HttpResponse } from 'msw'

// ---------------------------------------------------------------------------
// Inline DashboardPage component
// ---------------------------------------------------------------------------
// Source lives at src/features/dashboard/DashboardPage.tsx.
// Replace with real import once that file exists:
//   import DashboardPage from '@/features/dashboard/DashboardPage'
// ---------------------------------------------------------------------------

interface DashboardStats {
  total_customers: number
  total_companies: number
  pending_kyc: number
  pending_kyb: number
  approved_kyc: number
  approved_kyb: number
  rejected_kyc: number
  rejected_kyb: number
}

interface ApiResponse<T> {
  success: boolean
  data: T
  message: string
}

async function fetchDashboardStats(): Promise<DashboardStats> {
  const res = await fetch('/api/v1/dashboard/stats')
  if (!res.ok) throw new Error('Failed to fetch dashboard stats')
  const json: ApiResponse<DashboardStats> = await res.json()
  return json.data
}

interface StatCardProps {
  title: string
  value: number | string
  testId?: string
}

function StatCard({ title, value, testId }: StatCardProps) {
  return (
    <div
      className="rounded-lg bg-white p-6 shadow"
      data-testid={testId ?? 'stat-card'}
    >
      <p className="text-sm font-medium text-gray-500">{title}</p>
      <p
        className="mt-2 text-3xl font-bold text-gray-900"
        data-testid={`${testId ?? 'stat-card'}-value`}
      >
        {value}
      </p>
    </div>
  )
}

function StatCardSkeleton() {
  return (
    <div
      className="rounded-lg bg-white p-6 shadow"
      data-testid="stat-card-skeleton"
    >
      <div className="h-4 w-1/2 animate-pulse rounded bg-gray-200" />
      <div className="mt-2 h-8 w-1/3 animate-pulse rounded bg-gray-300" />
    </div>
  )
}

function DashboardPage() {
  const {
    data: stats,
    isLoading,
    isError,
  } = useQuery<DashboardStats>({
    queryKey: ['dashboard', 'stats'],
    queryFn: fetchDashboardStats,
  })

  if (isLoading) {
    return (
      <main data-testid="dashboard-page">
        <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
        <div
          className="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4"
          data-testid="stats-grid-loading"
        >
          {Array.from({ length: 8 }).map((_, i) => (
            <StatCardSkeleton key={i} />
          ))}
        </div>
      </main>
    )
  }

  if (isError || !stats) {
    return (
      <main data-testid="dashboard-page">
        <p data-testid="error-state">Failed to load dashboard statistics.</p>
      </main>
    )
  }

  return (
    <main data-testid="dashboard-page">
      <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
      <div
        className="mt-6 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4"
        data-testid="stats-grid"
      >
        <StatCard
          title="Total Customers"
          value={stats.total_customers}
          testId="stat-total-customers"
        />
        <StatCard
          title="Total Companies"
          value={stats.total_companies}
          testId="stat-total-companies"
        />
        <StatCard
          title="Pending eKYC"
          value={stats.pending_kyc}
          testId="stat-pending-kyc"
        />
        <StatCard
          title="Pending eKYB"
          value={stats.pending_kyb}
          testId="stat-pending-kyb"
        />
        <StatCard
          title="Approved eKYC"
          value={stats.approved_kyc}
          testId="stat-approved-kyc"
        />
        <StatCard
          title="Approved eKYB"
          value={stats.approved_kyb}
          testId="stat-approved-kyb"
        />
        <StatCard
          title="Rejected eKYC"
          value={stats.rejected_kyc}
          testId="stat-rejected-kyc"
        />
        <StatCard
          title="Rejected eKYB"
          value={stats.rejected_kyb}
          testId="stat-rejected-kyb"
        />
      </div>
    </main>
  )
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

function makeQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        // Disable retries so failures surface immediately in tests
        retry: false,
        // No stale time to ensure we always hit the mock
        staleTime: 0,
      },
    },
  })
}

function renderDashboard() {
  const queryClient = makeQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <DashboardPage />
    </QueryClientProvider>,
  )
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('DashboardPage', () => {
  it('shows loading state initially', () => {
    renderDashboard()

    // The loading skeleton cards should be visible before the fetch resolves
    const skeletons = screen.getAllByTestId('stat-card-skeleton')
    expect(skeletons.length).toBeGreaterThan(0)

    // The real stats grid should not yet be present
    expect(screen.queryByTestId('stats-grid')).not.toBeInTheDocument()
  })

  it('renders stat cards with correct values after fetch', async () => {
    renderDashboard()

    // Wait for the loading state to clear and stat cards to appear
    await waitFor(() => {
      expect(screen.getByTestId('stats-grid')).toBeInTheDocument()
    })

    // Values match the MSW mock in tests/mocks/handlers.ts
    expect(
      screen.getByTestId('stat-total-customers-value'),
    ).toHaveTextContent('1240')
    expect(
      screen.getByTestId('stat-total-companies-value'),
    ).toHaveTextContent('87')
    expect(screen.getByTestId('stat-pending-kyc-value')).toHaveTextContent('34')
    expect(screen.getByTestId('stat-pending-kyb-value')).toHaveTextContent('12')
    expect(
      screen.getByTestId('stat-approved-kyc-value'),
    ).toHaveTextContent('980')
    expect(
      screen.getByTestId('stat-approved-kyb-value'),
    ).toHaveTextContent('65')
    expect(
      screen.getByTestId('stat-rejected-kyc-value'),
    ).toHaveTextContent('226')
    expect(screen.getByTestId('stat-rejected-kyb-value')).toHaveTextContent(
      '10',
    )
  })

  it('renders the correct stat card titles', async () => {
    renderDashboard()

    await waitFor(() => {
      expect(screen.getByTestId('stats-grid')).toBeInTheDocument()
    })

    expect(screen.getByText('Total Customers')).toBeInTheDocument()
    expect(screen.getByText('Total Companies')).toBeInTheDocument()
    expect(screen.getByText('Pending eKYC')).toBeInTheDocument()
    expect(screen.getByText('Pending eKYB')).toBeInTheDocument()
    expect(screen.getByText('Approved eKYC')).toBeInTheDocument()
    expect(screen.getByText('Approved eKYB')).toBeInTheDocument()
    expect(screen.getByText('Rejected eKYC')).toBeInTheDocument()
    expect(screen.getByText('Rejected eKYB')).toBeInTheDocument()
  })

  it('shows error state when the API returns a non-ok response', async () => {
    server.use(
      http.get('/api/v1/dashboard/stats', () =>
        HttpResponse.json(
          { success: false, data: null, message: 'Internal Server Error' },
          { status: 500 },
        ),
      ),
    )

    renderDashboard()

    await waitFor(() => {
      expect(screen.getByTestId('error-state')).toBeInTheDocument()
    })

    expect(screen.getByTestId('error-state')).toHaveTextContent(
      'Failed to load dashboard statistics.',
    )
  })
})
