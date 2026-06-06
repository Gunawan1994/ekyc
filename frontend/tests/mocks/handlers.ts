import { http, HttpResponse } from 'msw'

// -------------------------------------------------------
// Auth handlers
// -------------------------------------------------------
export const authHandlers = [
  http.post('/api/v1/auth/login', async ({ request }) => {
    const body = (await request.json()) as { email: string; password: string }

    if (body.email === 'admin@example.com' && body.password === 'Admin123!') {
      return HttpResponse.json({
        success: true,
        data: {
          access_token: 'mock-access-token',
          refresh_token: 'mock-refresh-token',
          user: {
            id: '1',
            email: 'admin@example.com',
            name: 'Admin User',
            role: 'admin',
          },
        },
        message: 'Login successful',
      })
    }

    return HttpResponse.json(
      {
        success: false,
        data: null,
        message: 'Invalid email or password',
      },
      { status: 401 },
    )
  }),
]

// -------------------------------------------------------
// Dashboard handlers
// -------------------------------------------------------
export const dashboardHandlers = [
  http.get('/api/v1/dashboard/stats', () => {
    return HttpResponse.json({
      success: true,
      data: {
        total_customers: 1240,
        total_companies: 87,
        pending_kyc: 34,
        pending_kyb: 12,
        approved_kyc: 980,
        approved_kyb: 65,
        rejected_kyc: 226,
        rejected_kyb: 10,
      },
      message: 'Stats retrieved successfully',
    })
  }),
]

// -------------------------------------------------------
// Combined default handlers
// -------------------------------------------------------
export const handlers = [...authHandlers, ...dashboardHandlers]
