output "api_url" {
  description = "Public HTTP URL of the API load balancer."
  value       = module.ecs.api_url
}

output "alb_dns_name" {
  description = "API Application Load Balancer DNS name."
  value       = module.ecs.alb_dns_name
}

output "frontend_url" {
  description = "HTTPS CloudFront URL for the dashboard."
  value       = module.frontend.cloudfront_url
}

output "frontend_http_url" {
  description = "HTTP CloudFront URL; CloudFront redirects this to HTTPS."
  value       = "http://${module.frontend.cloudfront_domain_name}"
}

output "cloudfront_distribution_id" {
  value = module.frontend.distribution_id
}

output "frontend_bucket" {
  value = module.frontend.bucket_name
}

output "ecr_repository_urls" {
  value = module.ecs.ecr_repository_urls
}

output "rds_endpoint" {
  value = module.rds.endpoint
}

output "redis_endpoint" {
  value = module.redis.endpoint
}



output "ecs_cluster_name" {
  value = module.ecs.cluster_name
}
