import { useState, useCallback } from 'react'
import { XCircle, Plus, Copy, Trash2 } from 'lucide-react'
import { api, type ApiKeyInfo } from '../api/client'
import { useAutoRefresh } from '../hooks/useAutoRefresh'

export default function ApiKeys() {
  const [showCreate, setShowCreate] = useState(false)
  const [newKeyName, setNewKeyName] = useState('')
  const [createdKey, setCreatedKey] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState('')
  const [revokeConfirm, setRevokeConfirm] = useState<string | null>(null)

  const fetcher = useCallback(() => api.listKeys(), [])
  const { data, error: fetchErr, loading, refresh } = useAutoRefresh<{ data: ApiKeyInfo[] }>(fetcher)

  const handleCreate = async () => {
    if (!newKeyName.trim()) return
    setCreating(true)
    setError('')
    try {
      const res = await api.createKey(newKeyName.trim())
      setCreatedKey(res.data.api_key)
      setNewKeyName('')
      refresh()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '创建密钥失败')
    } finally {
      setCreating(false)
    }
  }

  const handleRevoke = async (id: string) => {
    try {
      await api.revokeKey(id)
      setRevokeConfirm(null)
      refresh()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '撤销密钥失败')
    }
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-stone-400 border-t-transparent" />
      </div>
    )
  }

  if (fetchErr) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-2 text-red-500">
        <XCircle className="h-8 w-8" />
        <p className="text-sm">{fetchErr}</p>
      </div>
    )
  }

  const keys = data?.data ?? []
  const activeKeys = keys.filter((k) => k.status === 'active')

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold text-gray-900">API 密钥</h2>
        <button
          onClick={() => { setShowCreate(true); setCreatedKey(null) }}
          className="flex items-center gap-1.5 bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors duration-150 cursor-pointer"
        >
          <Plus className="h-4 w-4" /> 创建密钥
        </button>
      </div>

      {error && <p className="text-sm text-red-500">{error}</p>}

      {createdKey && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-4">
          <p className="mb-2 text-sm font-medium text-green-800">API 密钥已创建！请立即复制，之后将不再显示。</p>
          <div className="flex items-center gap-2">
            <code className="flex-1 rounded bg-white px-3 py-2 font-mono text-xs text-gray-900 border border-green-200 break-all">
              {createdKey}
            </code>
            <button
              onClick={() => navigator.clipboard.writeText(createdKey)}
              className="rounded-lg p-2 text-green-700 hover:bg-green-100 cursor-pointer"
            >
              <Copy className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {showCreate && !createdKey && (
        <div className="rounded-lg border border-gray-200 bg-gray-50 p-4">
          <label className="mb-1.5 block text-xs font-medium text-gray-600">密钥名称</label>
          <div className="flex gap-2">
            <input
              type="text"
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              placeholder="例如：生产环境、测试环境"
              className="flex-1 rounded-lg border border-stone-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
              autoFocus
            />
            <button
              onClick={handleCreate}
              disabled={creating || !newKeyName.trim()}
              className="bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors duration-150 disabled:opacity-50 cursor-pointer"
            >
              {creating ? '创建中...' : '创建'}
            </button>
            <button
              onClick={() => setShowCreate(false)}
              className="rounded-lg border border-gray-200 px-4 py-2 text-sm text-gray-600 hover:bg-gray-100 cursor-pointer"
            >
              取消
            </button>
          </div>
        </div>
      )}

      <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-stone-200 bg-gray-50">
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">名称</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">前缀</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">状态</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">创建时间</th>
              <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-stone-100">
            {keys.map((key) => (
              <tr key={key.id} className="hover:bg-gray-50">
                <td className="px-4 py-3 font-medium text-gray-900">{key.name}</td>
                <td className="px-4 py-3 font-mono text-xs text-gray-500">{key.key_prefix}...</td>
                <td className="px-4 py-3">
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
                    key.status === 'active' ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'
                  }`}>
                    {key.status}
                  </span>
                </td>
                <td className="px-4 py-3 text-xs text-gray-500">{new Date(key.created_at).toLocaleDateString()}</td>
                <td className="px-4 py-3 text-right">
                  {key.status === 'active' && (
                    revokeConfirm === key.id ? (
                      <div className="flex items-center justify-end gap-1">
                        <span className="text-xs text-red-600">
                          {activeKeys.length === 1 ? '最后一个密钥！撤销后 CLI/API 将无法访问。' : '确认撤销？'}
                        </span>
                        <button
                          onClick={() => handleRevoke(key.id)}
                          className="rounded px-2 py-1 text-xs text-red-600 hover:bg-red-50 cursor-pointer"
                        >
                          是
                        </button>
                        <button
                          onClick={() => setRevokeConfirm(null)}
                          className="rounded px-2 py-1 text-xs text-gray-500 hover:bg-gray-100 cursor-pointer"
                        >
                          否
                        </button>
                      </div>
                    ) : (
                      <button
                        onClick={() => setRevokeConfirm(key.id)}
                        className="rounded-lg p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-500 cursor-pointer"
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    )
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
