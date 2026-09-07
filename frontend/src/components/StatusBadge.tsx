import type { MonitorStatus } from '../api/types'

const labels: Record<MonitorStatus, string> = {
  up: 'Up',
  down: 'Down',
  unknown: 'Unknown',
}

export function StatusBadge({ status }: { status: MonitorStatus }) {
  return <span className={`badge badge-${status}`}>{labels[status]}</span>
}
