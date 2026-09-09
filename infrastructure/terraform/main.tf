data "aws_caller_identity" "current" {}

locals {
  name = "${var.project}-${var.environment}"

  ecs_subnet_ids   = var.enable_nat_gateway ? module.networking.private_subnet_ids : module.networking.public_subnet_ids
  assign_public_ip = !var.enable_nat_gateway

  # Amazon MQ belongs in private subnets. The default ECS broker uses public
  # subnets (no NAT) so it can pull its image through the Internet Gateway.


  extra_cors = [
    for origin in split(",", var.cors_origins) : trimspace(origin)
    if trimspace(origin) != ""
  ]
  cors_origins = join(",", concat([
    module.frontend.cloudfront_url,
  ], local.extra_cors))

  database_url = "postgres://pulse:${random_password.db.result}@${module.rds.endpoint}:${module.rds.port}/pulse?sslmode=require"
  redis_url    = var.enable_redis ? "redis://${module.redis.endpoint}:${module.redis.port}/0" : "redis://127.0.0.1:9/0"

}

resource "random_password" "db" {
  length  = 24
  special = false
}



module "networking" {
  source = "./modules/networking"

  project            = var.project
  environment        = var.environment
  vpc_cidr           = var.vpc_cidr
  enable_nat_gateway = var.enable_nat_gateway
}

module "observability" {
  source = "./modules/observability"

  project        = var.project
  environment    = var.environment
  retention_days = var.log_retention_days
}

module "frontend" {
  source = "./modules/frontend"

  project         = var.project
  environment     = var.environment
  account_id      = data.aws_caller_identity.current.account_id
  api_domain_name = module.ecs.alb_dns_name
}

module "rds" {
  source = "./modules/rds"

  project           = var.project
  environment       = var.environment
  vpc_id            = module.networking.vpc_id
  subnet_ids        = module.networking.private_subnet_ids
  security_group_id = module.networking.rds_security_group_id
  instance_class    = var.db_instance_class
  allocated_storage = var.db_allocated_storage
  master_username   = "pulse"
  master_password   = random_password.db.result
  database_name     = "pulse"
}

module "redis" {
  source = "./modules/redis"

  project           = var.project
  environment       = var.environment
  vpc_id            = module.networking.vpc_id
  subnet_ids        = module.networking.private_subnet_ids
  security_group_id = module.networking.redis_security_group_id
  node_type         = var.redis_node_type
  enabled           = var.enable_redis
}





resource "aws_ssm_parameter" "database_url" {
  name  = "/${local.name}/database_url"
  type  = "SecureString"
  value = local.database_url
}

resource "aws_ssm_parameter" "redis_url" {
  name  = "/${local.name}/redis_url"
  type  = "SecureString"
  value = local.redis_url
}


resource "aws_sqs_queue" "monitor_checks_dlq" {
  name = "${local.name}-monitor-checks-dlq"

  message_retention_seconds = 1209600
}

resource "aws_sqs_queue" "monitor_checks" {
  name = "${local.name}-monitor-checks"

  visibility_timeout_seconds = 30
  receive_wait_time_seconds  = 20
  message_retention_seconds  = 345600
}

module "ecs" {
  source = "./modules/ecs"

  project                     = var.project
  environment                 = var.environment
  aws_region                  = var.aws_region
  vpc_id                      = module.networking.vpc_id
  public_subnet_ids           = module.networking.public_subnet_ids
  ecs_subnet_ids              = local.ecs_subnet_ids
  assign_public_ip            = local.assign_public_ip
  alb_security_group_id       = module.networking.alb_security_group_id
  api_security_group_id       = module.networking.api_security_group_id
  worker_security_group_id    = module.networking.worker_security_group_id
  scheduler_security_group_id = module.networking.scheduler_security_group_id
  image_tag                   = var.image_tag
  api_desired_count           = var.api_desired_count
  worker_desired_count        = var.worker_desired_count
  scheduler_desired_count     = var.scheduler_desired_count
  api_cpu                     = var.api_cpu
  api_memory                  = var.api_memory
  worker_cpu                  = var.worker_cpu
  worker_memory               = var.worker_memory
  scheduler_cpu               = var.scheduler_cpu
  scheduler_memory            = var.scheduler_memory
  log_group_names = {
    for name, group in module.observability.log_group_names : name => group
    if name != "rabbitmq"
  }
  database_url_parameter_arn = aws_ssm_parameter.database_url.arn
  redis_url_parameter_arn    = aws_ssm_parameter.redis_url.arn

  sqs_queue_url = aws_sqs_queue.monitor_checks.url
  sqs_dlq_url   = aws_sqs_queue.monitor_checks_dlq.url
  sqs_queue_arn = aws_sqs_queue.monitor_checks.arn
  sqs_dlq_arn   = aws_sqs_queue.monitor_checks_dlq.arn

  cors_origins = local.cors_origins

}
