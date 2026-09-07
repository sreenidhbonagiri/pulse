export function formatPercent(value: number): string {
  return `${value.toFixed(1)}%`
}

export function formatLatency(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) {
    return '—'
  }
  if (ms >= 1000) {
    return `${(ms / 1000).toFixed(2)} s`
  }
  return `${Math.round(ms)} ms`
}

export function formatDateTime(value: string | null): string {
  if (!value) {
    return 'Never'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return 'Never'
  }
  return new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(date)
}

export function formatRelativeTime(value: string | null): string {
  if (!value) {
    return 'Not checked yet'
  }
  const date = new Date(value)
  const delta = Date.now() - date.getTime()
  if (Number.isNaN(date.getTime())) {
    return 'Not checked yet'
  }
  const minutes = Math.round(delta / 60000)
  if (Math.abs(minutes) < 1) {
    return 'Just now'
  }
  if (Math.abs(minutes) < 60) {
    return `${minutes}m ago`
  }
  const hours = Math.round(minutes / 60)
  if (Math.abs(hours) < 24) {
    return `${hours}h ago`
  }
  const days = Math.round(hours / 24)
  return `${days}d ago`
}

export function average(values: number[]): number {
  if (values.length === 0) {
    return 0
  }
  return values.reduce((sum, value) => sum + value, 0) / values.length
}
