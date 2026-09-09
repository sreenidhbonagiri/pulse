# Pulse

A distributed uptime and API monitoring platform built with Go, React, PostgreSQL, Amazon SQS, and AWS.

**Live Demo:** https://d1qd5as77atru7.cloudfront.net

## Overview

Pulse continuously checks websites and APIs, records availability and latency, tracks incidents, and displays historical monitoring data through a live dashboard.

The project is designed around a distributed backend architecture with separate API, scheduler, and worker services.

## Architecture

```text
React Dashboard
      |
      v
S3 + CloudFront
      |
      v
Application Load Balancer
      |
      v
ECS Fargate API
      |
      +----------------------+
      |                      |
      v                      v
RDS PostgreSQL          Amazon SQS
                              |
                              v
                         ECS Worker
                              |
                              v
                         HTTP Checks

ECS Scheduler
      |
      v
Amazon SQS
```

## Features

- Website and API uptime monitoring
- Configurable check intervals and timeouts
- HTTP status validation
- Latency tracking
- Historical check results
- Incident detection and resolution
- Automatic background scheduling
- Distributed job processing
- Retry handling with delayed messages
- Dead-letter queue support
- Idempotent job processing
- PostgreSQL persistence
- Optional Redis statistics caching
- Prometheus metrics
- Grafana dashboard
- Dockerized services
- AWS infrastructure managed with Terraform
- Automated frontend and backend deployment with GitHub Actions

## Tech Stack

### Frontend

- React
- TypeScript
- Vite

### Backend

- Go
- REST API
- PostgreSQL
- Amazon SQS
- RabbitMQ for local development
- Redis for optional statistics caching

### Infrastructure

- AWS ECS Fargate
- Amazon RDS
- Amazon SQS
- Amazon ECR
- Amazon S3
- Amazon CloudFront
- Application Load Balancer
- CloudWatch
- SSM Parameter Store
- Terraform
- Docker
- GitHub Actions

### Observability

- Prometheus
- Grafana
- CloudWatch Logs

## Local Development

Pulse uses PostgreSQL, RabbitMQ, and Redis for local development.

Start the local infrastructure:

```bash
cp .env.example .env
docker compose up -d
```

RabbitMQ management UI:

```text
http://localhost:15672
```

Default credentials:

```text
guest / guest
```

Redis is optional for statistics caching. If Redis is unavailable, Pulse falls back to PostgreSQL.

## Run the Backend

### API

```bash
cd backend
go run ./cmd/api
```

### Worker

In another terminal:

```bash
cd backend
go run ./cmd/worker
```

### Scheduler

In another terminal:

```bash
cd backend
go run ./cmd/scheduler
```

Health check:

```bash
curl http://localhost:8080/health
```

## Run the Frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

The frontend communicates with the API using:

```text
VITE_API_BASE_URL
```

The default local API URL is:

```text
http://localhost:8080
```

## Queue Providers

Pulse supports multiple queue backends through a shared queue abstraction.

For local development:

```text
QUEUE_PROVIDER=rabbitmq
```

For AWS:

```text
QUEUE_PROVIDER=sqs
```

The API, scheduler, and worker use the same job model regardless of the queue provider.

## Monitoring Flow

```text
Scheduler
   |
   v
Queue
   |
   v
Worker
   |
   v
HTTP Request
   |
   v
Check Result
   |
   v
PostgreSQL
```

The scheduler finds monitors whose next check is due and publishes jobs to the queue.

Workers consume those jobs, perform HTTP requests, measure latency, determine success or failure, and save the result.

## Monitors API

Create a monitor:

```bash
curl -X POST http://localhost:8080/api/monitors \
  -H "Content-Type: application/json" \
  -d '{
    "name":"My API",
    "url":"https://example.com/health",
    "http_method":"GET",
    "check_interval_seconds":60,
    "timeout_seconds":5,
    "expected_status_code":200
  }'
```

List monitors:

```bash
curl http://localhost:8080/api/monitors
```

Run a manual check:

```bash
curl -X POST http://localhost:8080/api/monitors/MONITOR_ID/check
```

View check history:

```bash
curl http://localhost:8080/api/monitors/MONITOR_ID/checks
```

View incidents:

