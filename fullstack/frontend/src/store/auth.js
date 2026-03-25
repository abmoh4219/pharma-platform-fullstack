import { reactive } from 'vue'

const STORAGE_KEY = 'pharma_auth_session'

function loadSession() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return { token: '', user: null }
    }
    const parsed = JSON.parse(raw)
    return {
      token: parsed.token || '',
      user: parsed.user || null,
    }
  } catch {
    return { token: '', user: null }
  }
}

const initial = loadSession()

export const authState = reactive({
  token: initial.token,
  user: initial.user,
})

export function setSession(token, user) {
  authState.token = token
  authState.user = user
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ token, user }))
}

export function clearSession() {
  authState.token = ''
  authState.user = null
  localStorage.removeItem(STORAGE_KEY)
}

export function isAuthenticated() {
  return Boolean(authState.token)
}

export function hasAnyRole(roles = []) {
  if (!roles.length) return true
  const role = authState.user?.role
  return roles.includes(role)
}

export function roleMenus(role) {
  const dashboard = { key: '/dashboard', label: 'Dashboard' }
  const recruitment = { key: '/recruitment', label: 'Recruitment' }
  const compliance = { key: '/compliance', label: 'Compliance' }
  const cases = { key: '/cases', label: 'Cases' }
  const audit = { key: '/audit', label: 'Audit Logs' }

  if (role === 'business_specialist') {
    return [dashboard, cases]
  }
  if (role === 'recruitment_specialist') {
    return [dashboard, recruitment, cases]
  }
  if (role === 'compliance_admin') {
    return [dashboard, compliance, cases, audit]
  }
  if (role === 'system_admin') {
    return [dashboard, recruitment, compliance, cases, audit]
  }
  return [dashboard, cases]
}
