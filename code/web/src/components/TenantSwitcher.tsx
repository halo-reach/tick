import { useState, useRef, useEffect } from 'react'
import { Building2, ChevronDown, Pencil } from 'lucide-react'
import { useAuth } from '../context/AuthContext'
import { api, setTenantInfo, setTenantsInfo } from '../api/client'

export default function TenantSwitcher() {
  const { tenants, currentTenant, selectTenant } = useAuth()
  const isOwner = currentTenant?.role === 'owner' || tenants.find(t => t.id === currentTenant?.id)?.role === 'owner'
  const [open, setOpen] = useState(false)
  const [switching, setSwitching] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [editName, setEditName] = useState('')
  const [saving, setSaving] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setEditing(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [])

  useEffect(() => {
    if (editing && inputRef.current) inputRef.current.focus()
  }, [editing])

  const handleSwitch = async (id: string) => {
    if (id === currentTenant?.id) { setOpen(false); return }
    setSwitching(id)
    try {
      await selectTenant(id)
      setOpen(false)
      window.location.href = '/tasks'
    } finally {
      setSwitching(null)
    }
  }

  const handleStartEdit = (e: React.MouseEvent) => {
    e.stopPropagation()
    setEditName(currentTenant?.name || '')
    setEditing(true)
  }

  const handleSaveEdit = async () => {
    if (!editName.trim() || editName.trim() === currentTenant?.name) {
      setEditing(false)
      return
    }
    setSaving(true)
    try {
      await api.renameTenant(editName.trim())
      const updated = { ...currentTenant!, name: editName.trim() }
      setTenantInfo(updated)
      const updatedList = tenants.map(t => t.id === updated.id ? { ...t, name: editName.trim() } : t)
      setTenantsInfo(updatedList)
      window.location.reload()
    } catch {
      // ignore
    } finally {
      setSaving(false)
      setEditing(false)
    }
  }

  if (!currentTenant) return null

  if (tenants.length <= 1) {
    return (
      <div ref={ref} className="relative flex items-center gap-2 text-sm text-gray-600">
        <Building2 className="h-4 w-4" />
        <span>{currentTenant.name}</span>
        {isOwner && (
          <button onClick={handleStartEdit} className="rounded p-0.5 text-gray-400 hover:text-gray-600 hover:bg-stone-100 cursor-pointer">
            <Pencil className="h-3 w-3" />
          </button>
        )}
        {editing && (
          <div className="absolute right-0 top-full z-50 mt-1 w-52 rounded-lg border border-gray-200 bg-white shadow-lg p-3 space-y-2">
            <label className="block text-xs text-gray-500">工作空间名称</label>
            <input
              ref={inputRef}
              value={editName}
              onChange={e => setEditName(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') handleSaveEdit(); if (e.key === 'Escape') setEditing(false); }}
              className="w-full rounded border border-stone-200 px-2.5 py-1.5 text-sm outline-none focus:border-stone-400"
            />
            <div className="flex gap-2">
              <button type="button" onClick={() => setEditing(false)} className="flex-1 rounded border border-stone-200 px-2 py-1.5 text-sm text-gray-600 hover:bg-stone-50 cursor-pointer">
                取消
              </button>
              <button onClick={handleSaveEdit} disabled={saving} className="flex-1 rounded bg-gray-900 px-2 py-1.5 text-sm text-white hover:bg-gray-800 disabled:opacity-50 cursor-pointer">
                {saving ? '...' : '确认'}
              </button>
            </div>
          </div>
        )}
      </div>
    )
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm text-gray-600 hover:bg-stone-100 transition-colors cursor-pointer"
      >
        <Building2 className="h-4 w-4" />
        <span>{currentTenant.name}</span>
        <ChevronDown className="h-3.5 w-3.5" />
      </button>
      {open && (
        <div className="absolute right-0 top-full z-50 mt-1 w-52 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
          {tenants.map((t) => (
            <div key={t.id} className={`flex items-center px-3 py-2 text-sm transition-colors ${
              t.id === currentTenant.id ? 'bg-stone-50 text-gray-900 font-medium' : 'text-gray-600 hover:bg-stone-50'
            }`}>
              <button
                onClick={() => handleSwitch(t.id)}
                disabled={switching !== null}
                className="flex flex-1 items-center gap-2 cursor-pointer min-w-0"
              >
                <span className="flex-1 text-left truncate">{t.name}</span>
                <span className="text-xs text-gray-400 shrink-0">{t.role === 'owner' ? '管理员' : '成员'}</span>
                {switching === t.id && <span className="text-xs text-gray-400">...</span>}
              </button>
              {t.id === currentTenant.id && isOwner && (
                <button onClick={handleStartEdit} className="ml-1 rounded p-1 text-gray-400 hover:text-gray-600 hover:bg-stone-100 cursor-pointer">
                  <Pencil className="h-3 w-3" />
                </button>
              )}
            </div>
          ))}
          {editing && (
            <form onSubmit={e => { e.preventDefault(); handleSaveEdit() }} className="border-t border-gray-100 p-3 space-y-2">
              <label className="block text-xs text-gray-500">工作空间名称</label>
              <input
                ref={inputRef}
                value={editName}
                onChange={e => setEditName(e.target.value)}
                className="w-full rounded border border-stone-200 px-2.5 py-1.5 text-sm outline-none focus:border-stone-400"
              />
              <div className="flex gap-2">
                <button type="button" onClick={() => setEditing(false)} className="flex-1 rounded border border-stone-200 px-2 py-1.5 text-sm text-gray-600 hover:bg-stone-50 cursor-pointer">
                  取消
                </button>
                <button type="submit" disabled={saving} className="flex-1 rounded bg-gray-900 px-2 py-1.5 text-sm text-white hover:bg-gray-800 disabled:opacity-50 cursor-pointer">
                  {saving ? '...' : '确认'}
                </button>
              </div>
            </form>
          )}
        </div>
      )}
    </div>
  )
}
