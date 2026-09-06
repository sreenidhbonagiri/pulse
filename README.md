# Pulse

A distributed uptime and API monitoring platform.

## Quick start

**PostgreSQL**

```bash
cp .env.example .env
docker compose up -d
```

**Frontend**

```bash
cd frontend
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173).

**Backend** (requires [Go](https://go.dev/doc/install) and PostgreSQL running)

```bash
cd backend
go run ./cmd/api
```

Then test the health endpoint:

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
