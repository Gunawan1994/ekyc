export interface User {
  id: string
  email: string
  fullName: string
  role: string
  isActive: boolean
}

export interface AuthTokens {
  accessToken: string
  refreshToken: string
}

export interface PaginationMeta {
  page: number
  pageSize: number
  total: number
  pages: number
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: { code: string; message: string }
  meta?: PaginationMeta
}

export interface Customer {
  id: string
  companyId: string
  fullName: string
  idNumber: string
  idType: 'ktp' | 'passport' | 'sim'
  phone: string
  email: string
  address: string
  createdAt: string
}

export interface Company {
  id: string
  name: string
  registrationNumber: string
  address: string
  phone: string
  email: string
  status: 'pending' | 'active' | 'inactive'
  createdAt: string
}

export type VerificationStatus = 'pending' | 'in_review' | 'approved' | 'rejected' | 'additional_docs_required'
export type RiskLevel = 'low' | 'medium' | 'high' | 'critical'

export interface KYCVerification {
  id: string
  customerId: string
  customerName?: string
  submittedBy: string
  reviewerId?: string
  status: VerificationStatus
  idDocumentUrl?: string
  selfieUrl?: string
  livenessScore?: number
  faceMatchScore?: number
  riskLevel?: RiskLevel
  riskScore?: number
  rejectionReason?: string
  notes?: string
  submittedAt: string
  reviewedAt?: string
  createdAt?: string
  updatedAt?: string
}

export interface KYBVerification {
  id: string
  companyId: string
  companyName?: string
  submittedBy: string
  reviewerId?: string
  status: VerificationStatus
  businessDocUrl?: string
  taxDocUrl?: string
  directorIdDocUrl?: string
  riskLevel?: RiskLevel
  riskScore?: number
  rejectionReason?: string
  notes?: string
  submittedAt: string
  reviewedAt?: string
  createdAt?: string
  updatedAt?: string
}

export interface DashboardStats {
  totalCustomers: number
  totalCompanies: number
  totalKycPending: number
  totalKycApproved: number
  totalKycRejected: number
  totalKybPending: number
  totalKybApproved: number
  totalKybRejected: number
}

export interface ListParams {
  page?: number
  pageSize?: number
  search?: string
  status?: string
}
