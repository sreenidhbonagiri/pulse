import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { MonitorCardData } from '../api/types'
import { MonitorCard } from '../components/MonitorCard'
import { EmptyState, ErrorState, LoadingState } from '../components/States'
import { StatCard } from '../components/StatCard'
import { average, formatPercent } from '../lib/format'

export function DashboardPage() {
  const [cards, setCards] = useState<MonitorCardData[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const monitors = await api.listMonitors()
      const loaded = await Promise.all(
        monitors.map(async (monitor) => {
          const [stats, checks] = await Promise.all([
            api.getMonitorStats(monitor.id),
            api.listMonitorChecks(monitor.id),
          ])
          return {
            monitor,
            stats,
            lastCheckedAt: checks[0]?.checked_at ?? null,
          }
        }),
      )
      setCards(loaded)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load monitors')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const summary = useMemo(() => {
    const up = cards.filter((card) => card.stats.current_status === 'up').length
    const down = cards.filter((card) => card.stats.current_status === 'down').length
    const incidents = cards.filter((card) => card.stats.active_incident).length
    const uptimes = cards
      .filter((card) => card.stats.total_checks > 0)
      .map((card) => card.stats.uptime_percentage)
    return {
      total: cards.length,
      up,
      down,
      incidents,
      avgUptime: uptimes.length === 0 ? null : average(uptimes),
    }
  }, [cards])

  if (loading) {
    return <LoadingState label="Loading monitors…" />
  }

  if (error) {
    return <ErrorState message={error} onRetry={() => void load()} />
  }

  return (
    <section>
      <div className="page-header">
        <div>
          <p className="eyebrow">Overview</p>
          <h1>Dashboard</h1>
        </div>
        <Link className="button" to="/monitors/new">
          Add monitor
        </Link>
      </div>

      <div className="stat-grid">
        <StatCard label="Monitors" value={String(summary.total)} />
        <StatCard label="Currently up" value={String(summary.up)} />
        <StatCard label="Currently down" value={String(summary.down)} />
        <StatCard label="Active incidents" value={String(summary.incidents)} />
        <StatCard
          label="Average uptime"
          value={summary.avgUptime === null ? '—' : formatPercent(summary.avgUptime)}
          hint="Across monitors with check history"
        />
      </div>

      {cards.length === 0 ? (
        <EmptyState
          title="No monitors yet"
          body="Add an endpoint to start recording uptime, latency, and incidents."
          action={
            <Link className="button" to="/monitors/new">
              Create your first monitor
            </Link>
          }
        />
      ) : (
        <div className="monitor-grid">
          {cards.map((card) => (
            <MonitorCard key={card.monitor.id} data={card} />
          ))}
        </div>
      )}
    </section>
  )
}