```bash
curl http://localhost:8080/api/monitors/MONITOR_ID/incidents
```

View active incident:

```bash
curl http://localhost:8080/api/monitors/MONITOR_ID/incidents/active
```

View statistics:

```bash
curl http://localhost:8080/api/monitors/MONITOR_ID/stats
```

## Monitor Statistics

Pulse tracks:

- current status
- uptime percentage
- average latency
- latency percentiles
- total checks
- failed checks
- incident count
- active incident state

Statistics use recent monitoring history stored in PostgreSQL.

Redis can optionally cache statistics for faster reads.

The current AWS deployment does not run ElastiCache and serves statistics directly from PostgreSQL.

## Incidents

Pulse opens an incident after:

```text
3 consecutive failed checks
```

An incident resolves after:

```text
2 consecutive successful checks
```

Endpoint failures such as:

- HTTP 500 responses
- DNS failures
- timeouts
- unexpected status codes

are stored as failed check results.

Internal Pulse infrastructure failures are retried separately.

## Retries and Dead-Letter Queue

Internal worker failures are retried automatically.

```text
Attempt 1
   |
   | failure
   v
2 second delay
   |
   v
Attempt 2
   |
   | failure
   v
8 second delay
   |
   v
Attempt 3
   |
   | failure
   v
Dead-letter queue
```

Jobs preserve the same `job_id` across retries.

The database enforces idempotency so duplicate deliveries cannot create duplicate check results for the same job.

In AWS, retry delays are implemented with Amazon SQS delayed messages.

For local RabbitMQ development, retry queues use TTL-based delays.

## Observability

Pulse exposes Prometheus metrics from each Go process:

```text
API       http://localhost:8080/metrics
Worker    http://localhost:8081/metrics
Scheduler http://localhost:8082/metrics
```

Start Prometheus and Grafana with:

```bash
docker compose up -d
```

Prometheus:

```text
http://localhost:9090
```

Grafana:

```text
http://localhost:3000
```

AWS uses CloudWatch Logs for production logging.

Prometheus and Grafana remain local-only.

## Tests

From the backend directory:

```bash
cd backend
go test ./...
go build ./...
```

Some PostgreSQL-backed tests skip automatically if `DATABASE_URL` is unavailable.

Redis cache tests skip if `REDIS_URL` is unavailable.

## AWS Deployment

Infrastructure is managed with Terraform:

```text
infrastructure/terraform/
```

The production deployment includes:

- ECS Fargate API
- ECS Fargate worker
- ECS Fargate scheduler
- Amazon SQS
- RDS PostgreSQL
- Application Load Balancer
- S3
- CloudFront
- ECR
- CloudWatch
- SSM Parameter Store

The deployment intentionally does not use a NAT Gateway.

Redis/ElastiCache and Amazon MQ were removed from the always-on AWS architecture to reduce cost.

See:

- [Deployment Guide](docs/DEPLOYMENT.md)
- [AWS Cost Notes](docs/AWS_COSTS.md)

## CI/CD

Pulse uses GitHub Actions for automated deployment.

### Backend

Changes to backend files on `main` trigger:

```text
.github/workflows/deploy-backend.yml
```

The workflow:

1. runs Go tests
2. authenticates to AWS using GitHub OIDC
3. builds the API, worker, and scheduler Docker images
4. pushes the images to Amazon ECR
5. redeploys the three ECS services

### Frontend

Changes to frontend files on `main` trigger:

```text
.github/workflows/deploy-frontend.yml
```

The workflow:

1. installs frontend dependencies
2. builds the React application
3. authenticates to AWS using OIDC
4. uploads the build to S3
5. invalidates CloudFront

No long-lived AWS access keys are stored in GitHub.

## Terraform

Before applying infrastructure changes:

```bash
cd infrastructure/terraform
terraform fmt -recursive
terraform init
terraform validate
terraform plan
```

Review the plan before running:

```bash
terraform apply
```

Terraform state and local variable files are ignored by Git and should never be committed.

## Live Deployment

Dashboard:

```text
https://d1qd5as77atru7.cloudfront.net
```

API requests are proxied through the same CloudFront domain:

```text
https://d1qd5as77atru7.cloudfront.net/api/*
```
