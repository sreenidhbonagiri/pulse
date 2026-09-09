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
