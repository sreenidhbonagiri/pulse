import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts'
import { api } from '../api/client'
import type { CheckResult, Incident, Monitor, MonitorInput, MonitorStats } from '../api/types'
import { EmptyState, ErrorState, LoadingState } from '../components/States'
import { StatusBadge } from '../components/StatusBadge'
import { formatDateTime, formatLatency, formatPercent } from '../lib/format'

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']

export function MonitorDetailPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()
  const [monitor, setMonitor] = useState<Monitor | null>(null)
  const [stats, setStats] = useState<MonitorStats | null>(null)
  const [checks, setChecks] = useState<CheckResult[]>([])
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState<MonitorInput | null>(null)
  const [saving, setSaving] = useState(false)
  const [actionError, setActionError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [nextMonitor, nextStats, nextChecks, nextIncidents] = await Promise.all([
        api.getMonitor(id),
        api.getMonitorStats(id),
        api.listMonitorChecks(id),
        api.listMonitorIncidents(id),
      ])
      setMonitor(nextMonitor)
      setStats(nextStats)
      setChecks(nextChecks)
      setIncidents(nextIncidents)
      setForm({
        name: nextMonitor.name,
        url: nextMonitor.url,
        http_method: nextMonitor.http_method,
        check_interval_seconds: nextMonitor.check_interval_seconds,
        timeout_seconds: nextMonitor.timeout_seconds,
        expected_status_code: nextMonitor.expected_status_code,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load monitor')
    } finally {
      setLoading(false)
    }
  }, [id])

  useEffect(() => {
    void load()
  }, [load])

  const chartData = useMemo(
    () =>
      [...checks]
        .slice()
        .reverse()
        .map((check) => ({
          time: formatDateTime(check.checked_at),
          latency: check.response_time_ms,
        })),
    [checks],
  )

  async function saveEdits() {
    if (!form) {
      return
    }
    setSaving(true)
    setActionError(null)
    try {
      const updated = await api.updateMonitor(id, form)
      setMonitor(updated)
      setEditing(false)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Could not update monitor')
    } finally {
      setSaving(false)
    }
  }

  async function removeMonitor() {
    if (!window.confirm('Delete this monitor and its check history?')) {
      return
    }
    setActionError(null)
    try {
      await api.deleteMonitor(id)
      navigate('/dashboard')
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Could not delete monitor')
    }
  }

  if (loading) {
    return <LoadingState label="Loading monitor…" />
  }

  if (error || !monitor || !stats || !form) {
    return <ErrorState message={error ?? 'Monitor not found'} onRetry={() => void load()} />
  }

  return (
    <section>
      <div className="page-header">
        <div>
          <p className="eyebrow">
            <Link to="/dashboard">Dashboard</Link> / {monitor.http_method}
          </p>
          <h1>{monitor.name}</h1>
          <p className="muted url">{monitor.url}</p>
        </div>
        <div className="header-actions">
          <StatusBadge status={stats.current_status} />
          <button type="button" className="button ghost" onClick={() => setEditing((value) => !value)}>
            {editing ? 'Cancel' : 'Edit'}
          </button>
          <button type="button" className="button danger" onClick={() => void removeMonitor()}>
            Delete
          </button>
        </div>
      </div>

      {stats.active_incident ? (
        <p className="incident-banner">This monitor has an open incident.</p>
      ) : null}
      {actionError ? <p className="form-error">{actionError}</p> : null}

      {editing ? (
        <form
          className="panel form"
          onSubmit={(event) => {
            event.preventDefault()
            void saveEdits()
          }}
        >
          <div className="form-row">
            <label>
              Name
              <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
            </label>
            <label>
              URL
              <input value={form.url} onChange={(event) => setForm({ ...form, url: event.target.value })} />
            </label>
          </div>
          <div className="form-row">
            <label>
              HTTP method
              <select
                value={form.http_method}
                onChange={(event) => setForm({ ...form, http_method: event.target.value })}
              >
                {methods.map((method) => (
                  <option key={method} value={method}>
                    {method}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Expected status
              <input
                type="number"
                value={form.expected_status_code}
                onChange={(event) =>
                  setForm({ ...form, expected_status_code: Number(event.target.value) })
                }
              />
            </label>
          </div>
          <div className="form-row">
            <label>
              Interval (seconds)
              <input
                type="number"
                value={form.check_interval_seconds}
                onChange={(event) =>
                  setForm({ ...form, check_interval_seconds: Number(event.target.value) })
                }
              />
            </label>
            <label>
              Timeout (seconds)
              <input
                type="number"
                value={form.timeout_seconds}
                onChange={(event) => setForm({ ...form, timeout_seconds: Number(event.target.value) })}
              />
            </label>
          </div>
          <button className="button" type="submit" disabled={saving}>
            {saving ? 'Saving…' : 'Save changes'}
          </button>
        </form>
      ) : null}

      <div className="stat-grid">
        <article className="stat-card">
          <p className="stat-label">Uptime</p>
          <p className="stat-value">{stats.total_checks === 0 ? '—' : formatPercent(stats.uptime_percentage)}</p>
        </article>
        <article className="stat-card">
          <p className="stat-label">Average latency</p>
          <p className="stat-value">{formatLatency(stats.average_latency_ms)}</p>
        </article>
        <article className="stat-card">
          <p className="stat-label">p50</p>
          <p className="stat-value">{formatLatency(stats.p50_latency_ms)}</p>
        </article>
        <article className="stat-card">
          <p className="stat-label">p95</p>
          <p className="stat-value">{formatLatency(stats.p95_latency_ms)}</p>
        </article>
        <article className="stat-card">
          <p className="stat-label">p99</p>
          <p className="stat-value">{formatLatency(stats.p99_latency_ms)}</p>
        </article>
        <article className="stat-card">
          <p className="stat-label">Checks</p>
          <p className="stat-value">{stats.total_checks}</p>
          <p className="stat-hint">
            {stats.total_checks === 0 ? 'No history yet' : `${stats.failed_checks} failed`}
          </p>
        </article>
        <article className="stat-card">
          <p className="stat-label">Active incident</p>
          <p className="stat-value">{stats.active_incident ? 'Open' : 'None'}</p>
        </article>
      </div>

      <section className="panel">
        <h2>Latency over time</h2>
        {chartData.length === 0 ? (
          <EmptyState
            title="No checks yet"
            body="Once Pulse starts checking this endpoint, latency will appear here."
          />
        ) : (
          <div className="chart">
            <ResponsiveContainer width="100%" height={280}>
              <LineChart data={chartData}>
                <CartesianGrid stroke="rgba(243, 239, 230, 0.1)" strokeDasharray="4 4" />
                <XAxis dataKey="time" tick={{ fill: '#9aa39e', fontSize: 12 }} />
                <YAxis tick={{ fill: '#9aa39e', fontSize: 12 }} unit="ms" />
                <Tooltip
                  contentStyle={{
                    background: '#1b2421',
                    border: '1px solid rgba(243, 239, 230, 0.1)',
                    borderRadius: 12,
                    color: '#f3efe6',
                  }}
                />
                <Line type="monotone" dataKey="latency" stroke="#5ee0c8" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}
      </section>

      <section className="panel">
        <h2>Recent checks</h2>
        {checks.length === 0 ? (
          <p className="muted">No check history yet.</p>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Time</th>
                  <th>Status</th>
                  <th>Code</th>
                  <th>Latency</th>
                  <th>Error</th>
                </tr>
              </thead>
              <tbody>
                {checks.map((check) => (
                  <tr key={check.id}>
                    <td>{formatDateTime(check.checked_at)}</td>
                    <td>{check.success ? 'Pass' : 'Fail'}</td>
                    <td>{check.status_code ?? '—'}</td>
                    <td>{formatLatency(check.response_time_ms)}</td>
                    <td>{check.error_message ?? '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="panel">
        <h2>Incidents</h2>
        {incidents.length === 0 ? (
          <p className="muted">No incidents recorded.</p>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Status</th>
                  <th>Started</th>
                  <th>Resolved</th>
                  <th>Failures</th>
                </tr>
              </thead>
              <tbody>
                {incidents.map((incident) => (
                  <tr key={incident.id}>
                    <td className={incident.status === 'open' ? 'text-down' : ''}>{incident.status}</td>
                    <td>{formatDateTime(incident.started_at)}</td>
                    <td>{formatDateTime(incident.resolved_at)}</td>
                    <td>{incident.failure_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </section>
  )
}
