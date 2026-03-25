import { clearSession, authState } from '../store/auth'

const API_BASE = '/api/v1'

async function request(path, options = {}) {
  const headers = { ...(options.headers || {}) }
  const token = authState.token
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const config = {
    method: options.method || 'GET',
    headers,
  }

  if (options.body !== undefined) {
    if (options.isFormData) {
      config.body = options.body
    } else {
      headers['Content-Type'] = 'application/json'
      config.body = JSON.stringify(options.body)
    }
  }

  const res = await fetch(`${API_BASE}${path}`, config)

  if (res.status === 401) {
    clearSession()
    if (window.location.pathname !== '/login') {
      window.location.href = '/login'
    }
    throw new Error('Session expired. Please login again.')
  }

  const isCSV = options.expectCSV
  if (isCSV) {
    if (!res.ok) {
      throw new Error('Failed to export CSV')
    }
    return await res.blob()
  }

  let payload = null
  try {
    payload = await res.json()
  } catch {
    payload = null
  }

  if (!res.ok || !payload?.success) {
    const message = payload?.error?.message || `Request failed (${res.status})`
    throw new Error(message)
  }

  return payload.data
}

export const api = {
  login(body) {
    return request('/auth/login', { method: 'POST', body })
  },
  logout() {
    return request('/auth/logout', { method: 'POST' })
  },
  me() {
    return request('/auth/me')
  },
  dashboard() {
    return request('/dashboard/summary')
  },

  listPositions() {
    return request('/recruitment/positions')
  },
  createPosition(body) {
    return request('/recruitment/positions', { method: 'POST', body })
  },
  listCandidates() {
    return request('/recruitment/candidates')
  },
  createCandidate(body) {
    return request('/recruitment/candidates', { method: 'POST', body })
  },
  updateCandidate(id, body) {
    return request(`/recruitment/candidates/${id}`, { method: 'PUT', body })
  },
  importCandidates(file) {
    const form = new FormData()
    form.append('file', file)
    return request('/recruitment/candidates/import', {
      method: 'POST',
      body: form,
      isFormData: true,
    })
  },
  mergeCandidates(body) {
    return request('/recruitment/candidates/merge', { method: 'POST', body })
  },
  searchCandidates(query) {
    return request(`/recruitment/candidates/search?q=${encodeURIComponent(query)}`)
  },

  listQualifications() {
    return request('/compliance/qualifications')
  },
  createQualification(body) {
    return request('/compliance/qualifications', { method: 'POST', body })
  },
  updateQualification(id, body) {
    return request(`/compliance/qualifications/${id}`, { method: 'PUT', body })
  },
  deleteQualification(id) {
    return request(`/compliance/qualifications/${id}`, { method: 'DELETE' })
  },
  listRestrictions() {
    return request('/compliance/restrictions')
  },
  createRestriction(body) {
    return request('/compliance/restrictions', { method: 'POST', body })
  },
  updateRestriction(id, body) {
    return request(`/compliance/restrictions/${id}`, { method: 'PUT', body })
  },
  deleteRestriction(id) {
    return request(`/compliance/restrictions/${id}`, { method: 'DELETE' })
  },
  checkRestriction(body) {
    return request('/compliance/restrictions/check', { method: 'POST', body })
  },

  listCases(params = {}) {
    const usp = new URLSearchParams()
    if (params.status) usp.set('status', params.status)
    if (params.q) usp.set('q', params.q)
    const query = usp.toString() ? `?${usp.toString()}` : ''
    return request(`/cases${query}`)
  },
  createCase(body) {
    return request('/cases', { method: 'POST', body })
  },
  assignCase(id, assignedTo) {
    return request(`/cases/${id}/assign`, { method: 'PUT', body: { assigned_to: assignedTo } })
  },
  updateCaseStatus(id, status) {
    return request(`/cases/${id}/status`, { method: 'PUT', body: { status } })
  },
  listCaseAttachments(caseId) {
    return request(`/cases/${caseId}/attachments`)
  },

  uploadInit(body) {
    return request('/files/initiate', { method: 'POST', body })
  },
  uploadChunk(uploadId, chunkIndex, blob, name = 'chunk.bin') {
    const form = new FormData()
    form.append('upload_id', uploadId)
    form.append('chunk_index', String(chunkIndex))
    form.append('chunk', blob, name)
    return request('/files/chunk', { method: 'POST', body: form, isFormData: true })
  },
  uploadComplete(uploadId) {
    return request('/files/complete', { method: 'POST', body: { upload_id: uploadId } })
  },
  uploadSession(uploadId) {
    return request(`/files/sessions/${uploadId}`)
  },
  downloadAttachmentUrl(attachmentId) {
    return `${API_BASE}/files/${attachmentId}/download`
  },

  listAuditLogs(params = {}) {
    const usp = new URLSearchParams()
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        usp.set(key, value)
      }
    })
    const query = usp.toString() ? `?${usp.toString()}` : ''
    return request(`/audit/logs${query}`)
  },
  exportAuditCSV(params = {}) {
    const usp = new URLSearchParams()
    Object.entries(params).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        usp.set(key, value)
      }
    })
    const query = usp.toString() ? `?${usp.toString()}` : ''
    return request(`/audit/logs/export${query}`, { expectCSV: true })
  },
}
