output "cluster_name" {
  value = aws_ecs_cluster.this.name
}

output "alb_dns_name" {
  value = aws_lb.api.dns_name
}

output "api_url" {
  value = "http://${aws_lb.api.dns_name}"
}

output "ecr_repository_urls" {
  value = { for k, repo in aws_ecr_repository.services : k => repo.repository_url }
}

output "ecr_repository_arns" {
  value = { for k, repo in aws_ecr_repository.services : k => repo.arn }
}