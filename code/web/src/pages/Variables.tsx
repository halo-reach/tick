import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Plus, Pencil, Trash2 } from 'lucide-react'
import { api } from '../api/client'
import ConfirmDialog from '../components/ConfirmDialog'

interface Variable {
  id: string
  key: string
  value: string
  created_at: string
}

export default function Variables() {
  const navigate = useNavigate()
  const [variables, setVariables] = useState<Variable[]>([])
  const [loading, setLoading] = useState(true)
  const [confirmDelete, setConfirmDelete] = useState<{ id: string; key: string } | null>(null)

  const fetchVariables = () => {
    api.listVariables().then((res) => {
      setVariables(res.data || [])
      setLoading(false)
    }).catch(() => setLoading(false))
  }

  useEffect(() => { fetchVariables() }, [])

  const handleDelete = (id: string) => {
    api.deleteVariable(id).then(() => {
      setVariables(variables.filter((v) => v.id !== id))
      setConfirmDelete(null)
    })
  }

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-stone-400 border-t-transparent" />
      </div>
    )
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-semibold text-gray-900">变量管理</h2>
          <p className="text-xs text-gray-400">管理可在任务中引用的变量</p>
        </div>
        <button onClick={() => navigate('/variables/new')} className="flex items-center gap-2 bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors cursor-pointer">
          <Plus className="h-4 w-4" /> 新建变量
        </button>
      </div>

      <div className="border border-stone-200 rounded-lg bg-white">
        {variables.length === 0 ? (
          <div className="p-12 text-center text-sm text-gray-400">暂无变量，点击上方按钮创建</div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-100 text-left text-xs font-medium text-gray-500">
                <th className="px-5 py-3">名称 / Key</th>
                <th className="px-5 py-3">值 / Value</th>
                <th className="px-5 py-3">创建时间</th>
                <th className="px-5 py-3 w-24">操作</th>
              </tr>
            </thead>
            <tbody>
              {variables.map((v) => (
                <tr key={v.id} className="border-b border-gray-100 last:border-0">
                  <td className="px-5 py-3 font-mono text-gray-900">{v.key}</td>
                  <td className="px-5 py-3 text-gray-600 max-w-xs truncate">{v.value}</td>
                  <td className="px-5 py-3 text-gray-400">{new Date(v.created_at).toLocaleString('zh-CN')}</td>
                  <td className="px-5 py-3">
                    <div className="flex items-center gap-2">
                      <button onClick={() => navigate(`/variables/${v.id}/edit`)} className="p-1.5 text-gray-400 hover:text-gray-900 transition-colors cursor-pointer">
                        <Pencil className="h-3.5 w-3.5" />
                      </button>
                      <button onClick={() => setConfirmDelete({ id: v.id, key: v.key })} className="p-1.5 text-gray-400 hover:text-red-500 transition-colors cursor-pointer">
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      {confirmDelete && (
        <ConfirmDialog
          title="删除变量"
          message={`确定要删除变量「${confirmDelete.key}」吗？`}
          confirmText="删除"
          onConfirm={() => handleDelete(confirmDelete.id)}
          onCancel={() => setConfirmDelete(null)}
          destructive
        />
      )}
    </div>
  )
}
