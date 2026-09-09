locals {
  services = ["api", "worker", "scheduler"]
}

resource "aws_cloudwatch_log_group" "services" {
  for_each          = toset(local.services)
  name              = "/${var.project}/${var.environment}/${each.key}"
  retention_in_days = var.retention_days
}
