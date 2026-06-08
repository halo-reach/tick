import { Plus, Trash2, ChevronUp, ChevronDown } from 'lucide-react'
import CredentialSelect from './CredentialSelect'

interface PreHook {
  type: 'credential' | 'http'
  credential_id?: string
  inject?: { location: string; key: string; prefix?: string }
  url?: string
  method?: string
  headers?: Record<string, string>
  body?: string
  extract?: { path: string; as: string }
  timeout_secs?: number
}

interface PostHook {
  type: 'http' | 'feishu'
  url?: string
  method?: string
  headers?: Record<string, string>
  body?: string
  webhook_url?: string
  message_body?: string
  timeout_secs?: number
}

type Hook = PreHook | PostHook

interface PreProps {
  value: PreHook[]
  onChange: (hooks: PreHook[]) => void
  mode: 'pre'
}

interface PostProps {
  value: PostHook[]
  onChange: (hooks: PostHook[]) => void
  mode: 'post'
}

type Props = PreProps | PostProps

const inputCls = "w-full rounded-md border border-stone-200 bg-gray-50/60 px-3 py-2 text-sm text-gray-900 outline-none transition-all placeholder:text-gray-300 focus:bg-white focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
const labelCls = "mb-1 block text-xs font-medium text-gray-500"

