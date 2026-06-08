import { useState, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, ChevronLeft, ChevronRight, XCircle, Pencil, Trash2, Pause, Play, Zap, RefreshCw, ChevronDown, ChevronUp } from 'lucide-react'
import { api, type Task, type Execution } from '../api/client'
import { useAutoRefresh } from '../hooks/useAutoRefresh'
import Tabs from '../components/Tabs'

interface HookResult {
  type: string
  status: string
  error?: string
}

interface ExecutionWithHooks extends Execution {
  hooks_result?: {
    pre_hooks?: HookResult[]
    post_hooks?: HookResult[]
  }
}

const execStatusStyle: Record<string, string> = {
  success: 'bg-green-50 text-green-700',
  failed: 'bg-red-50 text-red-700',
  timeout: 'bg-yellow-50 text-yellow-700',
  retrying: 'bg-stone-50 text-stone-700',
}

const taskStatusStyle: Record<string, string> = {
  active: 'bg-green-50 text-green-700 border-green-200',
  paused: 'bg-yellow-50 text-yellow-700 border-yellow-200',
  deleted: 'bg-red-50 text-red-700 border-red-200',
}

function getScheduleDisplay(task: Task): string {
  if (task.schedule_type === 'cron') return task.cron_expr || '—'
  if (task.schedule_type === 'interval') return `每 ${task.interval_value} ${task.interval_unit}`
  return '一次性'
}

