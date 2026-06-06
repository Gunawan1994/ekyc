import api from '../lib/axios'
import type { ApiResponse, DashboardStats } from '../types'

export const dashboardApi = {
  getStats: () =>
    api.get<ApiResponse<DashboardStats>>('/dashboard/stats'),
}
