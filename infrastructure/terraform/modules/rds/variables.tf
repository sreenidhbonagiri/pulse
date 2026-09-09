variable "project" { type = string }
variable "environment" { type = string }
variable "vpc_id" { type = string }
variable "subnet_ids" { type = list(string) }
variable "security_group_id" { type = string }
variable "instance_class" { type = string }
variable "allocated_storage" { type = number }
variable "master_username" { type = string }
variable "master_password" {
  type      = string
  sensitive = true
}
variable "database_name" { type = string }
