const API_BASE = '/api/v1'

function getToken(): string | null {
  return localStorage.getItem('tick_token')
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getToken()
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error?.message || `HTTP ${res.status}`)
  }
  return res.json()
}

export interface Credential {
  id: string
  name: string
  code: string
  type: string
  status: 'active' | 'disabled'
  config_preview: Record<string, unknown> | string
  timeout_secs: number
  created_at: string
}

export const credentialApi = {
  list: (params?: { status?: string; page?: number; size?: number }) => {
    const p = params?.page ?? 1
    const s = params?.size ?? 20
    const qs = new URLSearchParams({ limit: String(s), offset: String((p - 1) * s) })
    if (params?.status) qs.set('status', params.status)
    return request<{ data: { items: Credential[]; total: number } }>(`/credentials?${qs}`)
  },
  create: (data: Record<string, unknown>) =>
    request<{ data: Credential }>('/credentials', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: Record<string, unknown>) =>
    request<{ data: Credential }>(`/credentials/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  getDecrypted: (id: string) =>
    request<{ data: { id: string; name: string; type: string; status: string; timeout_secs: number; config: Record<string, unknown>; created_at: string } }>(`/credentials/${id}/config`),
  patchStatus: (id: string, status: string) =>
    request<{ data: Credential }>(`/credentials/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) }),
  delete: (id: string) =>
    request<{ data: { deleted: boolean } }>(`/credentials/${id}`, { method: 'DELETE' }),
  test: (id: string) =>
    request<{ data: { credential_name: string; credential_type: string; injections: Array<{ location: string; key: string; value: string }> } }>(`/credentials/${id}/test`),
  bindTarget: (targetId: string, data: Record<string, unknown>) =>
    request<{ data: unknown }>(`/targets/${targetId}/credentials`, { method: 'POST', body: JSON.stringify(data) }),
  listTarget: (targetId: string) =>
    request<{ data: unknown[] }>(`/targets/${targetId}/credentials`),
  unbindTarget: (targetId: string, credId: string) =>
    request<{ data: { deleted: boolean } }>(`/targets/${targetId}/credentials/${credId}`, { method: 'DELETE' }),
}
