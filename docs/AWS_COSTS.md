# # AWS cost notes

Pulse is deployed as a student portfolio project.

The goal is to keep the application publicly accessible while avoiding unnecessary always-on AWS services.

Prices vary by region and can change over time, so this document focuses on relative cost rather than exact dollar amounts.

## Current architecture

The deployed stack currently uses:

- Application Load Balancer

- RDS PostgreSQL

- three ECS Fargate services

- Amazon SQS

- CloudFront

- S3

- ECR

- CloudWatch Logs

- SSM Parameter Store

- VPC networking

The deployment intentionally does **not** use:

- NAT Gateway

- Amazon MQ

- ElastiCache Redis

- Multi-AZ RDS

- Prometheus or Grafana on AWS

## Main cost drivers

The largest continuously running costs are expected to come from:

1. RDS PostgreSQL

2. Application Load Balancer

3. ECS Fargate API

4. ECS Fargate worker

5. ECS Fargate scheduler

6. public IPv4 addresses used by ECS tasks

Amazon SQS, S3, CloudFront, SSM Parameter Store, and ECR are comparatively small for low portfolio traffic.

## Continuously running resources

| Resource | Current state | Relative cost |

| --- | --- | --- |

| Application Load Balancer | On | Medium |

| RDS PostgreSQL `db.t4g.micro` | On | Medium |

| ECS Fargate API | 1 task | Low-Medium |

| ECS Fargate worker | 1 task | Low-Medium |

| ECS Fargate scheduler | 1 task | Low-Medium |

| Public IPv4 for ECS tasks | On | Low-Medium |

| CloudFront | On | Low |

| S3 frontend bucket | On | Low |

| ECR repositories | On | Low |

| Amazon SQS | On | Low |

| CloudWatch Logs | On | Low |

| SSM Parameter Store | On | Low |

## Cost optimizations already implemented

### No NAT Gateway

The deployment uses:

```hcl

enable_nat_gateway = false



ECS tasks run in public subnets with public IPs.

Security groups still restrict inbound traffic.

This avoids the fixed hourly cost of a NAT Gateway.

### Amazon SQS instead of RabbitMQ infrastructure

The original cloud design used RabbitMQ.

The production deployment now uses Amazon SQS instead.

This removed the need for:

- an ECS RabbitMQ task
- Amazon MQ
- RabbitMQ-specific passwords and infrastructure
- persistent RabbitMQ broker cost

RabbitMQ remains available for local development.

### Redis disabled in AWS

The application still supports Redis caching, but the current AWS deployment uses:

```

```

```
enable_redis = false
```

Statistics fall back to PostgreSQL.

This avoids running an ElastiCache node continuously.

### Small RDS instance

The database uses a small single-AZ instance intended for portfolio-scale traffic.

The deployment does not use Multi-AZ.

### Minimal ECS task sizes

The API, worker, and scheduler each use small Fargate task sizes appropriate for this project.

### Short CloudWatch retention

Logs use limited retention instead of being stored indefinitely.

### No Prometheus or Grafana in AWS

Prometheus and Grafana run locally only.

Deploying them as additional AWS services would increase continuously running compute and storage costs.

## Services intentionally avoided


| Service               | Current state |
| --------------------- | ------------- |
| NAT Gateway           | Off           |
| Amazon MQ             | Removed       |
| ElastiCache Redis     | Disabled      |
| Multi-AZ RDS          | Off           |
| AWS-hosted Grafana    | Off           |
| AWS-hosted Prometheus | Off           |


## Pausing ECS compute

To stop the three Fargate services, set:

```

```

```
api_desired_count       = 0
worker_desired_count    = 0
scheduler_desired_count = 0
```

Then run:

```

```

```
terraform plan
terraform apply
```

This stops Fargate compute and removes the public IPv4 addresses attached to those running tasks.

However, the following resources still remain and may continue costing money:

-  RDS 
-  Application Load Balancer 
-  CloudFront 
-  S3 
-  ECR 
-  stored CloudWatch logs 
-  SQS queues 

Because Pulse is intended to remain publicly accessible as a portfolio project, the current deployment normally keeps the three ECS services running.

## Full shutdown

The only way to remove nearly all ongoing infrastructure cost is:

```

```

```
cd infrastructure/terraform
terraform destroy
```

This removes the AWS stack.

The current Terraform configuration allows the RDS instance to be destroyed without creating a final snapshot.

That means the database contents are deleted.

## What to verify after destroy

After destroying the stack, verify in AWS that there are no remaining:

-  RDS instances 
-  Application Load Balancers 
-  ECS services 
-  NAT Gateways 
-  ElastiCache nodes 
-  Amazon MQ brokers 
-  SQS queues 
-  CloudFront distributions 
-  unused Elastic IPs 

## Billing alerts

AWS Budgets can alert when estimated spend crosses a threshold.

Budget alerts do not automatically stop resources.

They should be treated as monitoring, not as a hard spending limit.

## Current cost strategy

Pulse currently favors:

```

```

```
public portfolio availability
+
real AWS infrastructure
+
minimal always-on services
```

over the cheapest possible architecture.

This keeps the project useful as a live demo while still avoiding some of the most expensive default cloud patterns.

```

```

```

```

