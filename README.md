# Pulse

A distributed uptime and API monitoring platform. This repo currently has the initial project structure only — no monitoring features yet.

## Quick start

**Frontend**

```bash
cd frontend
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173).

**Backend** (requires [Go](https://go.dev/doc/install))

```bash
cd backend
go run ./cmd/api
```

Then test the health endpoint:

```bash
curl http://localhost:8080/health
```
