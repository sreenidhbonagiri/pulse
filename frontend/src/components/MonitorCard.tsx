import { Link } from 'react-router-dom'
import type { MonitorCardData } from '../api/types'
import { formatLatency, formatPercent, formatRelativeTime } from '../lib/format'
import { StatusBadge } from './StatusBadge'

export function MonitorCard({ data }: { data: MonitorCardData }) {
  const { monitor, stats, lastCheckedAt } = data

  return (
    <Link className="monitor-card" to={`/monitors/${monitor.id}`}>
      <div className="monitor-card-top">
        <div>
          <h3>{monitor.name}</h3>
          <p className="muted url">{monitor.url}</p>
        </div>
        <StatusBadge status={stats.current_status} />
      </div>
      <dl className="monitor-card-meta">
        <div>
          <dt>Uptime</dt>
          <dd>{stats.total_checks === 0 ? '—' : formatPercent(stats.uptime_percentage)}</dd>
        </div>
        <div>
          <dt>Avg latency</dt>
          <dd>{stats.total_checks === 0 ? '—' : formatLatency(stats.average_latency_ms)}</dd>
        </div>
        <div>
          <dt>Last checked</dt>
          <dd>{formatRelativeTime(lastCheckedAt)}</dd>
        </div>
      </dl>
      {stats.active_incident ? <p className="incident-chip">Active incident</p> : null}
    </Link>
  )
}