export default function HookEditor({ value, onChange, mode }: Props) {
  const maxHooks = 5
  const hooks = value as Hook[]
  const setHooks = onChange as (hooks: Hook[]) => void

  const addHook = () => {
    if (hooks.length >= maxHooks) return
    const defaultType = mode === 'pre' ? 'credential' : 'http'
    setHooks([...hooks, { type: defaultType, timeout_secs: 10 } as Hook])
  }

  const removeHook = (idx: number) => {
    setHooks(hooks.filter((_, i) => i !== idx))
  }

  const updateHook = (idx: number, patch: Partial<Hook>) => {
    const updated = [...hooks]
    updated[idx] = { ...updated[idx], ...patch } as Hook
    setHooks(updated)
  }

  const moveHook = (idx: number, dir: -1 | 1) => {
    const target = idx + dir
    if (target < 0 || target >= hooks.length) return
    const updated = [...hooks]
    ;[updated[idx], updated[target]] = [updated[target], updated[idx]]
    setHooks(updated)
  }

  const typeOptions = mode === 'pre'
    ? [{ value: 'credential', label: '凭证注入' }, { value: 'http', label: 'HTTP 请求' }]
    : [{ value: 'http', label: 'HTTP 请求' }, { value: 'feishu', label: '飞书通知' }]

  return (
    <div className="space-y-3">
      {hooks.map((hook, idx) => (
        <div key={idx} className="rounded-lg border border-gray-100 bg-gray-50/40 p-4 space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="text-xs font-medium text-gray-400">#{idx + 1}</span>
              <select
                value={hook.type}
                onChange={(e) => updateHook(idx, { type: e.target.value as Hook['type'] })}
                className={inputCls + " !w-36"}
              >
                {typeOptions.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
              </select>
            </div>
            <div className="flex items-center gap-1">
              <button type="button" onClick={() => moveHook(idx, -1)} disabled={idx === 0} className="p-1 text-gray-400 hover:text-gray-700 disabled:opacity-30 cursor-pointer">
                <ChevronUp className="h-4 w-4" />
              </button>
              <button type="button" onClick={() => moveHook(idx, 1)} disabled={idx === hooks.length - 1} className="p-1 text-gray-400 hover:text-gray-700 disabled:opacity-30 cursor-pointer">
                <ChevronDown className="h-4 w-4" />
              </button>
              <button type="button" onClick={() => removeHook(idx)} className="p-1 text-gray-300 hover:text-red-500 cursor-pointer">
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          </div>

          {hook.type === 'credential' && (
            <div className="space-y-3">
              <div>
                <label className={labelCls}>凭证</label>
                <CredentialSelect value={(hook as PreHook).credential_id || ''} onChange={(id) => updateHook(idx, { credential_id: id })} />
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className={labelCls}>注入位置</label>
                  <select value={(hook as PreHook).inject?.location || 'header'} onChange={(e) => updateHook(idx, { inject: { ...((hook as PreHook).inject || { key: '', location: 'header' }), location: e.target.value } })} className={inputCls}>
                    <option value="header">Header</option>
                    <option value="query">Query</option>
                    <option value="body">Body</option>
                  </select>
                </div>
                <div>
                  <label className={labelCls}>Key</label>
                  <input type="text" value={(hook as PreHook).inject?.key || ''} onChange={(e) => updateHook(idx, { inject: { ...((hook as PreHook).inject || { location: 'header', key: '' }), key: e.target.value } })} placeholder="Authorization" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>前缀</label>
                  <input type="text" value={(hook as PreHook).inject?.prefix || ''} onChange={(e) => updateHook(idx, { inject: { ...((hook as PreHook).inject || { location: 'header', key: '' }), prefix: e.target.value } })} placeholder="Bearer " className={inputCls} />
                </div>
              </div>
            </div>
          )}

          {hook.type === 'http' && (
            <div className="space-y-3">
              <div className="grid grid-cols-[1fr_100px] gap-3">
                <div>
                  <label className={labelCls}>URL</label>
                  <input type="url" value={hook.url || ''} onChange={(e) => updateHook(idx, { url: e.target.value })} placeholder="https://example.com/hook" className={inputCls} />
                </div>
                <div>
                  <label className={labelCls}>方法</label>
                  <select value={hook.method || 'POST'} onChange={(e) => updateHook(idx, { method: e.target.value })} className={inputCls}>
                    {['GET', 'POST', 'PUT'].map((m) => <option key={m}>{m}</option>)}
                  </select>
                </div>
              </div>
              <div>
                <label className={labelCls}>请求体（JSON）</label>
                <textarea value={hook.body || ''} onChange={(e) => updateHook(idx, { body: e.target.value })} rows={2} className={inputCls + " font-mono resize-y"} />
              </div>
              {mode === 'pre' && (
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className={labelCls}>提取路径</label>
                    <input type="text" value={(hook as PreHook).extract?.path || ''} onChange={(e) => updateHook(idx, { extract: { path: e.target.value, as: (hook as PreHook).extract?.as || '' } })} placeholder="data.token" className={inputCls + " font-mono"} />
                  </div>
                  <div>
                    <label className={labelCls}>注入为</label>
                    <input type="text" value={(hook as PreHook).extract?.as || ''} onChange={(e) => updateHook(idx, { extract: { path: (hook as PreHook).extract?.path || '', as: e.target.value } })} placeholder="auth_token" className={inputCls} />
                  </div>
                </div>
              )}
            </div>
          )}

          {hook.type === 'feishu' && (
            <div className="space-y-3">
              <div>
                <label className={labelCls}>Webhook URL</label>
                <input type="url" value={(hook as PostHook).webhook_url || ''} onChange={(e) => updateHook(idx, { webhook_url: e.target.value })} placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/xxx" className={inputCls} />
              </div>
              <div>
                <label className={labelCls}>消息模板（JSON）</label>
                <textarea value={(hook as PostHook).message_body || ''} onChange={(e) => updateHook(idx, { message_body: e.target.value })} rows={3} placeholder='{"msg_type":"text","content":{"text":"任务执行完成"}}' className={inputCls + " font-mono resize-y"} />
              </div>
            </div>
          )}

          <div className="w-32">
            <label className={labelCls}>超时（秒）</label>
            <input type="number" min={1} value={hook.timeout_secs || 10} onChange={(e) => updateHook(idx, { timeout_secs: parseInt(e.target.value) || 10 })} className={inputCls} />
          </div>
        </div>
      ))}

      {hooks.length < maxHooks && (
        <button type="button" onClick={addHook} className="flex items-center gap-1.5 text-xs font-medium text-gray-600 hover:text-gray-900 cursor-pointer">
          <Plus className="h-3.5 w-3.5" /> 添加{mode === 'pre' ? '前置' : '后置'} Hook
        </button>
      )}
    </div>
  )
}

export type { PreHook, PostHook }
