import { useState, useEffect, useCallback, useRef } from 'react'
import { Copy, Trash2, UserMinus, ArrowUpDown, UserPlus, Search, X } from 'lucide-react'
import { api, MemberInfo, InvitationInfo } from '../api/client'
import { useAuth } from '../context/AuthContext'
import ConfirmDialog from '../components/ConfirmDialog'

interface SearchUser {
  id: string
  username: string
  display_name: string
}

function AddMemberModal({ onClose, onAdded }: { onClose: () => void; onAdded: () => void }) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchUser[]>([])
  const [searching, setSearching] = useState(false)
  const [selectedRole, setSelectedRole] = useState<'member' | 'owner'>('member')
  const [adding, setAdding] = useState('')
  const [error, setError] = useState('')
  const timerRef = useRef<ReturnType<typeof setTimeout>>()

  const doSearch = useCallback(async (q: string) => {
    if (q.length < 1) { setResults([]); return }
    setSearching(true)
    try {
      const res = await api.searchUsers(q)
      setResults(res.data.users || [])
    } catch { setResults([]) }
    finally { setSearching(false) }
  }, [])

  const handleInput = (val: string) => {
    setQuery(val)
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = setTimeout(() => doSearch(val), 300)
  }

  const handleAdd = async (u: SearchUser) => {
    setAdding(u.id)
    setError('')
    try {
      await api.addMember(u.id, selectedRole)
      onAdded()
      onClose()
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '添加失败')
    } finally { setAdding('') }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30" onClick={onClose}>
      <div className="w-full max-w-md rounded-xl bg-white p-5 shadow-xl" onClick={e => e.stopPropagation()}>
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-900">添加成员</h3>
          <button onClick={onClose} className="rounded p-1 text-gray-400 hover:text-gray-600 cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="mb-3 flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2">
          <Search className="h-4 w-4 text-gray-400" />
          <input
            autoFocus
            value={query}
            onChange={e => handleInput(e.target.value)}
            placeholder="输入用户名搜索..."
            className="flex-1 text-sm outline-none"
          />
        </div>

        <div className="mb-3 flex items-center gap-3">
          <span className="text-xs text-gray-500">角色：</span>
          <label className="flex items-center gap-1 text-xs cursor-pointer">
            <input type="radio" checked={selectedRole === 'member'} onChange={() => setSelectedRole('member')} className="cursor-pointer" />
            成员
          </label>
          <label className="flex items-center gap-1 text-xs cursor-pointer">
            <input type="radio" checked={selectedRole === 'owner'} onChange={() => setSelectedRole('owner')} className="cursor-pointer" />
            管理员
          </label>
        </div>

        {error && <p className="mb-2 text-xs text-red-500">{error}</p>}

        <div className="max-h-48 overflow-y-auto">
          {searching && <p className="py-2 text-center text-xs text-gray-400">搜索中...</p>}
          {!searching && query && results.length === 0 && (
            <p className="py-2 text-center text-xs text-gray-400">未找到用户</p>
          )}
          {results.map(u => (
            <div key={u.id} className="flex items-center justify-between rounded-lg px-3 py-2 hover:bg-stone-50">
              <div>
                <span className="text-sm text-gray-900">{u.username}</span>
                {u.display_name && <span className="ml-2 text-xs text-gray-400">{u.display_name}</span>}
              </div>
              <button
                onClick={() => handleAdd(u)}
                disabled={adding === u.id}
                className="rounded-md bg-gray-900 px-2.5 py-1 text-xs font-medium text-white hover:bg-gray-800 disabled:opacity-50 cursor-pointer"
              >
                {adding === u.id ? '添加中...' : '添加'}
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

export default function Members() {
  const { role, user } = useAuth()
  const [members, setMembers] = useState<MemberInfo[]>([])
  const [invitations, setInvitations] = useState<InvitationInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [inviteLoading, setInviteLoading] = useState(false)
  const [newCode, setNewCode] = useState('')
  const [showAddModal, setShowAddModal] = useState(false)
  const [confirmRemove, setConfirmRemove] = useState<{ userId: string; username: string } | null>(null)

  const isOwner = role === 'owner'

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const [mRes, iRes] = await Promise.all([
        api.listMembers(),
        isOwner ? api.listInvitations() : Promise.resolve(null),
      ])
      setMembers(mRes.data.members || [])
      if (iRes) setInvitations(iRes.data.invitations || [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '加载失败')
    } finally {
      setLoading(false)
    }
  }, [isOwner])

  useEffect(() => { loadData() }, [loadData])

  const handleInvite = async () => {
    setInviteLoading(true)
    try {
      const res = await api.createInvite()
      setNewCode(res.data.code)
      const iRes = await api.listInvitations()
      setInvitations(iRes.data.invitations || [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '生成邀请码失败')
    } finally {
      setInviteLoading(false)
    }
  }

  const handleRemove = async (userId: string) => {
    try {
      await api.removeMember(userId)
      setMembers(prev => prev.filter(m => m.user_id !== userId))
      setConfirmRemove(null)
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '移除失败')
    }
  }

  const handleChangeRole = async (userId: string, currentRole: string) => {
    const newRole = currentRole === 'owner' ? 'member' : 'owner'
    try {
      await api.changeRole(userId, newRole)
      setMembers(prev => prev.map(m => m.user_id === userId ? { ...m, role: newRole as 'owner' | 'member' } : m))
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '变更角色失败')
    }
  }

  const handleRevokeInvite = async (id: string) => {
    try {
      await api.revokeInvitation(id)
      setInvitations(prev => prev.filter(i => i.id !== id))
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : '撤销失败')
    }
  }

  if (loading) return <p className="text-sm text-gray-500">加载中...</p>

  return (
    <div className="space-y-6">
      {error && <p className="text-xs text-red-500">{error}</p>}

      {/* Member list */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold text-gray-900">成员列表</h2>
          {isOwner && (
            <button
              onClick={() => setShowAddModal(true)}
              className="flex items-center gap-1 rounded-md bg-gray-900 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-gray-800 cursor-pointer"
            >
              <UserPlus className="h-3.5 w-3.5" />
              添加成员
            </button>
          )}
        </div>
        <div className="overflow-hidden rounded-lg border border-gray-200">
          <table className="w-full text-sm">
            <thead className="bg-stone-50 text-left text-xs text-gray-500">
              <tr>
                <th className="px-4 py-2.5">用户名</th>
                <th className="px-4 py-2.5">显示名称</th>
                <th className="px-4 py-2.5">角色</th>
                <th className="px-4 py-2.5">加入时间</th>
                {isOwner && <th className="px-4 py-2.5">操作</th>}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {members.map((m) => (
                <tr key={m.user_id}>
                  <td className="px-4 py-2.5 text-gray-900">{m.username}</td>
                  <td className="px-4 py-2.5 text-gray-600">{m.display_name}</td>
                  <td className="px-4 py-2.5">
                    <span className={`inline-block rounded px-1.5 py-0.5 text-xs ${m.role === 'owner' ? 'bg-amber-50 text-amber-700' : 'bg-gray-100 text-gray-600'}`}>
                      {m.role === 'owner' ? '管理员' : '成员'}
                    </span>
                  </td>
                  <td className="px-4 py-2.5 text-gray-500">{new Date(m.joined_at).toLocaleDateString()}</td>
                  {isOwner && (
                    <td className="px-4 py-2.5">
                      {m.user_id !== user?.user_id && (
                        <div className="flex gap-1">
                          <button
                            onClick={() => handleChangeRole(m.user_id, m.role)}
                            title="切换角色"
                            className="rounded p-1 text-gray-400 hover:bg-stone-100 hover:text-gray-600 cursor-pointer"
                          >
                            <ArrowUpDown className="h-3.5 w-3.5" />
                          </button>
                          <button
                            onClick={() => setConfirmRemove({ userId: m.user_id, username: m.username })}
                            title="移除成员"
                            className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-500 cursor-pointer"
                          >
                            <UserMinus className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      )}
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Invitations (owner only) */}
      {isOwner && (
        <div>
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-gray-900">邀请管理</h2>
            <button
              onClick={handleInvite}
              disabled={inviteLoading}
              className="rounded-md bg-gray-900 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-gray-800 disabled:opacity-50 cursor-pointer"
            >
              {inviteLoading ? '生成中...' : '生成邀请码'}
            </button>
          </div>

          {newCode && (
            <div className="mb-3 flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 px-4 py-2.5">
              <span className="text-sm text-green-800 font-mono">{newCode}</span>
              <button
                onClick={() => { navigator.clipboard.writeText(newCode); }}
                className="rounded p-1 text-green-600 hover:bg-green-100 cursor-pointer"
              >
                <Copy className="h-3.5 w-3.5" />
              </button>
            </div>
          )}

          {invitations.length > 0 && (
            <div className="overflow-hidden rounded-lg border border-gray-200">
              <table className="w-full text-sm">
                <thead className="bg-stone-50 text-left text-xs text-gray-500">
                  <tr>
                    <th className="px-4 py-2.5">邀请码</th>
                    <th className="px-4 py-2.5">角色</th>
                    <th className="px-4 py-2.5">使用次数</th>
                    <th className="px-4 py-2.5">过期时间</th>
                    <th className="px-4 py-2.5">操作</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {invitations.map((inv) => (
                    <tr key={inv.id}>
                      <td className="px-4 py-2.5 font-mono text-gray-900">{inv.code}</td>
                      <td className="px-4 py-2.5 text-gray-600">{inv.role === 'owner' ? '管理员' : '成员'}</td>
                      <td className="px-4 py-2.5 text-gray-600">{inv.used_count}/{inv.max_uses}</td>
                      <td className="px-4 py-2.5 text-gray-500">{new Date(inv.expires_at).toLocaleDateString()}</td>
                      <td className="px-4 py-2.5">
                        <button
                          onClick={() => handleRevokeInvite(inv.id)}
                          title="撤销"
                          className="rounded p-1 text-gray-400 hover:bg-red-50 hover:text-red-500 cursor-pointer"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      {showAddModal && <AddMemberModal onClose={() => setShowAddModal(false)} onAdded={loadData} />}
      {confirmRemove && (
        <ConfirmDialog
          title="移除成员"
          message={`确定要将「${confirmRemove.username}」从租户中移除吗？`}
          confirmText="移除"
          onConfirm={() => handleRemove(confirmRemove.userId)}
          onCancel={() => setConfirmRemove(null)}
          destructive
        />
      )}
    </div>
  )
}
