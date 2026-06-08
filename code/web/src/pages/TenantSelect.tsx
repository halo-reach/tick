import { Building2, Plus } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'
import { useState } from 'react'

export default function TenantSelect() {
  const { tenants, selectTenant } = useAuth()
  const [loading, setLoading] = useState<string | null>(null)
  const [error, setError] = useState('')
  const navigate = useNavigate()

  const handleSelect = async (id: string) => {
    setLoading(id)
    setError('')
    try {
      await selectTenant(id)
      navigate('/tasks')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '选择失败')
    } finally {
      setLoading(null)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-white p-4">
      <div className="w-full max-w-md">
        <div className="mb-8 text-center">
          <h1 className="text-2xl font-semibold text-gray-900">选择工作空间</h1>
          <p className="mt-1 text-sm text-gray-500">选择要进入的工作空间</p>
        </div>

        {error && <p className="mb-4 text-center text-xs text-red-500">{error}</p>}

        <div className="space-y-2">
          {tenants.map((t) => (
            <button
              key={t.id}
              onClick={() => handleSelect(t.id)}
              disabled={loading !== null}
              className="flex w-full items-center gap-4 rounded-lg border border-gray-200 p-4 text-left transition-colors hover:bg-stone-50 disabled:opacity-50 cursor-pointer"
            >
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-stone-100">
                <Building2 className="h-5 w-5 text-gray-700" />
              </div>
              <div className="flex-1">
                <div className="text-sm font-medium text-gray-900">{t.name}</div>
                <div className="text-xs text-gray-500">{t.role === 'owner' ? '管理员' : '成员'}</div>
              </div>
              {loading === t.id && <span className="text-xs text-gray-400">进入中...</span>}
            </button>
          ))}
        </div>

        <button
          onClick={() => navigate('/onboarding')}
          className="mt-4 flex w-full items-center justify-center gap-2 rounded-lg border border-dashed border-gray-300 px-4 py-3 text-sm text-gray-500 transition-colors hover:border-gray-400 hover:text-gray-700 cursor-pointer"
        >
          <Plus className="h-4 w-4" />
          创建新工作空间
        </button>
      </div>
    </div>
  )
}
