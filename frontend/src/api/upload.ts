import api from '../lib/axios'

export async function uploadFile(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const res = await api.post<{ url: string }>('/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    transformResponse: [(data) => {
      try { return JSON.parse(data) } catch { return data }
    }],
  })
  return res.data.url
}
