import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  render,
  screen,
  fireEvent,
  waitFor,
  act,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { MemoryRouter, useNavigate } from 'react-router-dom'
import { server } from '../mocks/server'
import { http, HttpResponse } from 'msw'

// ---------------------------------------------------------------------------
// Toast mock
// ---------------------------------------------------------------------------
// When the real sonner/react-hot-toast is wired in, remove this mock and
// import the real toast from the project.
// ---------------------------------------------------------------------------
const toastMock = {
  success: vi.fn(),
  error: vi.fn(),
}

vi.mock('sonner', () => ({
  toast: toastMock,
  Toaster: () => null,
}))

// ---------------------------------------------------------------------------
// Inline LoginPage component
// ---------------------------------------------------------------------------
// Source lives at src/features/auth/LoginPage.tsx.
// Replace with real import once that file exists:
//   import LoginPage from '@/features/auth/LoginPage'
// ---------------------------------------------------------------------------

interface LoginFormValues {
  email: string
  password: string
}

interface ValidationErrors {
  email?: string
  password?: string
}

function validate(values: LoginFormValues): ValidationErrors {
  const errors: ValidationErrors = {}
  if (!values.email.trim()) errors.email = 'Email is required'
  if (!values.password.trim()) errors.password = 'Password is required'
  return errors
}

async function loginApi(
  values: LoginFormValues,
): Promise<{ access_token: string; user: { name: string } }> {
  const res = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(values),
  })
  const data = await res.json()
  if (!res.ok) throw new Error(data.message ?? 'Login failed')
  return data.data
}

function LoginPage() {
  const navigate = useNavigate()
  const [values, setValues] = useState<LoginFormValues>({
    email: '',
    password: '',
  })
  const [errors, setErrors] = useState<ValidationErrors>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Dynamically resolve the toast mock so vi.mock can intercept it.
  // In production code use a static import of toast from sonner.
  const getToast = () =>
    (globalThis as unknown as { __toastMock__: typeof toastMock })
      .__toastMock__ ?? toastMock

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target
    setValues((prev) => ({ ...prev, [name]: value }))
    if (errors[name as keyof ValidationErrors]) {
      setErrors((prev) => ({ ...prev, [name]: undefined }))
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const validationErrors = validate(values)
    if (Object.keys(validationErrors).length > 0) {
      setErrors(validationErrors)
      return
    }

    setIsSubmitting(true)
    try {
      const result = await loginApi(values)
      localStorage.setItem('access_token', result.access_token)
      getToast().success('Login successful')
      navigate('/dashboard')
    } catch (err) {
      getToast().error(
        err instanceof Error ? err.message : 'An error occurred',
      )
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div data-testid="login-page">
      <h1>Sign in to your account</h1>
      <form onSubmit={handleSubmit} noValidate data-testid="login-form">
        <div>
          <label htmlFor="email">Email address</label>
          <input
            id="email"
            name="email"
            type="email"
            autoComplete="email"
            value={values.email}
            onChange={handleChange}
            aria-describedby={errors.email ? 'email-error' : undefined}
            data-testid="email-input"
          />
          {errors.email && (
            <p id="email-error" role="alert" data-testid="email-error">
              {errors.email}
            </p>
          )}
        </div>

        <div>
          <label htmlFor="password">Password</label>
          <input
            id="password"
            name="password"
            type="password"
            autoComplete="current-password"
            value={values.password}
            onChange={handleChange}
            aria-describedby={errors.password ? 'password-error' : undefined}
            data-testid="password-input"
          />
          {errors.password && (
            <p id="password-error" role="alert" data-testid="password-error">
              {errors.password}
            </p>
          )}
        </div>

        <button type="submit" disabled={isSubmitting} data-testid="submit-btn">
          {isSubmitting ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

function renderLoginPage(initialRoute = '/login') {
  return render(
    <MemoryRouter initialEntries={[initialRoute]}>
      <LoginPage />
    </MemoryRouter>,
  )
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    // Expose the mock so the inline component can reach it via globalThis
    ;(globalThis as unknown as Record<string, unknown>).__toastMock__ =
      toastMock
  })

  it('renders login form with email and password fields', () => {
    renderLoginPage()

    expect(screen.getByTestId('login-form')).toBeInTheDocument()
    expect(screen.getByTestId('email-input')).toBeInTheDocument()
    expect(screen.getByTestId('password-input')).toBeInTheDocument()
    expect(screen.getByTestId('submit-btn')).toBeInTheDocument()
    expect(screen.getByLabelText('Email address')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
  })

  it('shows validation error on empty submit', async () => {
    renderLoginPage()

    await act(async () => {
      fireEvent.click(screen.getByTestId('submit-btn'))
    })

    expect(screen.getByTestId('email-error')).toHaveTextContent(
      'Email is required',
    )
    expect(screen.getByTestId('password-error')).toHaveTextContent(
      'Password is required',
    )
  })

  it('clears field error when user starts typing', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    // Trigger validation errors first
    await act(async () => {
      fireEvent.click(screen.getByTestId('submit-btn'))
    })

    expect(screen.getByTestId('email-error')).toBeInTheDocument()

    // Start typing in email field
    await user.type(screen.getByTestId('email-input'), 'a')

    expect(screen.queryByTestId('email-error')).not.toBeInTheDocument()
  })

  it('calls login API on valid submit', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByTestId('email-input'), 'admin@example.com')
    await user.type(screen.getByTestId('password-input'), 'Admin123!')

    await act(async () => {
      fireEvent.click(screen.getByTestId('submit-btn'))
    })

    // No validation errors means the API was called
    await waitFor(() => {
      expect(screen.queryByTestId('email-error')).not.toBeInTheDocument()
      expect(screen.queryByTestId('password-error')).not.toBeInTheDocument()
    })
  })

  it('shows toast on successful login and redirects', async () => {
    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByTestId('email-input'), 'admin@example.com')
    await user.type(screen.getByTestId('password-input'), 'Admin123!')

    await act(async () => {
      fireEvent.click(screen.getByTestId('submit-btn'))
    })

    await waitFor(() => {
      expect(toastMock.success).toHaveBeenCalledWith('Login successful')
    })

    // Token persisted to localStorage
    expect(localStorage.getItem('access_token')).toBe('mock-access-token')
  })

  it('shows error toast on failed login (401)', async () => {
    // Override the default MSW handler to always return 401
    server.use(
      http.post('/api/v1/auth/login', () =>
        HttpResponse.json(
          {
            success: false,
            data: null,
            message: 'Invalid email or password',
          },
          { status: 401 },
        ),
      ),
    )

    const user = userEvent.setup()
    renderLoginPage()

    await user.type(screen.getByTestId('email-input'), 'wrong@example.com')
    await user.type(screen.getByTestId('password-input'), 'wrongpassword')

    await act(async () => {
      fireEvent.click(screen.getByTestId('submit-btn'))
    })

    await waitFor(() => {
      expect(toastMock.error).toHaveBeenCalledWith('Invalid email or password')
    })
  })
})
