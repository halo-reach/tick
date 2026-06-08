import { useState, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronLeft, ChevronRight, XCircle, Plus } from 'lucide-react'
import { api, type Task } from '../api/client'
import { useAutoRefresh } from '../hooks/useAutoRefresh'

const statusColors: Record<string, string> = {
  active: 'bg-green-50 text-green-700',
  paused: 'bg-yellow-50 text-yellow-700',
  deleted: 'bg-red-50 text-red-700',
}

const typeLabels: Record<string, string> = {
  cron: 'Cron',
  interval: '固定间隔',
  once: '一次性',
}

export default function TaskList() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const navigate = useNavigate()

  const fetcher = useCallback(() => api.getTasks(page, pageSize), [page])
  const { data, error, loading } = useAutoRefresh<{ data: Task[]; total: number }>(fetcher)

  if (loading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-stone-300 border-t-transparent" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-2 text-red-500">
        <XCircle className="h-8 w-8" />
        <p className="text-sm">{error}</p>
      </div>
    )
  }

  if (!data || !data.data || data.data.length === 0) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-3 text-gray-500">
        <p className="text-sm">暂无任务</p>
        <button
          onClick={() => navigate('/tasks/new')}
          className="flex items-center gap-1.5 bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors duration-150 cursor-pointer"
        >
          <Plus className="h-4 w-4" /> 创建第一个任务
        </button>
      </div>
    )
  }

  const totalPages = Math.ceil(data.total / pageSize)

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-gray-900">任务列表</h2>
        <button
          onClick={() => navigate('/tasks/new')}
          className="flex items-center gap-1.5 bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors duration-150 cursor-pointer"
        >
          <Plus className="h-4 w-4" /> 创建任务
        </button>
      </div>

      <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-stone-200 bg-white">
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">名称</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">类型</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">下次触发</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-stone-200">
            {data.data.map((task) => (
              <tr
                key={task.id}
                onClick={() => navigate(`/tasks/${task.id}`)}
                className="transition-colors hover:bg-stone-50 cursor-pointer"
              >
                <td className="px-4 py-3 font-medium text-gray-900">{task.name}</td>
                <td className="px-4 py-3">
                  <span className="rounded bg-gray-100 px-2 py-0.5 font-mono text-xs text-gray-600">
                    {typeLabels[task.schedule_type] ?? task.schedule_type}
                  </span>
                </td>
                <td className="px-4 py-3">
                  <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${statusColors[task.status] ?? ''}`}>
                    {task.status}
                  </span>
                </td>
                <td className="px-4 py-3 font-mono text-xs text-gray-500">
                  {task.next_trigger_at ? new Date(task.next_trigger_at).toLocaleString() : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-between text-sm text-gray-500">
          <span>
            第 {page} 页，共 {totalPages} 页（{data.total} 个任务）
          </span>
          <div className="flex gap-1">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
              className="rounded-lg p-1.5 hover:bg-gray-100 disabled:opacity-30 cursor-pointer transition-colors"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className="rounded-lg p-1.5 hover:bg-gray-100 disabled:opacity-30 cursor-pointer transition-colors"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
