# Pulse

A distributed uptime and API monitoring platform.

## Quick start

**PostgreSQL and RabbitMQ**

```bash
cp .env.example .env
docker compose up -d
```

RabbitMQ management UI: [http://localhost:15672](http://localhost:15672) (user `guest`, password `guest`).

**Frontend**

```bash
cd frontend
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173).

**Backend** (requires [Go](https://go.dev/doc/install), PostgreSQL, and RabbitMQ)

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
curl http://localhost:8080/api/monitors/MONITOR_ID/checks
```

## Tests

From `backend/`:

```bash
go test ./...
go build ./...
```

Postgres-backed tests skip automatically if `DATABASE_URL` is unreachable.

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