export default function TaskDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [histPage, setHistPage] = useState(1)
  const [expandedExec, setExpandedExec] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState('')
  const [refreshing, setRefreshing] = useState(false)
  const histSize = 20

  const taskFetcher = useCallback(() => api.getTask(id!), [id])
  const histFetcher = useCallback(
    () => api.getHistory(id!, histPage, histSize),
    [id, histPage],
  )

  const { data: taskRes, error: taskErr, loading: taskLoading, refresh: refreshTask } = useAutoRefresh<{ data: Task }>(taskFetcher)
  const { data: hist, error: histErr, loading: histLoading, refresh: refreshHist } = useAutoRefresh<{ data: ExecutionWithHooks[] }>(histFetcher)

  const task = taskRes?.data

  const handleRefresh = async () => {
    setRefreshing(true)
    refreshTask()
    refreshHist()
    setTimeout(() => setRefreshing(false), 600)
  }

  const handleAction = async (action: 'pause' | 'resume' | 'delete' | 'trigger') => {
    setActionLoading(action)
    try {
      if (action === 'pause') await api.pauseTask(id!)
      else if (action === 'resume') await api.resumeTask(id!)
      else if (action === 'trigger') await api.triggerTask(id!)
      else if (action === 'delete') {
        await api.deleteTask(id!)
        navigate('/tasks')
        return
      }
      refreshTask()
      refreshHist()
    } catch { /* silent */ }
    setActionLoading('')
  }

  if (taskLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-stone-300 border-t-transparent" />
      </div>
    )
  }

  if (taskErr) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-2 text-red-500">
        <XCircle className="h-8 w-8" />
        <p className="text-sm">{taskErr}</p>
      </div>
    )
  }

  if (!task) return null

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <button
          onClick={() => navigate('/tasks')}
          className="flex items-center justify-center w-8 h-8 rounded-lg border border-stone-200 text-gray-400 hover:text-gray-900 hover:border-stone-300 transition-colors cursor-pointer"
        >
          <ArrowLeft className="h-4 w-4" />
        </button>
        <div className="flex items-center gap-2">
          <button
            onClick={() => handleAction('trigger')}
            disabled={!!actionLoading}
            className="flex items-center gap-1.5 rounded-md bg-black text-white px-3 py-2 text-sm font-medium hover:bg-gray-800 transition-colors cursor-pointer disabled:opacity-50"
          >
            <Zap className="h-3.5 w-3.5" /> {actionLoading === 'trigger' ? '执行中...' : '立即执行'}
          </button>
          <button
            onClick={() => navigate(`/tasks/${id}/edit`)}
            className="flex items-center gap-1.5 rounded-md border border-stone-200 px-3 py-2 text-sm text-gray-600 hover:bg-stone-50 hover:border-stone-300 transition-colors cursor-pointer"
          >
            <Pencil className="h-3.5 w-3.5" /> 编辑
          </button>
          {task.status === 'active' && (
            <button
              onClick={() => handleAction('pause')}
              disabled={!!actionLoading}
              className="flex items-center gap-1.5 rounded-md border border-yellow-200 bg-yellow-50 px-3 py-2 text-sm font-medium text-yellow-700 hover:bg-yellow-100 transition-colors cursor-pointer disabled:opacity-50"
            >
              <Pause className="h-3.5 w-3.5" /> 暂停
            </button>
          )}
          {task.status === 'paused' && (
            <button
              onClick={() => handleAction('resume')}
              disabled={!!actionLoading}
              className="flex items-center gap-1.5 rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm font-medium text-green-700 hover:bg-green-100 transition-colors cursor-pointer disabled:opacity-50"
            >
              <Play className="h-3.5 w-3.5" /> 恢复
            </button>
          )}
          <button
            onClick={() => handleAction('delete')}
            disabled={!!actionLoading}
            className="flex items-center gap-1.5 rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm font-medium text-red-600 hover:bg-red-100 transition-colors cursor-pointer disabled:opacity-50"
          >
            <Trash2 className="h-3.5 w-3.5" /> 删除
          </button>
        </div>
      </div>

      {/* Task Info Card */}
      <div className="rounded-lg border border-stone-200 bg-white overflow-hidden">
        {/* Title row */}
        <div className="flex items-center gap-3 border-b border-gray-100 px-6 py-4">
          <h1 className="text-lg font-semibold text-gray-900">{task.name}</h1>
          <span className={`inline-flex rounded-full border px-2.5 py-0.5 text-xs font-medium ${taskStatusStyle[task.status] ?? ''}`}>
            {task.status === 'active' ? '运行中' : task.status === 'paused' ? '已暂停' : task.status}
          </span>
          <span className="ml-auto font-mono text-xs text-gray-400">{task.id}</span>
        </div>

        <div className="px-6 py-5">
          <Tabs
            items={[
              {
                key: 'overview',
                label: '概览',
                content: (
                  <div className="grid grid-cols-3 gap-x-8 gap-y-4 text-sm lg:grid-cols-6">
                    <div>
                      <dt className="text-xs text-gray-500">调度类型</dt>
                      <dd className="mt-1 font-medium text-gray-900">
                        {task.schedule_type === 'cron' ? 'Cron' : task.schedule_type === 'interval' ? '固定间隔' : '一次性'}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs text-gray-500">定时周期</dt>
                      <dd className="mt-1 font-mono text-sm text-gray-900">{getScheduleDisplay(task)}</dd>
                    </div>
                    <div>
                      <dt className="text-xs text-gray-500">超时时间</dt>
                      <dd className="mt-1 font-mono text-sm text-gray-900">{task.timeout_secs}s</dd>
                    </div>
                    <div>
                      <dt className="text-xs text-gray-500">重试次数</dt>
                      <dd className="mt-1 font-mono text-sm text-gray-900">{task.retry_count}</dd>
                    </div>
                    <div>
                      <dt className="text-xs text-gray-500">下次触发</dt>
                      <dd className="mt-1 font-mono text-sm text-gray-900">
                        {task.next_trigger_at ? new Date(task.next_trigger_at).toLocaleString() : '—'}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs text-gray-500">创建时间</dt>
                      <dd className="mt-1 font-mono text-sm text-gray-900">{new Date(task.created_at).toLocaleString()}</dd>
                    </div>
                  </div>
                ),
              },
              {
                key: 'http',
                label: 'HTTP 请求',
                content: (
                  <div className="space-y-4">
                    {task.url && (
                      <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-sm lg:grid-cols-4">
                        <div>
                          <dt className="text-xs text-gray-500">请求方法</dt>
                          <dd className="mt-1 font-mono text-sm text-gray-900">{task.method || 'POST'}</dd>
                        </div>
                        <div className="col-span-2">
                          <dt className="text-xs text-gray-500">目标 URL</dt>
                          <dd className="mt-1 font-mono text-sm text-gray-900 break-all">{task.url}</dd>
                        </div>
                        {task.content_type && (
                          <div>
                            <dt className="text-xs text-gray-500">请求体格式</dt>
                            <dd className="mt-1 font-mono text-sm text-gray-900">{task.content_type === 'json' ? 'JSON' : 'Form'}</dd>
                          </div>
                        )}
                      </div>
                    )}
                    {task.headers && Object.keys(task.headers).length > 0 ? (
                      <div>
                        <dt className="text-xs text-gray-500 mb-1">Headers</dt>
                        <dd className="rounded-lg bg-gray-50 p-3 font-mono text-xs text-gray-900 space-y-1">
                          {Object.entries(task.headers).map(([k, v]) => (
                            <div key={k}><span className="text-gray-500">{k}:</span> {v}</div>
                          ))}
                        </dd>
                      </div>
                    ) : (
                      <p className="text-sm text-gray-400">暂无自定义 Headers</p>
                    )}
                    {task.body ? (
                      <div>
                        <dt className="text-xs text-gray-500 mb-1">请求体</dt>
                        <dd className="rounded-lg bg-gray-50 p-3 font-mono text-xs text-gray-900 whitespace-pre-wrap break-all">
                          {typeof task.body === 'string' ? task.body : JSON.stringify(task.body, null, 2)}
                        </dd>
                      </div>
                    ) : (
                      <p className="text-sm text-gray-400">暂无请求体</p>
                    )}
                  </div>
                ),
              },
              {
                key: 'hooks',
                label: 'Hooks',
                content: (
                  <div className="space-y-4">
                    {(task as any).pre_hooks?.length > 0 ? (
                      <div>
                        <dt className="text-xs text-gray-500 mb-1">前置 Hook</dt>
                        <dd className="rounded-lg bg-gray-50 p-3 font-mono text-xs text-gray-900 space-y-1">
                          {(task as any).pre_hooks.map((h: any, i: number) => (
                            <div key={i}><span className="text-gray-500">{h.type}:</span> {h.url || h.credential_code || '—'}</div>
                          ))}
                        </dd>
                      </div>
                    ) : (
                      <p className="text-sm text-gray-400">暂无前置 Hook</p>
                    )}
                    {(task as any).post_hooks?.length > 0 ? (
                      <div>
                        <dt className="text-xs text-gray-500 mb-1">后置 Hook</dt>
                        <dd className="rounded-lg bg-gray-50 p-3 font-mono text-xs text-gray-900 space-y-1">
                          {(task as any).post_hooks.map((h: any, i: number) => (
                            <div key={i}><span className="text-gray-500">{h.type}:</span> {h.url || h.credential_code || '—'} {h.on && `(on: ${h.on})`}</div>
                          ))}
                        </dd>
                      </div>
                    ) : (
                      <p className="text-sm text-gray-400">暂无后置 Hook</p>
                    )}
                  </div>
                ),
              },
            ]}
          />
        </div>
      </div>

      {/* Execution History */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-xs font-medium text-gray-500 uppercase tracking-wider">
            执行历史
          </h2>
          <div className="flex items-center gap-2">
            <button
              onClick={handleRefresh}
              className="rounded-lg p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600 transition-colors cursor-pointer"
            >
              <RefreshCw className={`h-4 w-4 ${refreshing ? 'animate-spin' : ''}`} />
            </button>
            {hist?.data && hist.data.length > 0 && (
              <div className="flex items-center gap-1 text-sm text-gray-500">
                <button
                  onClick={() => setHistPage((p) => Math.max(1, p - 1))}
                  disabled={histPage === 1}
                  className="rounded-lg p-1.5 hover:bg-gray-100 disabled:opacity-30 cursor-pointer"
                >
                  <ChevronLeft className="h-4 w-4" />
                </button>
                <span className="min-w-[4rem] text-center text-xs">第 {histPage} 页</span>
                <button
                  onClick={() => setHistPage((p) => p + 1)}
                  disabled={!hist?.data || hist.data.length < histSize}
                  className="rounded-lg p-1.5 hover:bg-gray-100 disabled:opacity-30 cursor-pointer"
                >
                  <ChevronRight className="h-4 w-4" />
                </button>
              </div>
            )}
          </div>
        </div>

        {histLoading ? (
          <div className="flex h-32 items-center justify-center">
            <div className="h-5 w-5 animate-spin rounded-full border-2 border-stone-300 border-t-transparent" />
          </div>
        ) : histErr ? (
          <p className="text-sm text-red-500">{histErr}</p>
        ) : !hist?.data?.length ? (
          <p className="py-8 text-center text-sm text-gray-500">暂无执行记录</p>
        ) : (
          <div className="overflow-hidden rounded-lg border border-stone-200 bg-white">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-gray-200 bg-gray-50">
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">状态</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">HTTP</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">耗时</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Hook</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">触发时间</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {hist.data.map((exec) => {
                  const isExpanded = expandedExec === exec.id
                  const hasDetail = exec.error_msg || exec.response_body || exec.request_body || exec.request_headers
                  return (
                    <tr key={exec.id} className="group">
                      <td colSpan={5} className="p-0">
                        <div
                          className={`flex items-center px-4 py-3 ${hasDetail ? 'cursor-pointer hover:bg-gray-50' : ''}`}
                          onClick={() => hasDetail && setExpandedExec(isExpanded ? null : exec.id)}
                        >
                          <div className="flex-1 grid grid-cols-5 items-center">
                            <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium w-fit ${execStatusStyle[exec.status] ?? ''}`}>
                              {exec.status}
                              {exec.is_makeup && ' (补偿)'}
                            </span>
                            <span className="font-mono text-xs text-gray-500">{exec.status_code || '—'}</span>
                            <span className="font-mono text-xs text-gray-500">{exec.duration_ms}ms</span>
                            <span>
                              {exec.hooks_result ? (
                                <span className="flex gap-1 flex-wrap">
                                  {exec.hooks_result.pre_hooks?.map((h, i) => (
                                    <span key={`pre-${i}`} className={`inline-flex rounded-full px-1.5 py-0.5 text-[10px] font-medium ${h.status === 'success' ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`} title={h.error || ''}>
                                      前{i + 1}
                                    </span>
                                  ))}
                                  {exec.hooks_result.post_hooks?.map((h, i) => (
                                    <span key={`post-${i}`} className={`inline-flex rounded-full px-1.5 py-0.5 text-[10px] font-medium ${h.status === 'success' ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`} title={h.error || ''}>
                                      后{i + 1}
                                    </span>
                                  ))}
                                </span>
                              ) : (
                                <span className="text-xs text-gray-300">—</span>
                              )}
                            </span>
                            <span className="font-mono text-xs text-gray-500 flex items-center justify-between">
                              {new Date(exec.trigger_time).toLocaleString()}
                              {hasDetail && (
                                isExpanded ? <ChevronUp className="h-3.5 w-3.5 text-gray-400" /> : <ChevronDown className="h-3.5 w-3.5 text-gray-400" />
                              )}
                            </span>
                          </div>
                        </div>
                        {isExpanded && hasDetail && (
                          <div className="px-4 pb-3 space-y-2 border-t border-gray-100 bg-gray-50/50">
                            {exec.request_headers && (
                              <div className="pt-2">
                                <span className="text-[10px] font-medium text-gray-500 uppercase tracking-wider">请求头</span>
                                <pre className="mt-0.5 text-xs text-gray-700 font-mono whitespace-pre-wrap break-all bg-white rounded p-2 border border-gray-200 max-h-40 overflow-auto">
                                  {(() => {
                                    try { return JSON.stringify(JSON.parse(exec.request_headers), null, 2) } catch { return exec.request_headers }
                                  })()}
                                </pre>
                              </div>
                            )}
                            {exec.request_body && (
                              <div className="pt-2">
                                <span className="text-[10px] font-medium text-gray-500 uppercase tracking-wider">请求内容</span>
                                <pre className="mt-0.5 text-xs text-gray-700 font-mono whitespace-pre-wrap break-all bg-white rounded p-2 border border-gray-200 max-h-40 overflow-auto">
                                  {(() => {
                                    try { return JSON.stringify(JSON.parse(exec.request_body), null, 2) } catch { return exec.request_body }
                                  })()}
                                </pre>
                              </div>
                            )}
                            {exec.error_msg && (
                              <div className="pt-2">
                                <span className="text-[10px] font-medium text-red-600 uppercase tracking-wider">错误信息</span>
                                <p className="mt-0.5 text-xs text-red-700 font-mono">{exec.error_msg}</p>
                              </div>
                            )}
                            {exec.response_body && (
                              <div>
                                <span className="text-[10px] font-medium text-gray-500 uppercase tracking-wider">响应内容</span>
                                <pre className="mt-0.5 text-xs text-gray-700 font-mono whitespace-pre-wrap break-all bg-white rounded p-2 border border-gray-200 max-h-40 overflow-auto">
                                  {(() => {
                                    try { return JSON.stringify(JSON.parse(exec.response_body), null, 2) } catch { return exec.response_body }
                                  })()}
                                </pre>
                              </div>
                            )}
                          </div>
                        )}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
