import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { api } from '../api/client'
import type { MonitorInput } from '../api/types'

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'] as const

type FormErrors = Partial<Record<keyof MonitorInput, string>>

function validate(input: MonitorInput): FormErrors {
  const errors: FormErrors = {}
  if (!input.name.trim()) {
    errors.name = 'Name is required'
  }
  try {
    const parsed = new URL(input.url.trim())
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') {
      errors.url = 'URL must start with http:// or https://'
    }
  } catch {
    errors.url = 'Enter a valid URL'
  }
  if (!methods.includes(input.http_method as (typeof methods)[number])) {
    errors.http_method = 'Choose a valid HTTP method'
  }
  if (!Number.isInteger(input.check_interval_seconds) || input.check_interval_seconds <= 0) {
    errors.check_interval_seconds = 'Check interval must be greater than 0'
  }
  if (!Number.isInteger(input.timeout_seconds) || input.timeout_seconds <= 0) {
    errors.timeout_seconds = 'Timeout must be greater than 0'
  }
  if (
    !Number.isInteger(input.expected_status_code) ||
    input.expected_status_code < 100 ||
    input.expected_status_code > 599
  ) {
    errors.expected_status_code = 'Expected status must be between 100 and 599'
  }
  return errors
}

const emptyForm: MonitorInput = {
  name: '',
  url: '',
  http_method: 'GET',
  check_interval_seconds: 60,
  timeout_seconds: 5,
  expected_status_code: 200,
}

export function NewMonitorPage() {
  const navigate = useNavigate()
  const [form, setForm] = useState<MonitorInput>(emptyForm)
  const [errors, setErrors] = useState<FormErrors>({})
  const [submitError, setSubmitError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  function update<K extends keyof MonitorInput>(key: K, value: MonitorInput[K]) {
    setForm((current) => ({ ...current, [key]: value }))
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    const nextErrors = validate(form)
    setErrors(nextErrors)
    setSubmitError(null)
    if (Object.keys(nextErrors).length > 0) {
      return
    }

    setSaving(true)
    try {
      const created = await api.createMonitor({
        ...form,
        name: form.name.trim(),
        url: form.url.trim(),
        http_method: form.http_method.toUpperCase(),
      })
      navigate(`/monitors/${created.id}`)
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : 'Could not create monitor')
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="form-page">
      <div className="page-header">
        <div>
          <p className="eyebrow">Monitors</p>
          <h1>Add monitor</h1>
        </div>
      </div>

      <form className="panel form" onSubmit={onSubmit} noValidate>
        <label>
          Name
          <input
            value={form.name}
            onChange={(event) => update('name', event.target.value)}
            placeholder="Production API"
          />
          {errors.name ? <span className="field-error">{errors.name}</span> : null}
        </label>

        <label>
          URL
          <input
            value={form.url}
            onChange={(event) => update('url', event.target.value)}
            placeholder="https://api.example.com/health"
          />
          {errors.url ? <span className="field-error">{errors.url}</span> : null}
        </label>

        <div className="form-row">
          <label>
            HTTP method
            <select
              value={form.http_method}
              onChange={(event) => update('http_method', event.target.value)}
            >
              {methods.map((method) => (
                <option key={method} value={method}>
                  {method}
                </option>
              ))}
            </select>
            {errors.http_method ? <span className="field-error">{errors.http_method}</span> : null}
          </label>
          <label>
            Expected status code
            <input
              type="number"
              min={100}
              max={599}
              value={form.expected_status_code}
              onChange={(event) => update('expected_status_code', Number(event.target.value))}
            />
            {errors.expected_status_code ? (
              <span className="field-error">{errors.expected_status_code}</span>
            ) : null}
          </label>
        </div>

        <div className="form-row">
          <label>
            Check interval (seconds)
            <input
              type="number"
              min={1}
              value={form.check_interval_seconds}
              onChange={(event) => update('check_interval_seconds', Number(event.target.value))}
            />
            {errors.check_interval_seconds ? (
              <span className="field-error">{errors.check_interval_seconds}</span>
            ) : null}
          </label>
          <label>
            Timeout (seconds)
            <input
              type="number"
              min={1}
              value={form.timeout_seconds}
              onChange={(event) => update('timeout_seconds', Number(event.target.value))}
            />
            {errors.timeout_seconds ? <span className="field-error">{errors.timeout_seconds}</span> : null}
          </label>
        </div>

        {submitError ? <p className="form-error">{submitError}</p> : null}

        <div className="form-actions">
          <button className="button" type="submit" disabled={saving}>
            {saving ? 'Creating…' : 'Create monitor'}
          </button>
        </div>
      </form>
    </section>
  )
}
