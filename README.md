# Pulse

A distributed uptime and API monitoring platform.

## Quick start

**PostgreSQL, RabbitMQ, and Redis**

```bash
cp .env.example .env
docker compose up -d
```

RabbitMQ management UI: [http://localhost:15672](http://localhost:15672) (user `guest`, password `guest`).
Redis is used only to cache monitor statistics. The API still works if Redis is down.

**Frontend**

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173). The dashboard talks to the API using `VITE_API_BASE_URL` (default `http://localhost:8080`).

**Run frontend and backend together**

1. Start Postgres, RabbitMQ, and Redis: `docker compose up -d`
2. Start the API: `cd backend && go run ./cmd/api`
3. Start the worker and scheduler in other terminals (`go run ./cmd/worker`, `go run ./cmd/scheduler`) so checks actually run
4. Start the dashboard: `cd frontend && npm run dev`
5. Open [http://localhost:5173](http://localhost:5173)

The API allows those browser origins through `CORS_ORIGINS` (default `http://localhost:5173,http://127.0.0.1:5173`). Restart the API after changing that variable.

**Backend** (requires [Go](https://go.dev/doc/install), PostgreSQL, and RabbitMQ. Redis is optional.)

API:

```bash
cd backend
go run ./cmd/api
```

Worker (separate terminal):

```bash
cd backend
go run ./cmd/worker
```

Scheduler (separate terminal):

```bash
cd backend
go run ./cmd/scheduler
```

Health check:

```bash
curl http://localhost:8080/health
```

**Observability (Prometheus + Grafana)**

```bash
docker compose up -d
```

Prometheus scrapes the Go processes on the host through `host.docker.internal`:

- API: [http://localhost:8080/metrics](http://localhost:8080/metrics)
- Worker: [http://localhost:8081/metrics](http://localhost:8081/metrics)
- Scheduler: [http://localhost:8082/metrics](http://localhost:8082/metrics)

Open Prometheus at [http://localhost:9090](http://localhost:9090) and Grafana at [http://localhost:3000](http://localhost:3000) (anonymous viewer, or `admin` / `admin`). The Pulse dashboard and Prometheus datasource are provisioned from `monitoring/`.

If Prometheus or Grafana is down, the API, worker, and scheduler keep running.

**Monitors API**

```bash
curl -X POST http://localhost:8080/api/monitors \
  -H "Content-Type: application/json" \
  -d '{"name":"My API","url":"https://example.com/health","http_method":"GET","check_interval_seconds":60,"timeout_seconds":5,"expected_status_code":200}'

curl http://localhost:8080/api/monitors
```

**Enqueue a check** (the worker performs it in the background)

```bash
curl -X POST http://localhost:8080/api/monitors/MONITOR_ID/check
curl -s http://localhost:8080/api/monitors/MONITOR_ID/checks
curl -s http://localhost:8080/api/monitors/MONITOR_ID/incidents
curl -s http://localhost:8080/api/monitors/MONITOR_ID/incidents/active
curl -s http://localhost:8080/api/monitors/MONITOR_ID/stats
```

## Monitor statistics

`GET /api/monitors/{id}/stats` returns current status, uptime, latency percentiles, check counts, and incident summary.

- Check totals, uptime, and latency use the **last 24 hours**
- `current_status` is the latest check overall (`up`, `down`, or `unknown`)
- `active_incident` is the current open incident, if any
- Results are cached in Redis for 45 seconds (cache-aside)
- A new check or incident change deletes that monitor's cache entry

## Incidents

Pulse opens an incident after **3 consecutive failed checks** and resolves it after **2 consecutive successful checks**. Endpoint failures such as HTTP 500 count as failed checks. Internal Pulse errors still retry and do not open incidents by themselves.

## Tests

From `backend/`:

```bash
go test ./...
go build ./...
```

Postgres-backed tests skip automatically if `DATABASE_URL` is unreachable. Redis cache tests skip if `REDIS_URL` is unreachable.

## Retries and dead-letter queue

Internal worker failures (for example PostgreSQL unavailable) retry through RabbitMQ, not `time.Sleep`. Endpoint failures such as HTTP 500, timeouts, or DNS errors are saved as normal `CheckResult` rows and are not retried.

Topology (declared on API, worker, and scheduler startup):

- exchange `pulse.jobs` (direct, durable)
- queue `monitor.checks` (workers consume this)
- queue `monitor.checks.retry.1` (TTL 2s, dead-letters back to `monitor.checks`)
- queue `monitor.checks.retry.2` (TTL 8s, dead-letters back to `monitor.checks`)
- queue `monitor.checks.dlq` (not consumed by the worker)

A job keeps the same `job_id` across retries. After 3 failed attempts it is published to `monitor.checks.dlq`. Duplicate deliveries cannot insert a second `CheckResult` for the same `job_id`.

**Inspect queues** in the [RabbitMQ management UI](http://localhost:15672) or with:

```bash
docker compose exec rabbitmq rabbitmqctl list_queues name messages
```
