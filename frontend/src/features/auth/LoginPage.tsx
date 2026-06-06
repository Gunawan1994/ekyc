import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation } from '@tanstack/react-query'
import { Eye, EyeOff, ShieldCheck } from 'lucide-react'
import toast from 'react-hot-toast'
import { useAuth } from '../../auth/useAuth'
import { Button } from '../../components/ui/Button'
import { Input } from '../../components/ui/Input'

interface LoginForm {
  email: string
  password: string
}

interface FormErrors {
  email?: string
  password?: string
}

function validate(form: LoginForm): FormErrors {
  const errors: FormErrors = {}
  if (!form.email.trim()) {
    errors.email = 'Email is required.'
  } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    errors.email = 'Enter a valid email address.'
  }
  if (!form.password) {
    errors.password = 'Password is required.'
  }
  return errors
}

export function LoginPage() {
  const navigate = useNavigate()
  const { login } = useAuth()

  const [form, setForm] = useState<LoginForm>({ email: '', password: '' })
  const [errors, setErrors] = useState<FormErrors>({})
  const [showPassword, setShowPassword] = useState(false)

  const mutation = useMutation({
    mutationFn: ({ email, password }: LoginForm) => login(email, password),
    onSuccess: () => {
      navigate('/dashboard', { replace: true })
    },
    onError: (error: unknown) => {
      let message = 'Login failed. Please check your credentials.'
      if (error && typeof error === 'object' && 'response' in error) {
        const axiosError = error as {
          response?: { data?: { error?: { message?: string } } }
        }
        const serverMessage = axiosError.response?.data?.error?.message
        if (serverMessage) message = serverMessage
      }
      toast.error(message)
    },
  })

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
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
    mutation.mutate(form)
  }

  return (
    <div className="min-h-screen bg-slate-100 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        {/* Brand header */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-sky-600 mb-4 shadow-lg shadow-sky-200">
            <ShieldCheck size={28} className="text-white" aria-hidden="true" />
          </div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight">
            eKYC Platform
          </h1>
          <p className="mt-1 text-sm text-slate-500">Verification Management System</p>
        </div>

        {/* Login card */}
        <div className="bg-white rounded-2xl border border-slate-200 shadow-sm p-8">
          <h2 className="text-base font-semibold text-slate-800 mb-6">
            Sign in to your account
          </h2>

          <form onSubmit={handleSubmit} noValidate className="flex flex-col gap-5">
            <Input
              label="Email address"
              id="email"
              name="email"
              type="email"
              autoComplete="email"
              autoFocus
              placeholder="you@example.com"
              value={form.email}
              onChange={handleChange}
              error={errors.email}
              disabled={mutation.isPending}
            />

            {/* Password with show/hide toggle */}
            <div className="flex flex-col gap-1">
              <label
                htmlFor="password"
                className="text-sm font-medium text-slate-700"
              >
                Password
              </label>
              <div className="relative">
                <input
                  id="password"
                  name="password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="current-password"
                  placeholder="Enter your password"
                  value={form.password}
                  onChange={handleChange}
                  disabled={mutation.isPending}
                  aria-invalid={errors.password ? 'true' : undefined}
                  aria-describedby={errors.password ? 'password-error' : undefined}
                  className={[
                    'w-full px-3 py-2 pr-10 text-sm rounded-lg border bg-white text-slate-800',
                    'placeholder:text-slate-400 transition-colors',
                    'focus:outline-none focus:ring-2 focus:ring-offset-0',
                    'disabled:bg-slate-50 disabled:text-slate-400 disabled:cursor-not-allowed',
                    errors.password
                      ? 'border-red-400 focus:ring-red-400 focus:border-red-400'
                      : 'border-slate-300 focus:ring-sky-500 focus:border-sky-500',
                  ]
                    .filter(Boolean)
                    .join(' ')}
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  disabled={mutation.isPending}
                  className="absolute right-2.5 top-1/2 -translate-y-1/2 p-1 rounded
                    text-slate-400 hover:text-slate-600 transition-colors
                    focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-sky-500
                    disabled:cursor-not-allowed"
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                >
                  {showPassword ? (
                    <EyeOff size={16} aria-hidden="true" />
                  ) : (
                    <Eye size={16} aria-hidden="true" />
                  )}
                </button>
              </div>
              {errors.password && (
                <p id="password-error" role="alert" className="text-xs text-red-600">
                  {errors.password}
                </p>
              )}
            </div>

            <Button
              type="submit"
              size="lg"
              loading={mutation.isPending}
              className="w-full mt-1"
            >
              {mutation.isPending ? 'Signing in…' : 'Sign in'}
            </Button>
          </form>
        </div>

        <p className="text-center text-xs text-slate-400 mt-6">
          &copy; {new Date().getFullYear()} PT Sun Energy. All rights reserved.
        </p>
      </div>
    </div>
  )
}

export default LoginPage
