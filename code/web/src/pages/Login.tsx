import { useState } from 'react'
import { Lock, ArrowRight } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function Login() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim() || !password.trim()) return setError('请输入用户名和密码')
    setLoading(true)
    setError('')
    try {
      const tenants = await login(username.trim(), password.trim())
      if (tenants.length === 0) {
        navigate('/onboarding')
      } else if (tenants.length === 1) {
        navigate('/tasks')
      } else {
        const lastId = localStorage.getItem('tick_last_tenant')
        if (lastId && tenants.find((t: { id: string }) => t.id === lastId)) {
          navigate('/tasks')
        } else {
          navigate('/select-tenant')
        }
      }
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '登录失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-white p-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-stone-100 ring-1 ring-stone-200">
            <Lock className="h-6 w-6 text-gray-700" />
          </div>
          <h1 className="text-2xl font-semibold text-gray-900">Tick</h1>
          <p className="mt-1 text-sm text-gray-500">登录您的账户</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="username" className="mb-1.5 block text-sm font-medium text-gray-600">
              用户名
            </label>
            <input
              id="username"
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="请输入用户名"
              className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
              autoFocus
            />
          </div>
          <div>
            <label htmlFor="password" className="mb-1.5 block text-sm font-medium text-gray-600">
              密码
            </label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="至少 8 个字符"
              className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
            />
          </div>
          {error && <p className="text-xs text-red-500">{error}</p>}
          <button
            type="submit"
            disabled={loading}
            className="flex w-full items-center justify-center gap-2 rounded-lg bg-gray-900 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-gray-800 disabled:opacity-50 cursor-pointer"
          >
            {loading ? '登录中...' : '登录'}
            {!loading && <ArrowRight className="h-4 w-4" />}
          </button>
        </form>

        <p className="mt-4 text-center text-sm text-gray-500">
          还没有账户？{' '}
          <Link to="/register" className="text-gray-700 hover:text-gray-900">
            注册
          </Link>
        </p>
      </div>
    </div>
  )
}
