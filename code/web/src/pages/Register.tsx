import { useState } from 'react'
import { UserPlus, ArrowRight } from 'lucide-react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function Register() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { register } = useAuth()
  const navigate = useNavigate()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!username.trim() || !password.trim()) return setError('所有字段均为必填')
    if (password.length < 8) return setError('密码至少 8 个字符')
    setLoading(true)
    setError('')
    try {
      await register(username.trim(), password.trim(), displayName.trim() || undefined)
      navigate('/onboarding')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-white p-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 text-center">
          <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-stone-100 ring-1 ring-stone-200">
            <UserPlus className="h-6 w-6 text-gray-900" />
          </div>
          <h1 className="text-2xl font-semibold text-gray-900">创建账户</h1>
          <p className="mt-1 text-sm text-gray-500">注册 Tick 调度平台</p>
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
              placeholder="3-64 个字符，字母、数字、_ 或 -"
              className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
              autoFocus
            />
          </div>
          <div>
            <label htmlFor="displayName" className="mb-1.5 block text-sm font-medium text-gray-600">
              显示名称 <span className="text-gray-400">（可选）</span>
            </label>
            <input
              id="displayName"
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="您的显示名称"
              className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
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
            className="flex w-full items-center justify-center gap-2 rounded-md bg-black px-4 py-2.5 text-sm font-medium text-white transition-colors duration-150 hover:bg-gray-800 disabled:opacity-50 cursor-pointer"
          >
            {loading ? '创建中...' : '创建账户'}
            {!loading && <ArrowRight className="h-4 w-4" />}
          </button>
        </form>

        <p className="mt-4 text-center text-sm text-gray-500">
          已有账户？{' '}
          <Link to="/login" className="text-gray-700 underline hover:text-gray-900">
            登录
          </Link>
        </p>
      </div>
    </div>
  )
}
