import { useState, useEffect } from 'react'
import { credentialApi, type Credential } from '../api/credential'

interface Props {
  value: string
  onChange: (id: string) => void
  tenantId?: string
}

export default function CredentialSelect({ value, onChange }: Props) {
  const [credentials, setCredentials] = useState<Credential[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    credentialApi.list({ status: 'active', size: 100 }).then((res) => {
      const payload = res.data as { items: Credential[]; total: number }
      setCredentials(Array.isArray(payload.items) ? payload.items : [])
      setLoading(false)
    }).catch(() => setLoading(false))
  }, [])

  if (loading) {
    return <select disabled className="w-full rounded-md border border-stone-200 bg-gray-100 px-3 py-2 text-sm text-gray-400"><option>加载中...</option></select>
  }

  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="w-full rounded-md border border-stone-200 bg-gray-50/60 px-3 py-2 text-sm text-gray-900 outline-none transition-all focus:bg-white focus:border-stone-400 focus:ring-1 focus:ring-stone-200"
    >
      <option value="">选择凭证</option>
      {credentials.map((cred) => (
        <option key={cred.id} value={cred.id}>
          [{cred.type}] {cred.name}
        </option>
      ))}
    </select>
  )
}
