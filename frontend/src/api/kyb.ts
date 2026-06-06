import api from '../lib/axios'
import type { ApiResponse, KYBVerification, ListParams } from '../types'

export interface SubmitKYBDto {
  companyId: string
  notes?: string
  businessDocUrl?: string
  taxDocUrl?: string
  directorIdDocUrl?: string
}

export interface ReviewKYBDto {
  notes: string
}

export const kybApi = {
  list: (params?: ListParams) =>
    api.get<ApiResponse<KYBVerification[]>>('/kyb', { params }),

  getById: (id: string) =>
    api.get<ApiResponse<KYBVerification>>(`/kyb/${id}`),

  submit: ({ companyId, businessDocUrl, taxDocUrl, directorIdDocUrl, ...rest }: SubmitKYBDto) =>
    api.post<ApiResponse<KYBVerification>>('/kyb/submit', {
      ...rest,
      company_id: companyId,
      ...(businessDocUrl && { business_doc_url: businessDocUrl }),
      ...(taxDocUrl && { tax_doc_url: taxDocUrl }),
      ...(directorIdDocUrl && { director_id_doc_url: directorIdDocUrl }),
    }),

  approve: (id: string, data: ReviewKYBDto) =>
    api.put<ApiResponse<KYBVerification>>(`/kyb/${id}/approve`, data),

  reject: (id: string, data: ReviewKYBDto) =>
    api.put<ApiResponse<KYBVerification>>(`/kyb/${id}/reject`, data),

  setInReview: (id: string, data?: { notes?: string }) =>
    api.put<ApiResponse<KYBVerification>>(`/kyb/${id}/review`, data ?? {}),

  requestDocs: (id: string, data: { notes: string }) =>
    api.post<ApiResponse<KYBVerification>>(`/kyb/${id}/request-docs`, data),

  delete: (id: string) =>
    api.delete<void>(`/kyb/${id}`),
}
