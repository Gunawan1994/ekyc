import api from '../lib/axios'
import type { ApiResponse } from '../types'

export interface RiskAssessment {
  id: string
  entityType: string
  entityId: string
  riskLevel: string
  riskScore: number
  riskFactors: Record<string, boolean | number | string>
  assessedBy?: string
  notes?: string
  assessedAt: string
  createdAt: string
}

export interface ManualOverrideDto {
  risk_level: 'low' | 'medium' | 'high' | 'critical'
  notes?: string
}

export const riskApi = {
  getKYCRisk: (id: string) =>
    api.get<ApiResponse<RiskAssessment>>(`/kyc/${id}/risk`),

  getKYBRisk: (id: string) =>
    api.get<ApiResponse<RiskAssessment>>(`/kyb/${id}/risk`),

  overrideKYCRisk: (id: string, data: ManualOverrideDto) =>
    api.post<ApiResponse<RiskAssessment>>(`/kyc/${id}/risk`, data),

  overrideKYBRisk: (id: string, data: ManualOverrideDto) =>
    api.post<ApiResponse<RiskAssessment>>(`/kyb/${id}/risk`, data),

  listKYCHistory: (id: string) =>
    api.get<ApiResponse<RiskAssessment[]>>(`/kyc/${id}/risk/history`),

  listKYBHistory: (id: string) =>
    api.get<ApiResponse<RiskAssessment[]>>(`/kyb/${id}/risk/history`),
}
