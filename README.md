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
