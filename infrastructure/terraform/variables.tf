variable "aws_region" {
  description = "AWS region for all resources."
  type        = string
  default     = "us-east-1"
}

variable "project" {
  description = "Name prefix used on AWS resources."
  type        = string
  default     = "pulse"
}

variable "environment" {
  description = "Environment name (dev, staging, prod)."
  type        = string
  default     = "dev"
}

variable "image_tag" {
  description = "ECR image tag to run for api, worker, and scheduler."
  type        = string
  default     = "latest"
}

variable "cors_origins" {
  description = "Extra CORS origins besides the CloudFront URL. Comma-separated."
  type        = string
  default     = ""
}

variable "api_desired_count" {
  description = "Desired API Fargate tasks. Set to 0 to stop the API when not demoing."
  type        = number
  default     = 1
}

variable "worker_desired_count" {
  description = "Desired worker Fargate tasks. Set to 0 to stop workers when not demoing."
  type        = number
  default     = 1
}

variable "scheduler_desired_count" {
  description = "Desired scheduler Fargate tasks. Keep at 1 while demoing; set to 0 when not demoing."
  type        = number
  default     = 1
}

variable "rabbitmq_desired_count" {
  description = "Desired RabbitMQ Fargate tasks. Default 1 so the broker actually runs. Set to 0 to stop RabbitMQ and save Fargate cost when not demoing."
  type        = number
  default     = 1
}

variable "rabbitmq_cpu" {
  description = "Fargate CPU units for the ECS RabbitMQ task (256 = 0.25 vCPU)."
  type        = number
  default     = 256
}

variable "rabbitmq_memory" {
  description = "Fargate memory (MiB) for the ECS RabbitMQ task."
  type        = number
  default     = 512
}

variable "api_cpu" {
  type    = number
  default = 256
}

variable "api_memory" {
  type    = number
  default = 512
}

variable "worker_cpu" {
  type    = number
  default = 256
}

variable "worker_memory" {
  type    = number
  default = 512
}

variable "scheduler_cpu" {
  type    = number
  default = 256
}

variable "scheduler_memory" {
  type    = number
  default = 512
}

variable "enable_nat_gateway" {
  description = "Create a NAT Gateway and place ECS tasks in private subnets. Default false because NAT is a high ongoing cost."
  type        = bool
  default     = false
}

variable "enable_redis" {
  description = "Create ElastiCache Redis. Pulse still serves stats from Postgres if Redis is disabled."
  type        = bool
  default     = true
}

variable "rabbitmq_engine" {
  description = "RabbitMQ hosting: ecs (cheaper, default) or amazonmq (managed, higher cost)."
  type        = string
  default     = "ecs"

  validation {
    condition     = contains(["ecs", "amazonmq"], var.rabbitmq_engine)
    error_message = "rabbitmq_engine must be ecs or amazonmq."
  }
}

variable "db_instance_class" {
  type    = string
  default = "db.t4g.micro"
}

variable "db_allocated_storage" {
  type    = number
  default = 20
}

variable "redis_node_type" {
  type    = string
  default = "cache.t4g.micro"
}

variable "log_retention_days" {
  type    = number
  default = 7
}

variable "vpc_cidr" {
  type    = string
  default = "10.0.0.0/16"
}
