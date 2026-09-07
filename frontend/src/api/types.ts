export type MonitorStatus = 'up' | 'down' | 'unknown'
export type IncidentStatus = 'open' | 'resolved'
export type HTTPMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE' | 'HEAD' | 'OPTIONS'

export type Monitor = {
  id: string
  user_id: string | null
  name: string
  url: string
  http_method: string
  check_interval_seconds: number
  timeout_seconds: number
  expected_status_code: number
  is_active: boolean
  next_check_at: string | null
  created_at: string
  updated_at: string
}

export type MonitorInput = {
  name: string
  url: string
  http_method: string
  check_interval_seconds: number
  timeout_seconds: number
  expected_status_code: number
}

export type MonitorStats = {
  current_status: MonitorStatus
  uptime_percentage: number
  average_latency_ms: number
  p50_latency_ms: number
  p95_latency_ms: number
  p99_latency_ms: number
  total_checks: number
  failed_checks: number
  incident_count: number
  active_incident: boolean
}

export type CheckResult = {
  id: string
  job_id?: string
  monitor_id: string
  status_code: number | null
  response_time_ms: number
  success: boolean
  error_message: string | null
  checked_at: string
}

export type Incident = {
  id: string
  monitor_id: string
  started_at: string
  resolved_at: string | null
  status: IncidentStatus
  failure_count: number
  created_at: string
  updated_at: string
}

export type MonitorCardData = {
  monitor: Monitor
  stats: MonitorStats
  lastCheckedAt: string | null
}
