import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Plus, Trash2, X } from 'lucide-react'
import { api } from '../api/client'
import { credentialApi } from '../api/credential'
import Tabs from '../components/Tabs'
import Accordion from '../components/Accordion'
import HookEditor, { PreHook, PostHook } from '../components/HookEditor'

type KV = { key: string; value: string }

function SectionHeader({ title }: { title: string }) {
  return (
    <div className="flex items-center gap-2 pb-3 mb-4 border-b border-gray-100">
      <h3 className="text-[13px] font-semibold text-gray-900 tracking-wide uppercase">{title}</h3>
    </div>
  )
}

export default function TaskEdit() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [form, setForm] = useState({
    name: '',
    cron_expr: '',
    interval_value: 60,
    interval_unit: 'seconds',
    once_at: '',
    url: '',
    method: 'POST',
    timeout_secs: 30,
    retry_count: 3,
    concurrency_policy: 'skip',
    max_concurrency: 1,
    retry_backoff: 'exponential',
    execution_retention_days: 30,
  })
  const [scheduleType, setScheduleType] = useState('interval')
  const [credentials, setCredentials] = useState<{ id: string; name: string; type: string }[]>([])
  const [selectedCredentials, setSelectedCredentials] = useState<string[]>([])
  const [headers, setHeaders] = useState<KV[]>([])
  const [bodyType, setBodyType] = useState('none')
  const [jsonBody, setJsonBody] = useState('')
  const [formFields, setFormFields] = useState<KV[]>([])
  const [preHooks, setPreHooks] = useState<PreHook[]>([])
  const [postHooks, setPostHooks] = useState<PostHook[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [fetching, setFetching] = useState(true)
  const [showCredDropdown, setShowCredDropdown] = useState(false)

  useEffect(() => {
    credentialApi.list({ status: 'active', size: 100 }).then((res: any) => {
      setCredentials(Array.isArray(res.data?.items) ? res.data.items : [])
    }).catch(() => {})
  }, [])

  useEffect(() => {
    if (!showCredDropdown) return
    const handler = () => setShowCredDropdown(false)
    document.addEventListener('click', handler)
    return () => document.removeEventListener('click', handler)
  }, [showCredDropdown])

  useEffect(() => {
    api.getTask(id!).then((res) => {
      const t = res.data
      setForm({
        name: t.name,
        cron_expr: t.cron_expr || '',
        interval_value: t.interval_value || 60,
        interval_unit: t.interval_unit || 'seconds',
        once_at: t.once_at ? new Date(t.once_at).toISOString().slice(0, 16) : '',
        url: t.url || '',
        method: t.method || 'POST',
        timeout_secs: t.timeout_secs,
        retry_count: t.retry_count,
        concurrency_policy: t.concurrency_policy || 'skip',
        max_concurrency: t.max_concurrency || 1,
        retry_backoff: t.retry_backoff || 'exponential',
        execution_retention_days: t.execution_retention_days || 30,
      })
      setScheduleType(t.schedule_type || 'interval')
      setSelectedCredentials((t as any).credential_ids || [])
      setPreHooks((t as any).pre_hooks || [])
      setPostHooks((t as any).post_hooks || [])
      const customHeaders = Object.entries(t.headers || {})
        .map(([key, value]) => ({ key, value }))
      setHeaders(customHeaders)
      if (t.content_type === 'json' && t.body) {
        setBodyType('json')
        setJsonBody(typeof t.body === 'string' ? t.body : JSON.stringify(t.body, null, 2))
      } else if (t.content_type === 'form' && t.body) {
        setBodyType('form')
        try {
          const obj = typeof t.body === 'string' ? JSON.parse(t.body) : t.body
          setFormFields(Object.entries(obj).map(([key, value]) => ({ key, value: String(value) })))
        } catch { setFormFields([]) }
      }
      setFetching(false)
    }).catch(() => {
      setError('找不到该任务')
      setFetching(false)
    })
  }, [id])

  const buildHeaders = (): Record<string, string> => {
    const h: Record<string, string> = {}
    headers.forEach(({ key, value }) => { if (key.trim()) h[key.trim()] = value })
    return h
  }

  const buildBody = (): string | undefined => {
    if (bodyType === 'json' && jsonBody.trim()) return jsonBody.trim()
    if (bodyType === 'form' && formFields.length > 0) {
      const obj: Record<string, string> = {}
      formFields.forEach(({ key, value }) => { if (key.trim()) obj[key.trim()] = value })
      return JSON.stringify(obj)
    }
    return undefined
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      const payload: Record<string, unknown> = {
        name: form.name,
        schedule_type: scheduleType,
        url: form.url,
        method: form.method,
        timeout_secs: form.timeout_secs,
        retry_count: form.retry_count,
        concurrency_policy: form.concurrency_policy,
        max_concurrency: form.max_concurrency,
        retry_backoff: form.retry_backoff,
        execution_retention_days: form.execution_retention_days,
      }
      if (scheduleType === 'cron') {
        payload.cron_expr = form.cron_expr
      } else if (scheduleType === 'interval') {
        payload.interval_value = form.interval_value
        payload.interval_unit = form.interval_unit
      } else if (scheduleType === 'once' && form.once_at) {
        payload.once_at = new Date(form.once_at).toISOString()
      }
      payload.headers = buildHeaders()
      const body = buildBody()
      if (body) payload.body = body
      payload.content_type = bodyType === 'none' ? '' : bodyType
      if (selectedCredentials.length > 0) payload.credential_ids = selectedCredentials
      if (preHooks.length > 0) payload.pre_hooks = preHooks
      if (postHooks.length > 0) payload.post_hooks = postHooks
      await api.updateTask(id!, payload)
      navigate(`/tasks/${id}`)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '更新任务失败')
    } finally {
      setLoading(false)
    }
  }

  if (fetching) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-stone-400 border-t-transparent" />
      </div>
    )
  }

  const inputCls = "w-full rounded-md border border-stone-200 bg-gray-50/60 px-3 py-2 text-sm text-gray-900 outline-none transition-all placeholder:text-gray-300 focus:bg-white focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
  const labelCls = "mb-1 block text-xs font-medium text-gray-500"

  return (
    <div className="pb-10">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button onClick={() => navigate(`/tasks/${id}`)} className="flex items-center justify-center w-8 h-8 rounded-lg border border-stone-200 text-gray-400 hover:text-gray-900 hover:border-stone-300 transition-colors cursor-pointer">
            <ArrowLeft className="h-4 w-4" />
          </button>
          <div>
            <h2 className="text-2xl font-semibold text-gray-900">编辑任务</h2>
            <p className="text-xs text-gray-400">修改任务配置后保存生效</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => navigate(`/tasks/${id}`)} className="bg-white border border-stone-200 text-gray-600 rounded-md px-4 py-2 text-sm hover:bg-stone-50 transition-colors duration-150 cursor-pointer">
            取消
          </button>
          <button type="submit" form="task-form" disabled={loading} className="bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors duration-150 disabled:opacity-50 cursor-pointer">
            {loading ? '保存中...' : '保存修改'}
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 rounded-lg border border-red-100 bg-red-50 px-4 py-2.5 text-sm text-red-600">{error}</div>
      )}

      <form id="task-form" onSubmit={handleSubmit}>
        {/* 上部两列: 基本信息 + 定时调度（对半） */}
        <div className="grid grid-cols-2 gap-5 mb-5">
          {/* 基本信息 */}
          <div className="border border-stone-200 rounded-lg bg-white p-5">
            <SectionHeader title="基本信息" />
            <div className="space-y-3">
              <div>
                <label className={labelCls}>任务名称</label>
                <input type="text" required value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} className={inputCls} />
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className={labelCls}>超时时间（秒）</label>
                  <input type="number" min={1} value={form.timeout_secs} onChange={(e) => setForm({ ...form, timeout_secs: parseInt(e.target.value) || 30 })} className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>重试次数</label>
                  <input type="number" min={0} value={form.retry_count} onChange={(e) => setForm({ ...form, retry_count: parseInt(e.target.value) || 0 })} className={inputCls} />
                </div>
              </div>
            </div>
          </div>

          {/* 定时调度 */}
          <div className="border border-stone-200 rounded-lg bg-white p-5">
            <SectionHeader title="定时调度" />
            <div className="space-y-3">
              <div>
                <label className={labelCls}>调度类型</label>
                <select value={scheduleType} onChange={(e) => setScheduleType(e.target.value)} className={inputCls}>
                  <option value="interval">固定间隔</option>
                  <option value="cron">Cron 表达式</option>
                  <option value="once">一次性</option>
                </select>
              </div>
              {scheduleType === 'cron' && (
                <div>
                  <label className={labelCls}>Cron 表达式</label>
                  <input type="text" value={form.cron_expr} onChange={(e) => setForm({ ...form, cron_expr: e.target.value })} className={inputCls + " font-mono"} />
                </div>
              )}
              {scheduleType === 'interval' && (
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className={labelCls}>间隔值</label>
                    <input type="number" min={1} value={form.interval_value} onChange={(e) => setForm({ ...form, interval_value: parseInt(e.target.value) || 1 })} className={inputCls} />
                  </div>
                  <div>
                    <label className={labelCls}>单位</label>
                    <select value={form.interval_unit} onChange={(e) => setForm({ ...form, interval_unit: e.target.value })} className={inputCls}>
                      <option value="seconds">秒</option>
                      <option value="minutes">分钟</option>
                      <option value="hours">小时</option>
                      <option value="days">天</option>
                    </select>
                  </div>
                </div>
              )}
              {scheduleType === 'once' && (
                <div>
                  <label className={labelCls}>执行时间</label>
                  <input type="datetime-local" required value={form.once_at} onChange={(e) => setForm({ ...form, once_at: e.target.value })} className={inputCls} />
                </div>
              )}
            </div>
          </div>
        </div>

        {/* 执行保护（全宽） */}
        <div className="border border-stone-200 rounded-lg bg-white p-5 mb-5">
          <SectionHeader title="执行保护" />
          <div className="grid grid-cols-4 gap-3">
            <div>
              <label className={labelCls}>并发策略</label>
              <select value={form.concurrency_policy} onChange={(e) => setForm({ ...form, concurrency_policy: e.target.value })} className={inputCls}>
                <option value="skip">跳过（默认）</option>
                <option value="queue">排队</option>
                <option value="allow">允许并发</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>最大并发数</label>
              <input type="number" min={1} value={form.max_concurrency} onChange={(e) => setForm({ ...form, max_concurrency: parseInt(e.target.value) || 1 })} className={inputCls} />
            </div>
            <div>
              <label className={labelCls}>重试退避策略</label>
              <select value={form.retry_backoff} onChange={(e) => setForm({ ...form, retry_backoff: e.target.value })} className={inputCls}>
                <option value="exponential">指数退避</option>
                <option value="fixed">固定间隔</option>
                <option value="none">不退避</option>
              </select>
            </div>
            <div>
              <label className={labelCls}>记录保留天数</label>
              <input type="number" min={1} max={90} value={form.execution_retention_days} onChange={(e) => setForm({ ...form, execution_retention_days: Math.min(90, Math.max(1, parseInt(e.target.value) || 30)) })} className={inputCls} />
            </div>
          </div>
        </div>

        {/* HTTP 请求（全宽） */}
        <div className="border border-stone-200 rounded-lg bg-white p-5 mb-5">
          <SectionHeader title="HTTP 请求" />
          <div className="space-y-4">
            {/* URL 行：方法 + URL */}
            <div className="flex gap-3 items-end">
              <div className="w-28 flex-shrink-0">
                <label className={labelCls}>方法</label>
                <select value={form.method} onChange={(e) => setForm({ ...form, method: e.target.value })} className={inputCls}>
                  {['GET', 'POST', 'PUT', 'DELETE', 'PATCH'].map((m) => <option key={m}>{m}</option>)}
                </select>
              </div>
              <div className="flex-1">
                <label className={labelCls}>目标 URL</label>
                <input type="url" required value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} className={inputCls} />
              </div>
            </div>

            {/* 凭证绑定 */}
            <div className="relative">
              <label className={labelCls}>凭证绑定</label>
              <div
                className="flex items-center gap-1 flex-wrap w-full rounded-md border border-stone-200 bg-gray-50/60 px-3 py-2 min-h-[42px] cursor-pointer hover:border-stone-400 transition-all"
                onClick={(e) => { e.stopPropagation(); setShowCredDropdown(!showCredDropdown) }}
              >
                {selectedCredentials.length === 0 && (
                  <span className="text-sm text-gray-300">选择凭证</span>
                )}
                {selectedCredentials.map((cid) => {
                  const c = credentials.find(x => x.id === cid)
                  return (
                    <span key={cid} className="inline-flex items-center gap-1 rounded-md bg-gray-100 px-2 py-0.5 text-xs text-gray-700">
                      [{c?.type}] {c?.name || cid}
                      <button type="button" onClick={(e) => { e.stopPropagation(); setSelectedCredentials(selectedCredentials.filter(x => x !== cid)) }} className="text-gray-400 hover:text-gray-700 cursor-pointer"><X className="h-3 w-3" /></button>
                    </span>
                  )
                })}
              </div>
              {showCredDropdown && (
                <div className="absolute z-10 mt-1 w-full rounded-md border border-stone-200 bg-white shadow-lg max-h-48 overflow-y-auto">
                  {credentials.filter(c => !selectedCredentials.includes(c.id)).length === 0 ? (
                    <div className="px-3 py-2 text-sm text-gray-400">暂无可用凭证</div>
                  ) : (
                    credentials.filter(c => !selectedCredentials.includes(c.id)).map((c) => (
                      <div
                        key={c.id}
                        className="px-3 py-2 text-sm text-gray-700 hover:bg-stone-50 cursor-pointer"
                        onClick={() => { setSelectedCredentials([...selectedCredentials, c.id]); setShowCredDropdown(false) }}
                      >
                        [{c.type}] {c.name}
                      </div>
                    ))
                  )}
                </div>
              )}
            </div>

            {/* Headers */}
            <Accordion title="自定义 Headers" defaultOpen={false}>
              <div className="space-y-2">
                {headers.map((h, i) => (
                  <div key={i} className="flex gap-2">
                    <input type="text" value={h.key} onChange={(e) => { const n = [...headers]; n[i].key = e.target.value; setHeaders(n) }} placeholder="Key" className={inputCls} />
                    <input type="text" value={h.value} onChange={(e) => { const n = [...headers]; n[i].value = e.target.value; setHeaders(n) }} placeholder="Value" className={inputCls} />
                    <button type="button" onClick={() => setHeaders(headers.filter((_, j) => j !== i))} className="flex-shrink-0 p-2 text-gray-300 hover:text-red-500 transition-colors cursor-pointer">
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                ))}
                <button type="button" onClick={() => setHeaders([...headers, { key: '', value: '' }])} className="flex items-center gap-1 text-xs font-medium text-gray-600 hover:text-gray-900 cursor-pointer">
                  <Plus className="h-3.5 w-3.5" /> 添加 Header
                </button>
              </div>
            </Accordion>

            {/* 请求体 */}
            <Accordion title="请求体" defaultOpen={false}>
              <div className="grid grid-cols-[200px_1fr] gap-4 items-start">
                <div>
                  <label className={labelCls}>格式</label>
                  <select value={bodyType} onChange={(e) => setBodyType(e.target.value)} className={inputCls}>
                    <option value="none">无</option>
                    <option value="json">JSON</option>
                    <option value="form">Form</option>
                  </select>
                </div>
                <div className="min-h-0">
                  {bodyType === 'json' && (
                    <textarea value={jsonBody} onChange={(e) => setJsonBody(e.target.value)} rows={8} className={inputCls + " font-mono !leading-relaxed resize-y"} />
                  )}
                  {bodyType === 'form' && (
                    <div className="space-y-2">
                      {formFields.map((f, i) => (
                        <div key={i} className="flex gap-2">
                          <input type="text" value={f.key} onChange={(e) => { const n = [...formFields]; n[i].key = e.target.value; setFormFields(n) }} placeholder="Key" className={inputCls} />
                          <input type="text" value={f.value} onChange={(e) => { const n = [...formFields]; n[i].value = e.target.value; setFormFields(n) }} placeholder="Value" className={inputCls} />
                          <button type="button" onClick={() => setFormFields(formFields.filter((_, j) => j !== i))} className="flex-shrink-0 p-2 text-gray-300 hover:text-red-500 cursor-pointer">
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      ))}
                      <button type="button" onClick={() => setFormFields([...formFields, { key: '', value: '' }])} className="flex items-center gap-1 text-xs font-medium text-gray-600 hover:text-gray-900 cursor-pointer">
                        <Plus className="h-3.5 w-3.5" /> 添加字段
                      </button>
                    </div>
                  )}
                  {bodyType === 'none' && (
                    <div className="flex items-center h-[38px] text-xs text-gray-300">不发送请求体</div>
                  )}
                </div>
              </div>
            </Accordion>
          </div>
        </div>

        {/* Hooks（全宽 Tabs） */}
        <div className="border border-stone-200 rounded-lg bg-white p-5">
          <SectionHeader title="Hooks" />
          <Tabs
            items={[
              {
                key: 'pre',
                label: '前置 Hook',
                content: <HookEditor value={preHooks} onChange={setPreHooks} mode="pre" />
              },
              {
                key: 'post',
                label: '后置 Hook',
                content: <HookEditor value={postHooks} onChange={setPostHooks} mode="post" />
              },
            ]}
          />
        </div>
      </form>
    </div>
  )
}
