import { useCallback } from 'react'
import { XCircle } from 'lucide-react'
import { api } from '../api/client'
import { useAutoRefresh } from '../hooks/useAutoRefresh'

function ProgressBar({ label, used, limit }: { label: string; used: number; limit: number }) {
  const pct = limit > 0 ? (used / limit) * 100 : 0
  const color = pct >= 90 ? 'bg-red-500' : pct >= 70 ? 'bg-yellow-500' : 'bg-green-500'
  const textColor = pct >= 90 ? 'text-red-600' : pct >= 70 ? 'text-yellow-600' : 'text-green-600'

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <span className="text-xs font-medium text-gray-500 uppercase tracking-wider">{label}</span>
        <span className={`font-mono text-sm font-medium ${textColor}`}>
          {used} / {limit}
        </span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded-full bg-gray-100">
        <div
          className={`h-full rounded-full transition-all duration-500 ${color}`}
          style={{ width: `${Math.min(pct, 100)}%` }}
        />
      </div>
      <p className="mt-2 text-right font-mono text-xs text-gray-500">{pct.toFixed(1)}%</p>
    </div>
  )
}

export default function Quota() {
  const fetcher = useCallback(() => api.getQuota(), [])
  const { data, error, loading } = useAutoRefresh<{ data: { max_tasks: number; used_tasks: number; max_rps: number } }>(fetcher)

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

  if (!data) return null

  return (
    <div className="flex divide-x divide-stone-200 border border-stone-200 rounded-lg bg-white">
      <div className="flex-1 p-5">
        <ProgressBar label="任务数" used={data.data.used_tasks} limit={data.data.max_tasks} />
      </div>
      <div className="flex-1 p-5">
        <ProgressBar label="最大 RPS" used={0} limit={data.data.max_rps} />
      </div>
    </div>
  )
}
