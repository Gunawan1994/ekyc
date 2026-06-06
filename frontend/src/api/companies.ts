import api from '../lib/axios'
import type { ApiResponse, Company, ListParams } from '../types'

export interface CreateCompanyDto {
  name: string
  registrationNumber: string
  address: string
  phone: string
  email: string
}

export type UpdateCompanyDto = Partial<CreateCompanyDto> & {
  status?: 'pending' | 'active' | 'inactive'
}

export const companiesApi = {
  list: (params?: ListParams) =>
    api.get<ApiResponse<Company[]>>('/companies', { params }),

  getById: (id: string) =>
    api.get<ApiResponse<Company>>(`/companies/${id}`),

  create: ({ registrationNumber, ...rest }: CreateCompanyDto) =>
    api.post<ApiResponse<Company>>('/companies', { ...rest, registration_number: registrationNumber }),

  update: (id: string, { registrationNumber, ...rest }: UpdateCompanyDto) =>
    api.put<ApiResponse<Company>>(`/companies/${id}`, registrationNumber !== undefined ? { ...rest, registration_number: registrationNumber } : rest),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/companies/${id}`),
}
