import { useState } from 'react'
import { Building2, Ticket, ArrowRight } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

export default function Onboarding() {
  const [mode, setMode] = useState<'create' | 'join' | null>(null)
  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { createTenant, joinTenant } = useAuth()
  const navigate = useNavigate()

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return setError('请输入工作空间名称')
    setLoading(true)
    setError('')
    try {
      await createTenant(name.trim())
      navigate('/tasks')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setLoading(false)
    }
  }

  const handleJoin = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!code.trim()) return setError('请输入邀请码')
    setLoading(true)
    setError('')
    try {
      await joinTenant(code.trim())
      navigate('/tasks')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '加入失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-white p-4">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-semibold text-gray-900">欢迎使用 Tick</h1>
          <p className="mt-1 text-sm text-gray-500">创建或加入一个工作空间开始使用</p>
        </div>

        {!mode && (
          <div className="space-y-3">
            <button
              onClick={() => setMode('create')}
              className="flex w-full items-center gap-4 rounded-lg border border-gray-200 p-4 text-left transition-colors hover:bg-stone-50 cursor-pointer"
            >
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-stone-100">
                <Building2 className="h-5 w-5 text-gray-700" />
              </div>
              <div>
                <div className="text-sm font-medium text-gray-900">创建工作空间</div>
                <div className="text-xs text-gray-500">创建一个新的工作空间并成为管理员</div>
              </div>
            </button>
            <button
              onClick={() => setMode('join')}
              className="flex w-full items-center gap-4 rounded-lg border border-gray-200 p-4 text-left transition-colors hover:bg-stone-50 cursor-pointer"
            >
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-stone-100">
                <Ticket className="h-5 w-5 text-gray-700" />
              </div>
              <div>
                <div className="text-sm font-medium text-gray-900">通过邀请码加入</div>
                <div className="text-xs text-gray-500">使用邀请码加入已有的工作空间</div>
              </div>
            </button>
          </div>
        )}

        {mode === 'create' && (
          <form onSubmit={handleCreate} className="space-y-4">
            <div>
              <label htmlFor="tenant-name" className="mb-1.5 block text-sm font-medium text-gray-600">
                工作空间名称
              </label>
              <input
                id="tenant-name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="例如：我的团队"
                className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
                autoFocus
              />
            </div>
            {error && <p className="text-xs text-red-500">{error}</p>}
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => { setMode(null); setError('') }}
                className="rounded-lg border border-gray-200 px-4 py-2.5 text-sm text-gray-600 transition-colors hover:bg-stone-50 cursor-pointer"
              >
                返回
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-gray-900 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-gray-800 disabled:opacity-50 cursor-pointer"
              >
                {loading ? '创建中...' : '创建'}
                {!loading && <ArrowRight className="h-4 w-4" />}
              </button>
            </div>
          </form>
        )}

        {mode === 'join' && (
          <form onSubmit={handleJoin} className="space-y-4">
            <div>
              <label htmlFor="invite-code" className="mb-1.5 block text-sm font-medium text-gray-600">
                邀请码
              </label>
              <input
                id="invite-code"
                type="text"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                placeholder="请输入邀请码"
                className="w-full rounded-lg border border-gray-200 bg-white px-3 py-2.5 text-sm text-gray-900 placeholder-gray-400 outline-none transition-colors focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
                autoFocus
              />
            </div>
            {error && <p className="text-xs text-red-500">{error}</p>}
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => { setMode(null); setError('') }}
                className="rounded-lg border border-gray-200 px-4 py-2.5 text-sm text-gray-600 transition-colors hover:bg-stone-50 cursor-pointer"
              >
                返回
              </button>
              <button
                type="submit"
                disabled={loading}
                className="flex flex-1 items-center justify-center gap-2 rounded-lg bg-gray-900 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-gray-800 disabled:opacity-50 cursor-pointer"
              >
                {loading ? '加入中...' : '加入'}
                {!loading && <ArrowRight className="h-4 w-4" />}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
