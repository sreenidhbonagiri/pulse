locals {
  name = "${var.project}-${var.environment}"
}

resource "aws_elasticache_subnet_group" "this" {
  count      = var.enabled ? 1 : 0
  name       = "${local.name}-redis"
  subnet_ids = var.subnet_ids
}

resource "aws_elasticache_cluster" "this" {
  count                    = var.enabled ? 1 : 0
  cluster_id               = "${local.name}-redis"
  engine                   = "redis"
  engine_version           = "7.1"
  node_type                = var.node_type
  num_cache_nodes          = 1
  port                     = 6379
  parameter_group_name     = "default.redis7"
  subnet_group_name        = aws_elasticache_subnet_group.this[0].name
  security_group_ids       = [var.security_group_id]
  snapshot_retention_limit = 0
  apply_immediately        = true
}
