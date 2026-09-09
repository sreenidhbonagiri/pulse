import type { MouseEvent } from 'react'
import { Link } from 'react-router-dom'
import './Landing.css'

const GITHUB_URL = 'https://github.com/sreenidhbonagiri/pulse'

const features = [
  {
    title: 'Continuous uptime monitoring',
    body: 'Schedule HTTP checks against websites and APIs on a cadence you choose. Pulse records status codes, latency, and availability without blocking the request path.',
  },
  {
    title: 'Incident detection',
    body: 'Consecutive failures open an incident. Consecutive successes resolve it. One open incident per monitor keeps the signal clean for operators and dashboards.',
  },
  {
    title: 'Distributed job processing',
    body: 'A scheduler publishes due checks to Amazon SQS. Independent workers consume jobs, run the HTTP probe, retry internal failures, and dead-letter poison messages.',
  },
  {
    title: 'Historical latency and uptime',
    body: 'Every check is stored in PostgreSQL. The dashboard charts latency, uptime percentages, and incident history so you can see how an endpoint behaved over time.',
  },
] as const

const pipeline = ['Scheduler', 'Amazon SQS', 'Worker', 'HTTP Check', 'PostgreSQL'] as const

const stack = [
  'Go',
  'React',
  'TypeScript',
  'PostgreSQL',
  'Amazon SQS',
  'AWS ECS Fargate',
  'Terraform',
  'Docker',
  'GitHub Actions',
] as const

const highlights = [
  { value: 'HTTP', label: 'Website and API checks' },
  { value: '3 fails', label: 'Opens an incident' },
  { value: '2 oks', label: 'Resolves it again' },
  { value: '24h', label: 'Uptime and latency window' },
] as const

const monitorKinds = [
  {
    title: 'Public websites',
    body: 'Watch marketing sites and docs pages. Pulse records the status code you expect and how long the response took.',
  },
  {
    title: 'JSON APIs',
    body: 'Point a monitor at /health or any HTTP endpoint. Set method, timeout, interval, and the status that counts as healthy.',
  },
  {
    title: 'Internal services',
    body: 'Workers run the probes, not the dashboard. A slow check cannot stall the UI because jobs travel through SQS.',
  },
] as const

const steps = [
  {
    n: '01',
    title: 'Create a monitor',
    body: 'Name the endpoint, choose an HTTP method, and set interval, timeout, and expected status.',
  },
  {
    n: '02',
    title: 'Let the pipeline run',
    body: 'The scheduler finds due monitors and publishes jobs. Workers consume SQS messages and perform the HTTP check.',
  },
  {
    n: '03',
    title: 'Read the story',
    body: 'The dashboard shows current status, 24-hour uptime, latency percentiles, check history, and open incidents.',
  },
] as const

const internals = [
  {
    title: 'Retries with delay',
    body: 'Internal failures such as a database blip retry through delayed SQS messages instead of blocking a worker thread.',
  },
  {
    title: 'Dead-letter queue',
    body: 'Jobs that exhaust retries land on a DLQ so poison messages are visible instead of looping forever.',
  },
  {
    title: 'Idempotent results',
    body: 'Each job keeps a stable job ID. Duplicate deliveries cannot insert a second check result for the same attempt.',
  },
  {
    title: 'Cache when it helps',
    body: 'Stats can sit in Redis for a short window. If Redis is down, Pulse still serves numbers from PostgreSQL.',
  },
] as const

const sampleMonitors = [
  { name: 'Payments API', url: 'https://api.example.com/health', status: 'up' as const, uptime: '99.98%', latency: '41ms' },
  { name: 'Checkout', url: 'https://checkout.example.com', status: 'down' as const, uptime: '97.12%', latency: 'timeout' },
  { name: 'Docs', url: 'https://docs.example.com', status: 'up' as const, uptime: '100%', latency: '18ms' },
]

const faqs = [
  {
    q: 'What does Pulse actually check?',
    a: 'HTTP(S) endpoints. You configure the URL, method, interval, timeout, and expected status code. Pulse does not pretend to do ping, keyword, or DNS monitors.',
  },
  {
    q: 'How do incidents work?',
    a: 'Three consecutive failed checks open one incident per monitor. Two consecutive successful checks resolve it. Endpoint errors count; internal worker errors retry instead.',
  },
  {
    q: 'Is this a hosted SaaS?',
    a: 'Pulse is an open-source project with a live AWS demo. The source, Terraform, and GitHub Actions workflows are in the repository so you can see how it is built.',
  },
] as const

function statusLabel(status: 'up' | 'down') {
  return status === 'up' ? 'UP' : 'DOWN'
}

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function scrollBehavior(): ScrollBehavior {
  return prefersReducedMotion() ? 'auto' : 'smooth'
}

function scrollToSection(event: MouseEvent<HTMLAnchorElement>, id: string) {
  const section = document.getElementById(id)
  if (!section) return
  event.preventDefault()
  section.scrollIntoView({ behavior: scrollBehavior(), block: 'start' })
  history.replaceState(null, '', `#${id}`)
}

