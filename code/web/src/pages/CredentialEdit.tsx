import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Plus, Trash2, Eye, EyeOff, Play } from 'lucide-react'
import { credentialApi } from '../api/credential'
import Accordion from '../components/Accordion'
import FormField from '../components/FormField'

type KV = { key: string; value: string }

export default function CredentialEdit() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const [name, setName] = useState('')
  const [code, setCode] = useState('')
  const [type, setType] = useState('')
  const [timeoutSecs, setTimeoutSecs] = useState(30)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [fetching, setFetching] = useState(true)
  const [showSecret, setShowSecret] = useState(false)
  const [testing, setTesting] = useState(false)
  const [testResult, setTestResult] = useState<{ injections: Array<{ location: string; key: string; value: string }> } | null>(null)
  const [testError, setTestError] = useState('')

  const [token, setToken] = useState('')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [tokenUrl, setTokenUrl] = useState('')
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [scopes, setScopes] = useState('')
  const [dynUrl, setDynUrl] = useState('')
  const [dynMethod, setDynMethod] = useState('POST')
  const [dynHeaders, setDynHeaders] = useState<KV[]>([])
  const [dynBody, setDynBody] = useState('')
  const [extractPath, setExtractPath] = useState('')
  const [extractTtl, setExtractTtl] = useState(3600)
  const [hmacSecret, setHmacSecret] = useState('')
  const [hmacAlgorithm, setHmacAlgorithm] = useState('sha256')
  const [hmacHeaderName, setHmacHeaderName] = useState('')
  const [hmacSignFields, setHmacSignFields] = useState('')
  const [customHeaders, setCustomHeaders] = useState<KV[]>([{ key: '', value: '' }])

  // inject settings
  const [injectLocation, setInjectLocation] = useState('header')
  const [injectKey, setInjectKey] = useState('Authorization')
  const [injectPrefix, setInjectPrefix] = useState('Bearer ')

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
        (document.getElementById('cred-form') as HTMLFormElement)?.requestSubmit()
      }
    }
    window.addEventListener('keydown', handler)
    return () => window.removeEventListener('keydown', handler)
  }, [])

  const injectDefaults: Record<string, { location: string; key: string; prefix: string }> = {
    bearer: { location: 'header', key: 'Authorization', prefix: 'Bearer ' },
    basic: { location: 'header', key: 'Authorization', prefix: 'Basic ' },
    oauth2_cc: { location: 'header', key: 'Authorization', prefix: 'Bearer ' },
    dynamic: { location: 'header', key: 'Authorization', prefix: 'Bearer ' },
    hmac: { location: 'header', key: 'X-Signature', prefix: '' },
    custom_header: { location: 'header', key: '', prefix: '' },
  }

  useEffect(() => {
    if (!id) return
    credentialApi.getDecrypted(id).then((res) => {
      const d = res.data
      setName(d.name)
      setCode((d as unknown as { code: string }).code || '')
      setType(d.type)
      setTimeoutSecs(d.timeout_secs)
      fillFromConfig(d.type, d.config as Record<string, unknown>)
      setFetching(false)
    }).catch(() => {
      setError('加载凭证失败')
      setFetching(false)
    })
  }, [id])

  const fillFromConfig = (credType: string, config: Record<string, unknown>) => {
    const def = injectDefaults[credType] || injectDefaults.bearer
    setInjectLocation((config.inject_location as string) || def.location)
    setInjectKey((config.inject_key as string) || def.key)
    setInjectPrefix(config.inject_prefix !== undefined ? (config.inject_prefix as string) : def.prefix)

    switch (credType) {
      case 'bearer':
        setToken((config.token as string) || '')
        break
      case 'basic':
        setUsername((config.username as string) || '')
        setPassword((config.password as string) || '')
        break
      case 'oauth2_cc':
        setTokenUrl((config.token_url as string) || '')
        setClientId((config.client_id as string) || '')
        setClientSecret((config.client_secret as string) || '')
        setScopes(Array.isArray(config.scopes) ? config.scopes.join(', ') : '')
        break
      case 'dynamic': {
        const tr = config.token_request as Record<string, unknown> | undefined
        if (tr) {
          setDynUrl((tr.url as string) || '')
          setDynMethod((tr.method as string) || 'POST')
          const hdrs = tr.headers as Record<string, string> | undefined
          if (hdrs) setDynHeaders(Object.entries(hdrs).map(([key, value]) => ({ key, value })))
          setDynBody(tr.body ? (typeof tr.body === 'string' ? tr.body : JSON.stringify(tr.body, null, 2)) : '')
        }
        const te = config.token_extract as Record<string, unknown> | undefined
        if (te) {
          setExtractPath((te.path as string) || '')
          setExtractTtl((te.ttl as number) || 3600)
        }
        break
      }
      case 'hmac':
        setHmacSecret((config.secret as string) || '')
        setHmacAlgorithm((config.algorithm as string) || 'sha256')
        setHmacHeaderName((config.header_name as string) || '')
        setHmacSignFields(Array.isArray(config.sign_fields) ? config.sign_fields.join(', ') : '')
        break
      case 'custom_header': {
        const hdrs = config.headers as Record<string, string> | undefined
        if (hdrs) setCustomHeaders(Object.entries(hdrs).map(([key, value]) => ({ key, value })))
        break
      }
    }
  }

  const buildConfig = (): Record<string, unknown> => {
    const injectFields: Record<string, string> = {}
    const def = injectDefaults[type] || injectDefaults.bearer
    if (injectLocation && injectLocation !== def.location) injectFields.inject_location = injectLocation
    if (injectKey && injectKey !== def.key) injectFields.inject_key = injectKey
    if (injectPrefix !== undefined && injectPrefix !== def.prefix) injectFields.inject_prefix = injectPrefix

    let cfg: Record<string, unknown>
    switch (type) {
      case 'bearer': cfg = { token }; break
      case 'basic': cfg = { username, password }; break
      case 'oauth2_cc': cfg = { token_url: tokenUrl, client_id: clientId, client_secret: clientSecret, scopes: scopes.split(',').map(s => s.trim()).filter(Boolean) }; break
      case 'dynamic': {
        const hdrs: Record<string, string> = {}
        dynHeaders.forEach(({ key, value }) => { if (key.trim()) hdrs[key.trim()] = value })
        let parsedBody: unknown = undefined
        if (dynBody.trim()) {
          try { parsedBody = JSON.parse(dynBody) } catch { parsedBody = dynBody }
        }
        cfg = {
          token_request: { url: dynUrl, method: dynMethod, headers: hdrs, body: parsedBody },
          token_extract: { path: extractPath, ttl: extractTtl },
        }
        break
      }
      case 'hmac': cfg = { secret: hmacSecret, algorithm: hmacAlgorithm, header_name: hmacHeaderName, sign_fields: hmacSignFields.split(',').map(s => s.trim()).filter(Boolean) }; break
      case 'custom_header': {
        const pairs: Record<string, string> = {}
        customHeaders.forEach(({ key, value }) => { if (key.trim()) pairs[key.trim()] = value })
        cfg = { headers: pairs }
        break
      }
      default: cfg = {}
    }
    return { ...cfg, ...injectFields }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError('')
    try {
      await credentialApi.update(id!, { name, code, config: buildConfig(), timeout_secs: timeoutSecs })
      navigate('/credentials')
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '更新凭证失败')
    } finally {
      setLoading(false)
    }
  }

  const handleTest = async () => {
    setTesting(true)
    setTestResult(null)
    setTestError('')
    try {
      const res = await credentialApi.test(id!)
      setTestResult(res.data)
    } catch (err: unknown) {
      setTestError(err instanceof Error ? err.message : '凭证测试失败')
    } finally {
      setTesting(false)
    }
  }

  const inputCls = "w-full rounded-md border border-stone-200 bg-white px-3 py-2 text-sm text-gray-900 outline-none transition-all placeholder:text-gray-400 focus:border-stone-400 focus:ring-1 focus:ring-stone-200"

  if (fetching) {
    return (
      <div className="flex h-32 items-center justify-center">
        <div className="h-5 w-5 animate-spin rounded-full border-2 border-stone-300 border-t-transparent" />
      </div>
    )
  }

  return (
    <div className="pb-8">
      <div className="mb-5 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={() => navigate('/credentials')} className="flex items-center justify-center w-8 h-8 rounded-lg border border-stone-200 text-gray-400 hover:text-gray-900 hover:border-stone-300 transition-colors cursor-pointer">
            <ArrowLeft className="h-4 w-4" />
          </button>
          <div>
            <h2 className="text-xl font-semibold text-gray-900">编辑凭证</h2>
            <p className="text-xs text-gray-400">修改凭证配置</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={handleTest} disabled={testing} className="flex items-center gap-1.5 bg-white border border-stone-200 text-gray-600 rounded-md px-4 py-2 text-sm hover:bg-stone-50 transition-colors duration-150 disabled:opacity-50 cursor-pointer">
            <Play className="h-3.5 w-3.5" />
            {testing ? '测试中...' : '测试凭证'}
          </button>
          <button type="button" onClick={() => navigate('/credentials')} className="bg-white border border-stone-200 text-gray-600 rounded-md px-4 py-2 text-sm hover:bg-stone-50 transition-colors duration-150 cursor-pointer">
            取消
          </button>
          <button type="submit" form="cred-form" disabled={loading} className="bg-black text-white rounded-md px-4 py-2 text-sm font-medium hover:bg-gray-800 transition-colors duration-150 disabled:opacity-50 cursor-pointer">
            {loading ? '保存中...' : '保存'}
          </button>
        </div>
      </div>

      {error && <div className="mb-4 rounded-lg border border-red-100 bg-red-50 px-4 py-2.5 text-sm text-red-600">{error}</div>}
      {testError && <div className="mb-4 rounded-lg border border-red-100 bg-red-50 px-4 py-2.5 text-sm text-red-600">测试失败：{testError}</div>}
      {testResult && (
        <div className="mb-4 rounded-lg border border-green-100 bg-green-50 p-4">
          <h4 className="text-sm font-medium text-green-800 mb-2">凭证解析结果</h4>
          <div className="space-y-1">
            {testResult.injections.map((inj, i) => (
              <div key={i} className="flex items-center gap-2 text-xs font-mono text-green-700">
                <span className="px-1.5 py-0.5 bg-green-100 rounded">{inj.location}</span>
                <span className="font-semibold">{inj.key}:</span>
                <span className="truncate">{inj.value}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <form id="cred-form" onSubmit={handleSubmit}>
        <div className="border border-stone-200 rounded-lg bg-white p-5 mb-4">
          <h3 className="text-sm font-semibold text-gray-900 mb-4">基本信息</h3>
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <FormField label="凭证名称" required>
                <input type="text" required value={name} onChange={(e) => setName(e.target.value)} className={inputCls} />
              </FormField>
              <FormField label="凭证编码" required>
                <input type="text" required value={code} onChange={(e) => setCode(e.target.value)} className={inputCls + " font-mono"} />
              </FormField>
            </div>
            <div className="grid grid-cols-2 gap-3">
              <FormField label="凭证类型">
                <input type="text" disabled value={type} className={inputCls + " !bg-gray-100 !text-gray-500"} />
              </FormField>
              <FormField label="超时（秒）">
                <input type="number" min={1} value={timeoutSecs} onChange={(e) => setTimeoutSecs(parseInt(e.target.value) || 30)} className={inputCls} />
              </FormField>
            </div>
          </div>
        </div>

        <div className="border border-stone-200 rounded-lg bg-white p-5 mb-4">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-gray-900">凭证配置</h3>
            <button type="button" onClick={() => setShowSecret(!showSecret)} className="flex items-center gap-1 text-xs text-gray-500 hover:text-gray-900 cursor-pointer">
              {showSecret ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
              {showSecret ? '隐藏' : '显示'}
            </button>
          </div>
          <div className="space-y-3">
            {type === 'bearer' && (
              <FormField label="Token" required>
                <input type={showSecret ? 'text' : 'password'} required value={token} onChange={(e) => setToken(e.target.value)} className={inputCls} />
              </FormField>
            )}
            {type === 'basic' && (
              <>
                <FormField label="用户名" required>
                  <input type="text" required value={username} onChange={(e) => setUsername(e.target.value)} className={inputCls} />
                </FormField>
                <FormField label="密码" required>
                  <input type={showSecret ? 'text' : 'password'} required value={password} onChange={(e) => setPassword(e.target.value)} className={inputCls} />
                </FormField>
              </>
            )}
            {type === 'oauth2_cc' && (
              <>
                <FormField label="Token URL" required>
                  <input type="url" required value={tokenUrl} onChange={(e) => setTokenUrl(e.target.value)} className={inputCls} />
                </FormField>
                <FormField label="Client ID" required>
                  <input type="text" required value={clientId} onChange={(e) => setClientId(e.target.value)} className={inputCls} />
                </FormField>
                <FormField label="Client Secret" required>
                  <input type={showSecret ? 'text' : 'password'} required value={clientSecret} onChange={(e) => setClientSecret(e.target.value)} className={inputCls} />
                </FormField>
                <FormField label="Scopes（逗号分隔）">
                  <input type="text" value={scopes} onChange={(e) => setScopes(e.target.value)} className={inputCls} />
                </FormField>
              </>
            )}
            {type === 'hmac' && (
              <>
                <FormField label="密钥" required>
                  <input type={showSecret ? 'text' : 'password'} required value={hmacSecret} onChange={(e) => setHmacSecret(e.target.value)} className={inputCls} />
                </FormField>
                <FormField label="算法">
                  <select value={hmacAlgorithm} onChange={(e) => setHmacAlgorithm(e.target.value)} className={inputCls}>
                    <option value="sha256">SHA-256</option>
                    <option value="sha512">SHA-512</option>
                    <option value="sha1">SHA-1</option>
                  </select>
                </FormField>
                <FormField label="Header 名称" required>
                  <input type="text" required value={hmacHeaderName} onChange={(e) => setHmacHeaderName(e.target.value)} className={inputCls} />
                </FormField>
                <FormField label="签名字段（逗号分隔）">
                  <input type="text" value={hmacSignFields} onChange={(e) => setHmacSignFields(e.target.value)} className={inputCls} />
                </FormField>
              </>
            )}
            {type === 'custom_header' && (
              <div>
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium text-gray-600">Header 键值对</span>
                  <button type="button" onClick={() => setCustomHeaders([...customHeaders, { key: '', value: '' }])} className="flex items-center gap-1 text-xs font-medium text-gray-700 hover:text-gray-900 cursor-pointer">
                    <Plus className="h-3 w-3" /> 添加
                  </button>
                </div>
                <div className="space-y-2">
                  {customHeaders.map((h, i) => (
                    <div key={i} className="flex gap-2">
                      <input type="text" value={h.key} onChange={(e) => { const n = [...customHeaders]; n[i].key = e.target.value; setCustomHeaders(n) }} placeholder="Key" className={inputCls} />
                      <input type={showSecret ? 'text' : 'password'} value={h.value} onChange={(e) => { const n = [...customHeaders]; n[i].value = e.target.value; setCustomHeaders(n) }} placeholder="Value" className={inputCls} />
                      <button type="button" onClick={() => setCustomHeaders(customHeaders.filter((_, j) => j !== i))} className="flex-shrink-0 p-2 text-gray-300 hover:text-red-500 transition-colors cursor-pointer">
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>

        {type === 'dynamic' && (
          <Accordion title="动态凭证配置" defaultOpen>
            <div className="space-y-3">
              <div className="grid grid-cols-1 lg:grid-cols-[7.5rem_1fr] gap-3">
                <FormField label="请求方法">
                  <select value={dynMethod} onChange={(e) => setDynMethod(e.target.value)} className={inputCls}>
                    {['GET', 'POST', 'PUT'].map((m) => <option key={m}>{m}</option>)}
                  </select>
                </FormField>
                <FormField label="Token 请求 URL" required>
                  <input type="url" required value={dynUrl} onChange={(e) => setDynUrl(e.target.value)} placeholder="https://auth.example.com/token" className={inputCls} />
                </FormField>
              </div>

              <Accordion title="请求 Headers">
                <div className="space-y-2">
                  <div className="flex justify-end mb-2">
                    <button type="button" onClick={() => setDynHeaders([...dynHeaders, { key: '', value: '' }])} className="flex items-center gap-1 text-xs font-medium text-gray-700 hover:text-gray-900 cursor-pointer">
                      <Plus className="h-3 w-3" /> 添加
                    </button>
                  </div>
                  {dynHeaders.map((h, i) => (
                    <div key={i} className="flex gap-2">
                      <input type="text" value={h.key} onChange={(e) => { const n = [...dynHeaders]; n[i].key = e.target.value; setDynHeaders(n) }} placeholder="Key" className={inputCls} />
                      <input type="text" value={h.value} onChange={(e) => { const n = [...dynHeaders]; n[i].value = e.target.value; setDynHeaders(n) }} placeholder="Value" className={inputCls} />
                      <button type="button" onClick={() => setDynHeaders(dynHeaders.filter((_, j) => j !== i))} className="flex-shrink-0 p-2 text-gray-300 hover:text-red-500 cursor-pointer">
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </div>
                  ))}
                </div>
              </Accordion>

              <Accordion title="请求体">
                <FormField label="请求体（JSON）">
                  <textarea value={dynBody} onChange={(e) => setDynBody(e.target.value)} rows={4} placeholder='{"grant_type": "client_credentials"}' className={inputCls + " font-mono resize-y"} />
                </FormField>
              </Accordion>

              <div className="grid grid-cols-2 gap-3">
                <FormField label="Token 提取路径" required>
                  <input type="text" required value={extractPath} onChange={(e) => setExtractPath(e.target.value)} placeholder="data.access_token" className={inputCls + " font-mono"} />
                </FormField>
                <FormField label="Token TTL（秒）">
                  <input type="number" min={1} value={extractTtl} onChange={(e) => setExtractTtl(parseInt(e.target.value) || 3600)} className={inputCls} />
                </FormField>
              </div>
            </div>
          </Accordion>
        )}

        {type !== 'custom_header' && (
          <Accordion title="注入设置（高级）">
            <div className="space-y-3">
              <div className="grid grid-cols-3 gap-3">
                <FormField label="注入位置">
                  <select value={injectLocation} onChange={(e) => setInjectLocation(e.target.value)} className={inputCls}>
                    <option value="header">Header</option>
                    <option value="query">Query 参数</option>
                    <option value="cookie">Cookie</option>
                  </select>
                </FormField>
                <FormField label="Key">
                  <input type="text" value={injectKey} onChange={(e) => setInjectKey(e.target.value)} placeholder="Authorization" className={inputCls} />
                </FormField>
                <FormField label="前缀">
                  <input type="text" value={injectPrefix} onChange={(e) => setInjectPrefix(e.target.value)} placeholder="Bearer " className={inputCls} />
                </FormField>
              </div>
              <p className="text-xs text-gray-400">默认按凭证类型自动填充，仅在需要自定义时修改</p>
            </div>
          </Accordion>
        )}
      </form>
    </div>
  )
}
