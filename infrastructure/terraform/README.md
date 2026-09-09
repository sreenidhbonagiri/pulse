# Terraform (AWS)

Generate AWS infrastructure for Pulse. **Do not apply until you have reviewed the plan.**

```bash
cp terraform.tfvars.example terraform.tfvars
terraform fmt -recursive
terraform init
terraform validate
terraform plan -var-file=terraform.tfvars
```

See [docs/DEPLOYMENT.md](../../docs/DEPLOYMENT.md) and [docs/AWS_COSTS.md](../../docs/AWS_COSTS.md).
