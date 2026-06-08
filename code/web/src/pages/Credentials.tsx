import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Trash2, ChevronLeft, ChevronRight, Ban, Check } from 'lucide-react'
import { credentialApi, type Credential } from '../api/credential'
import ConfirmDialog from '../components/ConfirmDialog'

const typeBadge: Record<string, string> = {
  bearer: 'bg-blue-50 text-blue-700',
  basic: 'bg-purple-50 text-purple-700',
  oauth2_cc: 'bg-orange-50 text-orange-700',
  dynamic: 'bg-cyan-50 text-cyan-700',
  hmac: 'bg-emerald-50 text-emerald-700',
  custom_header: 'bg-stone-100 text-stone-700',
}

const statusBadge: Record<string, string> = {
  active: 'bg-green-50 text-green-700',
  disabled: 'bg-gray-100 text-gray-500',
}

export default function Credentials() {
  const navigate = useNavigate()
  const [credentials, setCredentials] = useState<Credential[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [statusFilter, setStatusFilter] = useState('')
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const size = 20

  const fetchData = async () => {
    setLoading(true)
    try {
      const res = await credentialApi.list({ status: statusFilter || undefined, page, size })
      const payload = res.data as { items: Credential[]; total: number }
      setCredentials(Array.isArray(payload.items) ? payload.items : [])
      setTotal(payload.total ?? 0)
    } catch { /* silent */ }
    setLoading(false)
  }

  useEffect(() => { fetchData() }, [page, statusFilter])

  const handleToggleStatus = async (cred: Credential) => {
    const newStatus = cred.status === 'active' ? 'disabled' : 'active'
    await credentialApi.patchStatus(cred.id, newStatus)
    fetchData()
  }

  const handleDelete = async (id: string) => {
    await credentialApi.delete(id)
    setConfirmDelete(null)
    fetchData()
  }

  return (
    <div className="space-y-5">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-gray-900">凭证管理</h2>
          <p className="text-xs text-gray-400">管理 HTTP 请求的认证凭证</p>
        </div>
        <button
          onClick={() => navigate('/credentials/new')}
          className="flex items-center gap-1.5 bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors cursor-pointer"
        >
          <Plus className="h-4 w-4" /> 新建凭证
        </button>
      </div>

      {/* Filter */}
      <div className="flex items-center">
        <div className="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5">
          {([['', '全部'], ['active', '启用'], ['disabled', '禁用']] as const).map(([val, label]) => (
            <button
              key={val}
              onClick={() => { setStatusFilter(val); setPage(1) }}
              className={`rounded-md px-3 py-1 text-sm font-medium transition-colors ${
                statusFilter === val
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex h-32 items-center justify-center">
          <div className="h-5 w-5 animate-spin rounded-full border-2 border-stone-300 border-t-transparent" />
        </div>
      ) : credentials.length === 0 ? (
        <p className="py-12 text-center text-sm text-gray-400">暂无凭证</p>
      ) : (
        <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-200 bg-gray-50">
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">名称</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">编码</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">类型</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">状态</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">配置预览</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-gray-500">创建时间</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-200">
              {credentials.map((cred) => (
                <tr key={cred.id} className="hover:bg-gray-50 cursor-pointer" onClick={() => navigate(`/credentials/${cred.id}/edit`)}>
                  <td className="px-4 py-3 font-medium text-gray-900">{cred.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-500">{`{{${cred.code}}}`}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${typeBadge[cred.type] ?? 'bg-gray-100 text-gray-600'}`}>
                      {cred.type}
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusBadge[cred.status]}`}>
                      {cred.status === 'active' ? '启用' : '禁用'}
                    </span>
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-500 max-w-[200px] truncate">{typeof cred.config_preview === 'string' ? cred.config_preview : JSON.stringify(cred.config_preview)}</td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-500">{new Date(cred.created_at).toLocaleString()}</td>
                  <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                    <div className="flex items-center justify-end gap-1">
                      <button
                        onClick={() => handleToggleStatus(cred)}
                        className="rounded-md px-2 py-1 text-xs font-medium text-gray-400 hover:bg-gray-100 hover:text-gray-900 transition-colors cursor-pointer"
                        title={cred.status === 'active' ? '禁用' : '启用'}
                      >
                        {cred.status === 'active' ? <Ban className="h-3.5 w-3.5" /> : <Check className="h-3.5 w-3.5" />}
                      </button>
                      <button
                        onClick={() => setConfirmDelete(cred.id)}
                        className="rounded-md px-2 py-1 text-xs font-medium text-gray-400 hover:bg-red-50 hover:text-red-600 transition-colors cursor-pointer"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Pagination */}
      {total > size && (
        <div className="flex items-center justify-center gap-1 text-sm text-gray-500">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1} className="rounded-lg p-1.5 hover:bg-gray-100 disabled:opacity-30 cursor-pointer">
            <ChevronLeft className="h-4 w-4" />
          </button>
          <span className="min-w-[4rem] text-center text-xs">第 {page} 页</span>
          <button onClick={() => setPage((p) => p + 1)} disabled={credentials.length < size} className="rounded-lg p-1.5 hover:bg-gray-100 disabled:opacity-30 cursor-pointer">
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      )}
      {confirmDelete && (
        <ConfirmDialog
          title="删除凭证"
          message="确认删除该凭证？此操作不可撤销。"
          confirmText="删除"
          onConfirm={() => handleDelete(confirmDelete)}
          onCancel={() => setConfirmDelete(null)}
          destructive
        />
      )}
    </div>
  )
}
