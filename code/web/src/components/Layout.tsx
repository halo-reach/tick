import { NavLink, useLocation } from 'react-router-dom'
import { ListTodo, Key, Gauge, LogOut, Clock, ChevronLeft, ChevronRight, Braces, ShieldCheck, Users, CircleUser, KeyRound, X } from 'lucide-react'
import { useState, useRef, useEffect, ReactNode } from 'react'
import { useAuth } from '../context/AuthContext'
import { api } from '../api/client'
import TenantSwitcher from './TenantSwitcher'

const navItems = [
  { to: '/tasks', icon: ListTodo, label: '任务列表' },
  { to: '/credentials', icon: ShieldCheck, label: '凭证管理' },
  { to: '/variables', icon: Braces, label: '变量管理' },
  { to: '/members', icon: Users, label: '成员管理' },
  { to: '/keys', icon: Key, label: 'API Key 管理' },
  { to: '/quota', icon: Gauge, label: '配额概览' },
]

export default function Layout({ children }: { children: ReactNode }) {
  const [collapsed, setCollapsed] = useState(false)
  const location = useLocation()
  const { user, logout } = useAuth()

  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const [pwdModalOpen, setPwdModalOpen] = useState(false)
  const [currentPwd, setCurrentPwd] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [pwdError, setPwdError] = useState('')
  const [pwdSuccess, setPwdSuccess] = useState('')
  const [pwdLoading, setPwdLoading] = useState(false)
  const userMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  useEffect(() => {
    if (!pwdModalOpen) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setPwdModalOpen(false)
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [pwdModalOpen])

  const openPwdModal = () => {
    setUserMenuOpen(false)
    setCurrentPwd('')
    setNewPwd('')
    setPwdError('')
    setPwdSuccess('')
    setPwdModalOpen(true)
  }

  const closePwdModal = () => {
    setPwdModalOpen(false)
    setCurrentPwd('')
    setNewPwd('')
    setPwdError('')
    setPwdSuccess('')
  }

  const handleChangePwd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (newPwd.length < 8) return setPwdError('新密码至少 8 个字符')
    setPwdLoading(true)
    setPwdError('')
    setPwdSuccess('')
    try {
      await api.changePassword(currentPwd, newPwd)
      setPwdSuccess('密码修改成功')
      setCurrentPwd('')
      setNewPwd('')
    } catch (err: unknown) {
      setPwdError(err instanceof Error ? err.message : '修改失败')
    } finally {
      setPwdLoading(false)
    }
  }

  return (
    <div className="flex h-screen overflow-hidden bg-white">
      {/* Sidebar */}
      <aside
        className={`${
          collapsed ? 'w-16' : 'w-56'
        } flex flex-col border-r border-stone-200 bg-[#F5F5F0] transition-all duration-200`}
      >
        {/* Logo */}
        <div className="flex h-14 items-center gap-2 border-b border-stone-200 px-4">
          <Clock className="h-5 w-5 shrink-0 text-gray-900" />
          {!collapsed && (
            <span className="text-lg font-semibold text-gray-900 tracking-tight">
              Tick
            </span>
          )}
        </div>

        {/* Nav */}
        <nav className="flex-1 space-y-1 p-2">
          {navItems.map(({ to, icon: Icon, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-md px-3 py-2.5 text-sm transition-colors duration-150 cursor-pointer ${
                  isActive
                    ? 'bg-[#E8EDE8] text-gray-900 font-medium'
                    : 'text-gray-600 hover:bg-stone-200/50 hover:text-gray-900'
                }`
              }
            >
              <Icon className="h-4.5 w-4.5 shrink-0" />
              {!collapsed && label}
            </NavLink>
          ))}
        </nav>

        {/* Collapse */}
        <div className="border-t border-stone-200 p-2">
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm text-gray-500 hover:bg-stone-200/50 hover:text-gray-700 transition-colors duration-150 cursor-pointer"
          >
            {collapsed ? (
              <ChevronRight className="h-4 w-4 shrink-0" />
            ) : (
              <>
                <ChevronLeft className="h-4 w-4 shrink-0" />
                收起
              </>
            )}
          </button>
        </div>
      </aside>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        {/* Top bar */}
        <header className="flex h-14 items-center justify-between bg-stone-50 border-b border-stone-200 px-8">
          <h1 className="text-lg font-semibold text-gray-900">
            {navItems.find((n) => location.pathname.startsWith(n.to))?.label ?? 'Tick'}
          </h1>
          <div className="flex items-center gap-3">
            <TenantSwitcher />
            <div ref={userMenuRef} className="relative">
              <button
                onClick={() => { setUserMenuOpen(!userMenuOpen) }}
                className="flex items-center gap-1.5 rounded-md px-2 py-1.5 text-sm text-gray-600 hover:bg-stone-100 transition-colors cursor-pointer"
                title={user?.display_name || user?.username || ''}
              >
                <CircleUser className="h-4 w-4" />
                <span>{user?.display_name || user?.username}</span>
              </button>
              {userMenuOpen && (
                <div className="absolute right-0 top-full z-50 mt-2 w-auto whitespace-nowrap rounded-lg border border-gray-200 bg-white shadow-lg">
                  <div className="absolute -top-1.5 right-3 h-3 w-3 rotate-45 border-l border-t border-gray-200 bg-white" />
                  <div className="px-2 py-2 border-b border-gray-100">
                    <p className="text-sm font-medium text-gray-900 text-right truncate">{user?.display_name || user?.username}</p>
                    <p className="text-xs text-gray-500 text-right truncate">{user?.username}</p>
                  </div>
                  <div className="py-1">
                    <button
                      onClick={openPwdModal}
                      className="flex w-full items-center justify-end gap-1.5 px-2 py-1.5 text-sm font-medium text-gray-600 hover:bg-stone-50 cursor-pointer"
                    >
                      <KeyRound className="h-3.5 w-3.5" />
                      修改密码
                    </button>
                    <div className="mx-2 border-t border-gray-100" />
                    <button
                      onClick={logout}
                      className="flex w-full items-center justify-end gap-1.5 px-2 py-1.5 text-sm font-medium text-red-600 hover:bg-red-50 cursor-pointer"
                    >
                      <LogOut className="h-3.5 w-3.5" />
                      退出登录
                    </button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </header>

        <main className="flex-1 overflow-y-auto bg-stone-50 p-6">
          <div className="mx-auto max-w-6xl">{children}</div>
        </main>
      </div>

      {pwdModalOpen && (
        <div className="fixed inset-0 z-[100] flex items-center justify-center">
          <div className="absolute inset-0 bg-black/40" onClick={closePwdModal} />
          <div className="relative z-10 w-[400px] rounded-lg border border-stone-200 bg-white p-5 shadow-xl">
            <div className="mb-4 flex items-center justify-between">
              <h3 className="text-base font-semibold text-gray-900">修改密码</h3>
              <button onClick={closePwdModal} className="rounded p-1 text-gray-400 hover:text-gray-600 hover:bg-stone-100 cursor-pointer">
                <X className="h-4 w-4" />
              </button>
            </div>
            <form onSubmit={handleChangePwd} className="space-y-3">
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-600">当前密码</label>
                <input
                  type="password" required value={currentPwd}
                  onChange={e => setCurrentPwd(e.target.value)}
                  autoFocus
                  className="w-full rounded-md border border-stone-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
                />
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gray-600">新密码</label>
                <input
                  type="password" required value={newPwd}
                  onChange={e => setNewPwd(e.target.value)}
                  placeholder="至少 8 个字符"
                  className="w-full rounded-md border border-stone-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
                />
              </div>
              {pwdError && <p className="text-xs text-red-500">{pwdError}</p>}
              {pwdSuccess && <p className="text-xs text-green-600">{pwdSuccess}</p>}
              <div className="flex gap-2 pt-2">
                <button type="button" onClick={closePwdModal} className="flex-1 rounded-md border border-stone-200 px-4 py-2 text-sm text-gray-600 hover:bg-stone-50 cursor-pointer">
                  取消
                </button>
                <button type="submit" disabled={pwdLoading} className="flex-1 rounded-md bg-gray-900 px-4 py-2 text-sm text-white hover:bg-gray-800 disabled:opacity-50 cursor-pointer">
                  {pwdLoading ? '修改中...' : '确认'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
