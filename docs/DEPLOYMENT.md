# # AWS deployment

This is the first Pulse AWS deployment: production-shaped and student-budget-conscious.

The dashboard is served over HTTPS through CloudFront. API requests also go through CloudFront, which proxies `/api/*` to the Application Load Balancer.

Do **not** run `terraform apply` until you have reviewed the plan. These files create real AWS resources that can cost money.

## Architecture

```text

Internet

   |

   v

CloudFront

   |

   +----> private S3 bucket (React dashboard)

   |

   +----> /api/* -> Application Load Balancer

                          |

                          v

                    ECS Fargate API

                          |

                          +----> RDS PostgreSQL

ECS Fargate Scheduler

          |

          v

      Amazon SQS

          |

          v

ECS Fargate Worker

          |

          v

    HTTP monitor checks

          |

          v

    RDS PostgreSQL



The current AWS deployment uses:

- ECS Fargate for the API, worker, and scheduler
- Amazon SQS for background monitoring jobs
- RDS PostgreSQL for persistent data
- S3 and CloudFront for the frontend
- an Application Load Balancer for the API
- ECR for container images
- CloudWatch Logs for AWS-side logging
- SSM Parameter Store for connection strings
- GitHub Actions with AWS OIDC for automated deployment

Redis is supported by the application as an optional statistics cache, but the live AWS deployment currently runs with `enable_redis = false` and serves statistics directly from PostgreSQL.

RabbitMQ remains supported for local development, but it is not part of the deployed AWS infrastructure.

Prometheus and Grafana are currently local-only.

## Networking

The default deployment does **not** create a NAT Gateway.

ECS tasks run in public subnets with public IPs so they can:

- pull images from ECR
- write logs to CloudWatch
- access Amazon SQS
- perform outbound website checks

Inbound traffic remains restricted by security groups.

Only the API service is exposed through the Application Load Balancer.

The worker, scheduler, RDS database, and internal AWS resources are not directly exposed to the public internet.

## Desired task counts

Default development values:


| Variable                  | Default | Purpose                  |
| ------------------------- | ------- | ------------------------ |
| `api_desired_count`       | 1       | API service              |
| `worker_desired_count`    | 1       | background worker        |
| `scheduler_desired_count` | 1       | scheduled monitor checks |


Set a desired count to `0` if you intentionally want to stop that ECS service.

## Prerequisites

You need:

1. An AWS account
2. AWS CLI authentication for the target account
3. Terraform 1.5+
4. Docker
5. Node.js 22+
6. IAM permissions to create the resources used by the stack

Copy the example variables:

```

```

```

cd infrastructure/terraform
cp terraform.tfvars.example terraform.tfvars

```

Keep:

```

```

```

enable_nat_gateway = false
enable_redis       = false

```

for the current low-cost deployment unless you intentionally want to change the architecture.

## Terraform workflow

Before applying infrastructure changes:

```

```

```

cd infrastructure/terraform
terraform fmt -recursive
terraform init
terraform validate
terraform plan

```

Review the plan carefully.

Only then run:

```

```

```

terraform apply

```

## Backend deployment

The backend consists of three Go services:

-  API 
-  worker 
-  scheduler 

Each service has its own Dockerfile:

```

```

```

backend/Dockerfile.api
backend/Dockerfile.worker
backend/Dockerfile.scheduler

```

Images are pushed to ECR repositories:

```

```

```

pulse-dev/api
pulse-dev/worker
pulse-dev/scheduler

```

ECS Fargate runs those images.

The API is attached to the Application Load Balancer.

The worker and scheduler run as background ECS services.

## GitHub Actions backend deployment

Backend deployments are automated through:

```

```

```

.github/workflows/deploy-backend.yml

```

On pushes to `main` that modify backend files, GitHub Actions:

1.  checks out the repository 
2.  sets up Go 
3.  runs `go test ./...` 
4.  authenticates to AWS using GitHub OIDC 
5.  logs in to Amazon ECR 
6.  builds the API, worker, and scheduler images 
7.  pushes them to ECR 
8.  forces ECS to redeploy all three services 

No long-lived AWS access keys are stored in GitHub.

The GitHub Actions IAM role is managed by Terraform.

## Frontend deployment

The React frontend is built with Vite.

CloudFront serves the site from a private S3 bucket.

The production API base URL is:

```

