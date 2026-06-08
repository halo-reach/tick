import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'
import { api } from '../api/client'

export default function VariableCreate() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const isEdit = !!id
  const [key, setKey] = useState('')
  const [value, setValue] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (isEdit) {
      api.getVariable(id!).then((res) => {
        setKey(res.data.key)
        setValue(res.data.value)
      }).catch(() => setError('加载变量失败'))
    }
  }, [id, isEdit])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      if (isEdit) {
        await api.updateVariable(id!, { key, value })
      } else {
        await api.createVariable({ key, value })
      }
      navigate('/variables')
    } catch (err: unknown) {
      if (err instanceof Error && err.message.includes('409')) {
        setError('变量 Key 已存在，请使用其他名称')
      } else {
        setError(err instanceof Error ? err.message : '操作失败')
      }
    } finally {
      setLoading(false)
    }
  }

  const inputCls = "w-full rounded-md border border-stone-200 bg-gray-50/60 px-3 py-2 text-sm text-gray-900 outline-none transition-all placeholder:text-gray-300 focus:bg-white focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
  const labelCls = "mb-1 block text-xs font-medium text-gray-500"

  return (
    <div className="pb-10">
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button onClick={() => navigate('/variables')} className="flex items-center justify-center w-8 h-8 rounded-lg border border-stone-200 text-gray-400 hover:text-gray-900 hover:border-stone-300 transition-colors cursor-pointer">
            <ArrowLeft className="h-4 w-4" />
          </button>
          <div>
            <h2 className="text-2xl font-semibold text-gray-900">{isEdit ? '编辑变量' : '新建变量'}</h2>
            <p className="text-xs text-gray-400">{isEdit ? '修改变量配置' : '创建一个可在任务中引用的变量'}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => navigate('/variables')} className="bg-white border border-stone-200 text-gray-600 rounded-md px-4 py-2 text-sm hover:bg-stone-50 transition-colors duration-150 cursor-pointer">
            取消
          </button>
          <button type="submit" form="variable-form" disabled={loading} className="bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors duration-150 disabled:opacity-50 cursor-pointer">
            {loading ? '保存中...' : (isEdit ? '保存修改' : '创建变量')}
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-100 bg-red-50 px-4 py-2.5 text-sm text-red-600">{error}</div>
      )}

      <form id="variable-form" onSubmit={handleSubmit}>
        <div className="border border-stone-200 rounded-lg bg-white p-5">
          <div className="space-y-4">
            <div>
              <label className={labelCls}>变量 Key</label>
              <input type="text" required value={key} onChange={(e) => setKey(e.target.value)} placeholder="例：API_SECRET" className={inputCls + " font-mono"} />
            </div>
            <div>
              <label className={labelCls}>变量值</label>
              <textarea required value={value} onChange={(e) => setValue(e.target.value)} placeholder="输入变量值" rows={4} className={inputCls + " resize-y"} />
            </div>
          </div>
        </div>
      </form>
    </div>
  )
}
