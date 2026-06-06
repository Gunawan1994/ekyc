import api from '../lib/axios'
import type { ApiResponse, ListParams } from '../types'

export interface AuditLog {
  id: string
  actorEmail: string
  action: string
  entityType: string
  entityId: string
  createdAt: string
}

export interface AuditListParams extends ListParams {
  action?: string
}

export const auditApi = {
  list: (params?: AuditListParams) =>
    api.get<ApiResponse<AuditLog[]>>('/audit-logs', { params }),
}