function scrollToTop(event: MouseEvent<HTMLAnchorElement>) {
  event.preventDefault()
  window.scrollTo({ top: 0, behavior: scrollBehavior() })
  history.replaceState(null, '', '/')
}

export function LandingPage() {
  return (
    <div className="landing">
      <header className="landing-nav">
        <Link to="/" className="landing-brand" onClick={scrollToTop}>
          <span className="landing-mark" aria-hidden="true" />
          Pulse
        </Link>
        <nav className="landing-anchors" aria-label="Page sections">
          <a href="#features" onClick={(event) => scrollToSection(event, 'features')}>
            Features
          </a>
          <a href="#incidents" onClick={(event) => scrollToSection(event, 'incidents')}>
            Incidents
          </a>
          <a href="#architecture" onClick={(event) => scrollToSection(event, 'architecture')}>
            Architecture
          </a>
        </nav>
        <div className="landing-nav-actions">
          <a className="landing-btn landing-btn-ghost" href={GITHUB_URL} target="_blank" rel="noreferrer">
            GitHub
          </a>
          <Link className="landing-btn landing-btn-primary" to="/dashboard">
            Open Dashboard
          </Link>
        </div>
      </header>

      <main>
        <section className="landing-hero">
          <div className="landing-hero-copy">
            <p className="landing-kicker">Uptime &amp; API monitoring</p>
            <h1>Know when an endpoint fails — and why it stayed down</h1>
            <p className="landing-lead">
              Pulse watches websites and APIs, records uptime and latency, and opens incidents when
              checks fail in a row. The dashboard stays snappy because probes run on a distributed
              Go pipeline: scheduler, Amazon SQS, and workers.
            </p>
            <div className="landing-hero-actions">
              <Link className="landing-btn landing-btn-primary" to="/dashboard">
                Open Dashboard
              </Link>
              <a className="landing-btn landing-btn-ghost" href={GITHUB_URL} target="_blank" rel="noreferrer">
                View GitHub
              </a>
            </div>
          </div>

          <aside className="landing-preview" aria-label="Example monitor status">
            <div className="landing-preview-top">
              <div>
                <p className="landing-preview-label">Live picture</p>
                <p className="landing-preview-name">What operators see</p>
              </div>
              <span className="landing-preview-count">3 monitors</span>
            </div>
            <ul className="landing-monitor-list">
              {sampleMonitors.map((monitor) => (
                <li key={monitor.name}>
                  <div>
                    <p className="landing-monitor-name">{monitor.name}</p>
                    <p className="landing-monitor-url">{monitor.url}</p>
                  </div>
                  <span className={`landing-status landing-status-${monitor.status}`}>
                    {statusLabel(monitor.status)}
                  </span>
                  <span className="landing-monitor-meta">{monitor.uptime}</span>
                  <span className="landing-monitor-meta">{monitor.latency}</span>
                </li>
              ))}
            </ul>
            <svg className="landing-wave" viewBox="0 0 320 88" role="img" aria-label="Latency waveform">
              <defs>
                <linearGradient id="pulse-wave" x1="0" x2="0" y1="0" y2="1">
                  <stop offset="0%" stopColor="#5ee0c8" stopOpacity="0.35" />
                  <stop offset="100%" stopColor="#5ee0c8" stopOpacity="0" />
                </linearGradient>
              </defs>
              <path
                className="landing-wave-fill"
                d="M0 58 C24 58 28 22 48 22 C68 22 72 64 96 64 C120 64 124 18 148 18 C172 18 176 52 200 52 C224 52 228 30 252 30 C276 30 280 70 304 70 L320 70 L320 88 L0 88 Z"
                fill="url(#pulse-wave)"
              />
              <path
                className="landing-wave-line"
                d="M0 58 C24 58 28 22 48 22 C68 22 72 64 96 64 C120 64 124 18 148 18 C172 18 176 52 200 52 C224 52 228 30 252 30 C276 30 280 70 304 70 L320 70"
                fill="none"
                stroke="#5ee0c8"
                strokeWidth="2.2"
                strokeLinecap="round"
              />
            </svg>
            <dl className="landing-preview-stats">
              <div>
                <dt>Fleet uptime</dt>
                <dd>99.03%</dd>
              </div>
              <div>
                <dt>Open incidents</dt>
                <dd>1</dd>
              </div>
              <div>
                <dt>Last check</dt>
                <dd>12s ago</dd>
              </div>
            </dl>
          </aside>
        </section>

        <section className="landing-strip" aria-label="Product rules">
          {highlights.map((item) => (
            <div key={item.label}>
              <p className="landing-strip-value">{item.value}</p>
              <p className="landing-strip-label">{item.label}</p>
            </div>
          ))}
        </section>

        <section className="landing-section" id="features">
          <p className="landing-kicker">Capabilities</p>
          <h2>What Pulse watches for you</h2>
          <div className="landing-feature-grid">
            {features.map((feature) => (
              <article key={feature.title} className="landing-card">
                <h3>{feature.title}</h3>
                <p>{feature.body}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="landing-section">
          <p className="landing-kicker">Coverage</p>
          <h2>Built for HTTP, not a kitchen sink</h2>
          <p className="landing-section-copy">
            Pulse focuses on websites and APIs. You get honest HTTP monitoring with the knobs that
            actually matter: method, interval, timeout, and expected status.
          </p>
          <div className="landing-kinds">
            {monitorKinds.map((kind) => (
              <article key={kind.title} className="landing-card">
                <h3>{kind.title}</h3>
                <p>{kind.body}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="landing-section" id="incidents">
          <p className="landing-kicker">Incidents</p>
          <h2>Noise stays out of the incident list</h2>
          <p className="landing-section-copy">
            A single timeout should not page anyone. Pulse waits for a streak, then opens exactly one
            incident until the endpoint recovers.
          </p>
          <ol className="landing-timeline">
            <li>
              <span>Fail</span>
              <p>Check 1 misses the expected status</p>
            </li>
            <li>
              <span>Fail</span>
              <p>Check 2 confirms it was not a blip</p>
            </li>
            <li className="landing-timeline-open">
              <span>Open</span>
              <p>Check 3 opens the incident</p>
            </li>
            <li>
              <span>Pass</span>
              <p>Recovery starts after a success</p>
            </li>
            <li className="landing-timeline-ok">
              <span>Resolved</span>
              <p>Two successes close it</p>
            </li>
          </ol>
        </section>

        <section className="landing-section">
          <p className="landing-kicker">Get started</p>
          <h2>From URL to history in three steps</h2>
          <div className="landing-steps">
            {steps.map((step) => (
              <article key={step.n} className="landing-card">
                <p className="landing-step-n">{step.n}</p>
                <h3>{step.title}</h3>
                <p>{step.body}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="landing-section" id="architecture">
          <p className="landing-kicker">Architecture</p>
          <h2>How it works</h2>
          <p className="landing-section-copy">
            The React dashboard talks to a Go API. Background checks are a separate pipeline so a
            slow probe never stalls the UI.
          </p>

          <div className="landing-flow" aria-label="Check pipeline">
            {pipeline.map((step, index) => (
              <div key={step} className="landing-flow-item">
                <div className="landing-flow-card">{step}</div>
                {index < pipeline.length - 1 ? (
                  <span className="landing-flow-arrow" aria-hidden="true">
                    →
                  </span>
                ) : null}
              </div>
            ))}
          </div>

          <div className="landing-control-plane" aria-label="Dashboard and API">
            <div className="landing-flow-card">React dashboard</div>
            <span className="landing-flow-arrow" aria-hidden="true">
              HTTP JSON
            </span>
            <div className="landing-flow-card">Go API</div>
            <span className="landing-flow-arrow" aria-hidden="true">
              reads / writes
            </span>
            <div className="landing-flow-card">PostgreSQL</div>
          </div>
        </section>

        <section className="landing-section">
          <p className="landing-kicker">Internals</p>
          <h2>Reliability that you can read in the source</h2>
          <p className="landing-section-copy">
            This is the part a hosted status tool usually hides. Pulse keeps retries, uniqueness, and
            failure isolation in the open.
          </p>
          <div className="landing-feature-grid">
            {internals.map((item) => (
              <article key={item.title} className="landing-card">
                <h3>{item.title}</h3>
                <p>{item.body}</p>
              </article>
            ))}
          </div>
        </section>

        <section className="landing-section">
          <p className="landing-kicker">Stack</p>
          <h2>Built with production tools</h2>
          <ul className="landing-stack">
            {stack.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        </section>

        <section className="landing-section" id="faq">
          <p className="landing-kicker">FAQ</p>
          <h2>Straight answers</h2>
          <div className="landing-faq">
            {faqs.map((item) => (
              <details key={item.q}>
                <summary>{item.q}</summary>
                <p>{item.a}</p>
              </details>
            ))}
          </div>
        </section>

        <section className="landing-cta">
          <h2>See Pulse in action</h2>
          <p>Open the live dashboard, add a monitor, and watch checks land. Or read the source on GitHub.</p>
          <div className="landing-hero-actions">
            <Link className="landing-btn landing-btn-primary" to="/dashboard">
              Open Dashboard
            </Link>
            <Link className="landing-btn landing-btn-ghost" to="/monitors/new">
              Add a monitor
            </Link>
          </div>
        </section>
      </main>

      <footer className="landing-footer">
        <span>Pulse</span>
        <span className="landing-footer-links">
          <a href="#features" onClick={(event) => scrollToSection(event, 'features')}>
            Features
          </a>
          <a href={GITHUB_URL} target="_blank" rel="noreferrer">
            GitHub
          </a>
          <Link to="/dashboard">Dashboard</Link>
        </span>
      </footer>
    </div>
  )
}
