const API_BASE = '/api/v1'

function getToken(): string | null {
  return localStorage.getItem('tick_token')
}

export function setToken(token: string) {
  localStorage.setItem('tick_token', token)
}

export function clearToken() {
  localStorage.removeItem('tick_token')
}

export function hasToken(): boolean {
  return !!getToken()
}

export interface UserInfo {
  user_id: string
  username: string
  display_name: string
}

export interface UserTenantInfo {
  id: string
  name: string
  role: 'owner' | 'member'
}

export interface LoginResponse {
  user_id: string
  username: string
  display_name: string
  tenants: UserTenantInfo[]
  token?: string
}

export interface MemberInfo {
  user_id: string
  username: string
  display_name: string
  role: 'owner' | 'member'
  joined_at: string
}

export interface InvitationInfo {
  id: string
  code: string
  role: string
  max_uses: number
  used_count: number
  expires_at: string
  created_at: string
}

export function setUserInfo(info: UserInfo) {
  localStorage.setItem('tick_user', JSON.stringify(info))
}

export function getUserInfo(): UserInfo | null {
  const raw = localStorage.getItem('tick_user')
  if (!raw) return null
  try { return JSON.parse(raw) } catch { return null }
}

export function clearUserInfo() {
  localStorage.removeItem('tick_user')
}

export function setTenantInfo(info: UserTenantInfo) {
  localStorage.setItem('tick_tenant', JSON.stringify(info))
}

export function getTenantInfo(): UserTenantInfo | null {
  const raw = localStorage.getItem('tick_tenant')
  if (!raw) return null
  try { return JSON.parse(raw) } catch { return null }
}

export function clearTenantInfo() {
  localStorage.removeItem('tick_tenant')
}

export function setTenantsInfo(list: UserTenantInfo[]) {
  localStorage.setItem('tick_tenants', JSON.stringify(list))
}

export function getTenantsInfo(): UserTenantInfo[] {
  const raw = localStorage.getItem('tick_tenants')
  if (!raw) return []
  try { return JSON.parse(raw) } catch { return [] }
}

export function clearTenantsInfo() {
  localStorage.removeItem('tick_tenants')
}

const AUTH_PAGES = ['/login', '/register', '/onboarding', '/select-tenant']

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
    if (res.status === 401 && !AUTH_PAGES.includes(window.location.pathname)) {
      clearToken()
      window.location.href = '/login'
      throw new Error('登录已过期，请重新登录')
    }
    const body = await res.json().catch(() => ({}))
    throw new Error(body.error?.message || `HTTP ${res.status}`)
  }
  if (res.status === 204) return {} as T
  return res.json()
}

export interface Task {
  id: string
  name: string
  schedule_type: 'cron' | 'interval' | 'once'
  cron_expr: string
  interval_value: number
  interval_unit: string
  status: 'active' | 'paused' | 'deleted'
  target_id: string
  timeout_secs: number
  retry_count: number
  retry_backoff: 'exponential' | 'fixed' | 'none'
  concurrency_policy: 'allow' | 'skip' | 'queue'
  max_concurrency: number
  execution_retention_days: number
  next_trigger_at: string | null
  created_at: string
  url?: string
  method?: string
  headers?: Record<string, string>
  body?: string | object
  content_type?: string
  auth_type?: string
  once_at?: string
}

export interface Execution {
  id: string
  task_id: string
  status: 'success' | 'failed' | 'timeout' | 'skipped' | 'retrying'
  status_code: number
  duration_ms: number
  error_msg: string
  request_headers: string
  request_body: string
  response_body: string
  is_makeup: boolean
  is_manual: boolean
  attempt: number
  trigger_time: string
}

export interface ApiKeyInfo {
  id: string
  name: string
  key_prefix: string
  status: string
  created_at: string
  revoked_at: string | null
}

export interface MeInfo {
  tenant_id: string
  username: string
  name: string
  quota_max_tasks: number
  quota_max_rps: number
  created_at: string
}

