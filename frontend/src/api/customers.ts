import api from '../lib/axios'
import type { ApiResponse, Customer, ListParams } from '../types'

export interface CreateCustomerDto {
  companyId: string
  fullName: string
  idNumber: string
  idType: 'ktp' | 'passport' | 'sim'
  phone: string
  email: string
  address: string
}

export type UpdateCustomerDto = Partial<CreateCustomerDto>

export const customersApi = {
  list: (params?: ListParams) =>
    api.get<ApiResponse<Customer[]>>('/customers', { params }),

  getById: (id: string) =>
    api.get<ApiResponse<Customer>>(`/customers/${id}`),

  create: ({ companyId, fullName, idNumber, idType, ...rest }: CreateCustomerDto) =>
    api.post<ApiResponse<Customer>>('/customers', {
      ...rest,
      company_id: companyId,
      full_name: fullName,
      id_number: idNumber,
      id_type: idType,
    }),

  update: (id: string, { companyId, fullName, idNumber, idType, ...rest }: UpdateCustomerDto) =>
    api.put<ApiResponse<Customer>>(`/customers/${id}`, {
      ...rest,
      ...(companyId !== undefined && { company_id: companyId }),
      ...(fullName !== undefined && { full_name: fullName }),
      ...(idNumber !== undefined && { id_number: idNumber }),
      ...(idType !== undefined && { id_type: idType }),
    }),

  delete: (id: string) =>
    api.delete<ApiResponse<null>>(`/customers/${id}`),
}
