import type { CheckResult, Incident, Monitor, MonitorInput, MonitorStats } from './types'

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

function apiBaseUrl(): string {
  const value = import.meta.env.VITE_API_BASE_URL
  return value?.replace(/\/$/, '') || 'http://localhost:8080'
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${apiBaseUrl()}${path}`, {
    ...init,
    headers: {
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...init?.headers,
    },
  })

  if (response.status === 204) {
    return undefined as T
  }

  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    const message =
      payload && typeof payload === 'object' && 'error' in payload && typeof payload.error === 'string'
        ? payload.error
        : `Request failed (${response.status})`
    throw new ApiError(message, response.status)
  }

  return payload as T
}

export const api = {
  listMonitors(): Promise<Monitor[]> {
    return request<Monitor[] | null>('/api/monitors').then((rows) => rows ?? [])
  },

  getMonitor(id: string): Promise<Monitor> {
    return request(`/api/monitors/${id}`)
  },

  createMonitor(input: MonitorInput): Promise<Monitor> {
    return request('/api/monitors', {
      method: 'POST',
      body: JSON.stringify(input),
    })
  },

  updateMonitor(id: string, input: MonitorInput): Promise<Monitor> {
    return request(`/api/monitors/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    })
  },

  deleteMonitor(id: string): Promise<void> {
    return request(`/api/monitors/${id}`, { method: 'DELETE' })
  },

  getMonitorStats(id: string): Promise<MonitorStats> {
    return request(`/api/monitors/${id}/stats`)
  },

  listMonitorChecks(id: string): Promise<CheckResult[]> {
    return request<CheckResult[] | null>(`/api/monitors/${id}/checks`).then((rows) => rows ?? [])
  },

  listMonitorIncidents(id: string): Promise<Incident[]> {
    return request<Incident[] | null>(`/api/monitors/${id}/incidents`).then((rows) => rows ?? [])
  },
}