export const api = {
  // Auth - new multi-user endpoints
  registerUser: (username: string, password: string, display_name?: string) =>
    request<{ data: { user_id: string; username: string; display_name: string } }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password, display_name }),
    }),
  loginUser: (username: string, password: string) =>
    request<{ data: LoginResponse }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  listTenants: () =>
    request<{ data: { tenants: UserTenantInfo[] } }>('/auth/tenants'),
  selectTenant: (tenant_id: string) =>
    request<{ data: { token: string; tenant: UserTenantInfo } }>('/auth/select-tenant', {
      method: 'POST',
      body: JSON.stringify({ tenant_id }),
    }),

  // Tenant
  createTenant: (name: string) =>
    request<{ data: { id: string; name: string; role: 'owner' | 'member'; token: string } }>('/tenants', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  joinTenant: (code: string) =>
    request<{ data: { tenant: UserTenantInfo; token: string } }>('/tenants/join', {
      method: 'POST',
      body: JSON.stringify({ code }),
    }),

  // Members
  listMembers: () =>
    request<{ data: { members: MemberInfo[] } }>('/members'),
  createInvite: (opts?: { role?: string; max_uses?: number; expires_in_days?: number }) =>
    request<{ data: InvitationInfo }>('/members/invite', {
      method: 'POST',
      body: JSON.stringify(opts ?? {}),
    }),
  removeMember: (user_id: string) =>
    request<{ data: { removed: boolean } }>(`/members/${user_id}`, { method: 'DELETE' }),
  changeRole: (user_id: string, role: string) =>
    request<{ data: { user_id: string; role: string } }>(`/members/${user_id}/role`, {
      method: 'PATCH',
      body: JSON.stringify({ role }),
    }),
  listInvitations: () =>
    request<{ data: { invitations: InvitationInfo[] } }>('/invitations'),
  revokeInvitation: (id: string) =>
    request<{ data: { revoked: boolean } }>(`/invitations/${id}`, { method: 'DELETE' }),
  searchUsers: (q: string) =>
    request<{ data: { users: { id: string; username: string; display_name: string }[] } }>(`/users/search?q=${encodeURIComponent(q)}`),
  addMember: (user_id: string, role: string) =>
    request<{ data: { user_id: string; username: string; display_name: string; role: string } }>('/members', {
      method: 'POST',
      body: JSON.stringify({ user_id, role }),
    }),

  // Legacy compat aliases
  register: (username: string, password: string) =>
    request<{ data: { user_id: string; username: string; display_name: string } }>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  login: (username: string, password: string) =>
    request<{ data: LoginResponse }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),

  me: () => request<{ data: MeInfo }>('/auth/me'),
  changePassword: (current_password: string, new_password: string) =>
    request<{ data: { changed: boolean } }>('/auth/change-password', {
      method: 'POST',
      body: JSON.stringify({ current_password, new_password }),
    }),

  renameTenant: (name: string) =>
    request<{ data: { id: string; name: string } }>('/tenant/name', {
      method: 'PUT',
      body: JSON.stringify({ name }),
    }),
  listKeys: () => request<{ data: ApiKeyInfo[] }>('/auth/keys'),
  createKey: (name: string) =>
    request<{ data: { id: string; name: string; api_key: string } }>('/auth/keys', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  revokeKey: (id: string) =>
    request<{ data: { revoked: boolean } }>(`/auth/keys/${id}`, { method: 'DELETE' }),
  getTasks: (page = 1, size = 20) =>
    request<{ data: Task[]; total: number }>(`/tasks?limit=${size}&offset=${(page - 1) * size}`),
  getTask: (id: string) => request<{ data: Task }>(`/tasks/${id}`),
  createTask: (task: Record<string, unknown>) =>
    request<{ data: Task }>('/tasks', { method: 'POST', body: JSON.stringify(task) }),
  updateTask: (id: string, task: Record<string, unknown>) =>
    request<{ data: Task }>(`/tasks/${id}`, { method: 'PUT', body: JSON.stringify(task) }),
  deleteTask: (id: string) =>
    request<{ data: { deleted: boolean } }>(`/tasks/${id}`, { method: 'DELETE' }),
  pauseTask: (id: string) =>
    request<{ data: { paused: boolean } }>(`/tasks/${id}/pause`, { method: 'POST' }),
  resumeTask: (id: string) =>
    request<{ data: { resumed: boolean } }>(`/tasks/${id}/resume`, { method: 'POST' }),
  triggerTask: (id: string) =>
    request<{ data: { message: string; status: string; duration_ms: number } }>(`/tasks/${id}/trigger`, { method: 'POST' }),
  getHistory: (taskId: string, page = 1, size = 20) =>
    request<{ data: Execution[] }>(`/tasks/${taskId}/history?limit=${size}&offset=${(page - 1) * size}`),
  getQuota: () => request<{ data: { max_tasks: number; used_tasks: number; max_rps: number } }>('/quota'),
  getStatus: () => request<{ data: { tenant_id: string; name: string; status: string } }>('/status'),
  listCredentials: () => request<{ data: any[] }>('/credentials'),
  listVariables: () => request<{ data: any[] }>('/variables'),
  getVariable: (id: string) => request<{ data: any }>(`/variables/${id}`),
  createVariable: (data: { key: string; value: string }) =>
    request<{ data: any }>('/variables', { method: 'POST', body: JSON.stringify(data) }),
  updateVariable: (id: string, data: { key: string; value: string }) =>
    request<{ data: any }>(`/variables/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  deleteVariable: (id: string) =>
    request<{ data: any }>(`/variables/${id}`, { method: 'DELETE' }),
}
