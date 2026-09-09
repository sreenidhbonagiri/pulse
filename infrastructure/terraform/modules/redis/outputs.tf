output "endpoint" {
  value = var.enabled ? aws_elasticache_cluster.this[0].cache_nodes[0].address : null
}

output "port" {
  value = var.enabled ? aws_elasticache_cluster.this[0].port : null
}