```

```

[https://d1qd5as77atru7.cloudfront.net](https://d1qd5as77atru7.cloudfront.net)

```

CloudFront routes:

```

```

```

/api/*

```

to the Application Load Balancer.

This avoids browser mixed-content problems because both the frontend and API are accessed through HTTPS from the user's browser.

## GitHub Actions frontend deployment

Frontend deployments are automated through:

```

```

```

.github/workflows/deploy-frontend.yml

```

When frontend files are pushed to `main`, GitHub Actions:

1.  checks out the repository 
2.  installs Node.js 
3.  installs frontend dependencies 
4.  builds the Vite application 
5.  authenticates to AWS using OIDC 
6.  syncs `frontend/dist` to S3 
7.  creates a CloudFront invalidation 

This makes frontend changes available without manually running deployment scripts.

## API -> PostgreSQL

ECS receives the database connection string through SSM Parameter Store.

The connection string is stored as a `SecureString`.

The database is not publicly accessible.

The API and worker access PostgreSQL through security-group rules inside the VPC.

## Scheduler -> SQS -> Worker

The scheduler identifies monitors that are due and publishes jobs to Amazon SQS.

The worker consumes those messages and performs HTTP checks.

The main queue is:

```

```

```

pulse-dev-monitor-checks

```

The dead-letter queue is:

```

```

```

pulse-dev-monitor-checks-dlq

```

Pulse retries internal worker failures using delayed SQS messages.

The retry schedule is:

```

```

```

attempt 1
  |
  | failure
  v
2 second delay
  |
  v
attempt 2
  |
  | failure
  v
8 second delay
  |
  v
attempt 3
  |
  | failure
  v
dead-letter queue

```

Each job keeps the same `job_id` across retries.

The database enforces idempotency so duplicate deliveries cannot create duplicate check results for the same job.

Endpoint failures such as HTTP 500 responses, DNS failures, and timeouts are stored as normal failed check results rather than retried as infrastructure failures.

## Redis

Pulse supports Redis as an optional cache for monitor statistics.

The live AWS deployment currently uses:

```

```

```

enable_redis = false

```

When Redis is disabled, the application falls back to PostgreSQL.

This removes the cost of running an ElastiCache node continuously.

## Local RabbitMQ support

RabbitMQ is still supported for local development.

Local development can use:

```

```

```

QUEUE_PROVIDER=rabbitmq

```

The AWS deployment uses:

```

```

```

QUEUE_PROVIDER=sqs

```

The shared queue abstraction allows the same API, worker, and scheduler code to work with either backend.

## Secrets

Terraform stores connection strings in SSM Parameter Store as `SecureString`.

Current sensitive values include:

```

```

```

/${project}-${environment}/database_url
/${project}-${environment}/redis_url

```

ECS task definitions reference these through the execution role.

Terraform state can contain generated secrets, so do not commit:

```

```

```

*.tfstate
*.tfstate.*
terraform.tfvars
.terraform/

```

## Observability

AWS currently uses CloudWatch Logs for the API, worker, and scheduler.

Prometheus and Grafana remain local and can be started with Docker Compose.

Backend metrics are exposed on:

```

```

```

API       :8080/metrics
Worker    :8081/metrics
Scheduler :8082/metrics

```

## Destroying the stack

To preview destruction:

```

```

```

cd infrastructure/terraform
terraform plan -destroy

```

To destroy:

```



```

```

terraform destroy

```

The current student deployment is configured so destruction can complete without requiring an RDS final snapshot.

Destroying the stack deletes the database.

CloudFront can take several minutes to fully delete.

After destroying, verify in AWS that the following are gone:

-  ECS services 
-  RDS database 
-  Application Load Balancer 
-  CloudFront distribution 
-  S3 frontend bucket 
-  ECR repositories 
-  SQS queues 
-  VPC resources 

See [AWS_COSTS.md](AWS_COSTS.md) for cost notes.
```

