import { createContext, useContext, useState, useCallback, ReactNode } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  api, UserInfo, UserTenantInfo,
  setToken, clearToken,
  setUserInfo, getUserInfo, clearUserInfo,
  setTenantInfo, getTenantInfo, clearTenantInfo,
  setTenantsInfo, getTenantsInfo, clearTenantsInfo,
} from '../api/client'

interface AuthState {
  user: UserInfo | null
  tenants: UserTenantInfo[]
  currentTenant: UserTenantInfo | null
  role: 'owner' | 'member' | null
  login: (username: string, password: string) => Promise<UserTenantInfo[]>
  register: (username: string, password: string, display_name?: string) => Promise<void>
  selectTenant: (tenant_id: string) => Promise<void>
  createTenant: (name: string) => Promise<void>
  joinTenant: (code: string) => Promise<void>
  logout: () => void
  refreshTenants: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserInfo | null>(getUserInfo)
  const [tenants, setTenants] = useState<UserTenantInfo[]>(getTenantsInfo)
  const [currentTenant, setCurrentTenant] = useState<UserTenantInfo | null>(getTenantInfo)
  const navigate = useNavigate()

  const role = currentTenant?.role ?? null

  const login = useCallback(async (username: string, password: string) => {
    const res = await api.loginUser(username, password)
    const d = res.data
    const info: UserInfo = { user_id: d.user_id, username: d.username, display_name: d.display_name }
    setUser(info)
    setUserInfo(info)
    const tenantList = d.tenants || []
    setTenants(tenantList)
    setTenantsInfo(tenantList)

    if (d.token) {
      setToken(d.token)
    }
    if (tenantList.length === 1) {
      const t = tenantList[0]
      setCurrentTenant(t)
      setTenantInfo(t)
      localStorage.setItem('tick_last_tenant', t.id)
    } else if (tenantList.length > 1) {
      const lastId = localStorage.getItem('tick_last_tenant')
      const last = lastId ? tenantList.find(t => t.id === lastId) : null
      if (last) {
        const res2 = await api.selectTenant(last.id)
        setToken(res2.data.token)
        const t = res2.data.tenant
        setCurrentTenant(t)
        setTenantInfo(t)
      }
    }
    return tenantList
  }, [])

  const register = useCallback(async (username: string, password: string, display_name?: string) => {
    await api.registerUser(username, password, display_name)
    // Register doesn't return a token, so we log in immediately to get one
    const res = await api.loginUser(username, password)
    const d = res.data
    const info: UserInfo = { user_id: d.user_id, username: d.username, display_name: d.display_name }
    setUser(info)
    setUserInfo(info)
    const tenantList = d.tenants || []
    setTenants(tenantList)
    setTenantsInfo(tenantList)
    if (d.token) {
      setToken(d.token)
    }
  }, [])

  const selectTenantFn = useCallback(async (tenant_id: string) => {
    const res = await api.selectTenant(tenant_id)
    setToken(res.data.token)
    const t = res.data.tenant
    setCurrentTenant(t)
    setTenantInfo(t)
    localStorage.setItem('tick_last_tenant', t.id)
    setTenants(prev => {
      if (prev.find(p => p.id === t.id)) return prev
      const next = [...prev, t]
      setTenantsInfo(next)
      return next
    })
  }, [])

  const createTenantFn = useCallback(async (name: string) => {
    const res = await api.createTenant(name)
    const d = res.data
    setToken(d.token)
    const t: UserTenantInfo = { id: d.id, name: d.name, role: d.role }
    setCurrentTenant(t)
    setTenantInfo(t)
    setTenants(prev => {
      const next = [...prev, t]
      setTenantsInfo(next)
      return next
    })
  }, [])

  const joinTenantFn = useCallback(async (code: string) => {
    const res = await api.joinTenant(code)
    setToken(res.data.token)
    const t = res.data.tenant
    setCurrentTenant(t)
    setTenantInfo(t)
    setTenants(prev => [...prev, t])
  }, [])

  const logout = useCallback(() => {
    clearToken()
    clearUserInfo()
    clearTenantInfo()
    clearTenantsInfo()
    setUser(null)
    setTenants([])
    setCurrentTenant(null)
    navigate('/login')
  }, [navigate])

  const refreshTenants = useCallback(async () => {
    const res = await api.listTenants()
    setTenants(res.data.tenants)
  }, [])

  return (
    <AuthContext.Provider value={{
      user, tenants, currentTenant, role,
      login, register,
      selectTenant: selectTenantFn,
      createTenant: createTenantFn,
      joinTenant: joinTenantFn,
      logout, refreshTenants,
    }}>
      {children}
    </AuthContext.Provider>
  )
}
