import api from '../lib/axios'
import type { ApiResponse, AuthTokens, User } from '../types'

export const authApi = {
  login: (email: string, password: string) =>
    api.post<ApiResponse<{ user: User; tokens: AuthTokens }>>('/auth/login', {
      email,
      password,
    }),

  refresh: (refreshToken: string) =>
    api.post<ApiResponse<AuthTokens>>('/auth/refresh', {
      refresh_token: refreshToken,
    }),

  logout: (refreshToken: string) =>
    api.post<ApiResponse<null>>('/auth/logout', {
      refresh_token: refreshToken,
    }),

  forgotPassword: (email: string) =>
    api.post<ApiResponse<{ reset_token: string }>>('/auth/forgot-password', {
      email,
    }),

  resetPassword: (email: string, token: string, newPassword: string) =>
    api.post<ApiResponse<null>>('/auth/reset-password', {
      email,
      token,
      new_password: newPassword,
    }),
}
