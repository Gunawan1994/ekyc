import api from '../lib/axios'
import type { ApiResponse, KYCVerification, ListParams } from '../types'

export interface SubmitKYCDto {
  customerId: string
  notes?: string
  idDocumentUrl?: string
  selfieUrl?: string
}

export interface ReviewKYCDto {
  notes: string
}

export const kycApi = {
  list: (params?: ListParams) =>
    api.get<ApiResponse<KYCVerification[]>>('/kyc', { params }),

  getById: (id: string) =>
    api.get<ApiResponse<KYCVerification>>(`/kyc/${id}`),

  submit: ({ customerId, idDocumentUrl, selfieUrl, ...rest }: SubmitKYCDto) =>
    api.post<ApiResponse<KYCVerification>>('/kyc/submit', {
      ...rest,
      customer_id: customerId,
      ...(idDocumentUrl && { id_document_url: idDocumentUrl }),
      ...(selfieUrl && { selfie_url: selfieUrl }),
    }),

  approve: (id: string, data: ReviewKYCDto) =>
    api.put<ApiResponse<KYCVerification>>(`/kyc/${id}/approve`, data),

  reject: (id: string, data: ReviewKYCDto) =>
    api.put<ApiResponse<KYCVerification>>(`/kyc/${id}/reject`, data),

  setInReview: (id: string, data?: { notes?: string }) =>
    api.put<ApiResponse<KYCVerification>>(`/kyc/${id}/review`, data ?? {}),

  requestDocs: (id: string, data: { notes: string }) =>
    api.post<ApiResponse<KYCVerification>>(`/kyc/${id}/request-docs`, data),

  delete: (id: string) =>
    api.delete<void>(`/kyc/${id}`),
}
