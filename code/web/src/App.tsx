import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { hasToken } from './api/client'
import { getUserInfo } from './api/client'
import { AuthProvider, useAuth } from './context/AuthContext'
import Layout from './components/Layout'
import Login from './pages/Login'
import Register from './pages/Register'
import Onboarding from './pages/Onboarding'
import TenantSelect from './pages/TenantSelect'
import Members from './pages/Members'
import TaskList from './pages/TaskList'
import TaskDetail from './pages/TaskDetail'
import TaskCreate from './pages/TaskCreate'
import TaskEdit from './pages/TaskEdit'
import ApiKeys from './pages/ApiKeys'
import Quota from './pages/Quota'
import Variables from './pages/Variables'
import Credentials from './pages/Credentials'
import VariableCreate from './pages/VariableCreate'
import CredentialCreate from './pages/CredentialCreate'
import CredentialEdit from './pages/CredentialEdit'

function AppRoutes() {
  const { user, currentTenant } = useAuth()
  const hasUser = !!user || !!getUserInfo()
  const hasTenant = !!currentTenant

  if (hasToken() && hasUser && hasTenant) {
    return (
      <Layout>
        <Routes>
          <Route path="/" element={<Navigate to="/tasks" replace />} />
          <Route path="/tasks" element={<TaskList />} />
          <Route path="/tasks/new" element={<TaskCreate />} />
          <Route path="/tasks/:id" element={<TaskDetail />} />
          <Route path="/tasks/:id/edit" element={<TaskEdit />} />
          <Route path="/variables" element={<Variables />} />
          <Route path="/variables/new" element={<VariableCreate />} />
          <Route path="/variables/:id/edit" element={<VariableCreate />} />
          <Route path="/credentials" element={<Credentials />} />
          <Route path="/credentials/new" element={<CredentialCreate />} />
          <Route path="/credentials/:id/edit" element={<CredentialEdit />} />
          <Route path="/members" element={<Members />} />
          <Route path="/keys" element={<ApiKeys />} />
          <Route path="/quota" element={<Quota />} />
          <Route path="/select-tenant" element={<TenantSelect />} />
          <Route path="/onboarding" element={<Onboarding />} />
          <Route path="*" element={<Navigate to="/tasks" replace />} />
        </Routes>
      </Layout>
    )
  }

  if (hasUser && !hasTenant) {
    return (
      <Routes>
        <Route path="/onboarding" element={<Onboarding />} />
        <Route path="/select-tenant" element={<TenantSelect />} />
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route path="*" element={<Navigate to={currentTenant ? '/select-tenant' : '/onboarding'} replace />} />
      </Routes>
    )
  }

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/register" element={<Register />} />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  )
}

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <AppRoutes />
      </AuthProvider>
    </BrowserRouter>
  )
}
