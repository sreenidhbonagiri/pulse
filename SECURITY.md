# Security Policy

## Supported Versions

Pulse is currently maintained as a portfolio and open-source project.

Security fixes are applied to the latest version of the `main` branch.

| Version | Supported |
| --- | --- |
| Latest `main` | Yes |
| Older commits/releases | No |

## Reporting a Vulnerability

Please do **not** open a public GitHub issue for security vulnerabilities.

If you discover a vulnerability, report it privately by emailing:

**sreenidhbonagiri@gmail.com**

Please include:

- a clear description of the issue
- steps to reproduce it
- the affected component or endpoint
- any relevant logs, screenshots, or proof-of-concept details
- the potential impact
- any suggested remediation, if known

I will review reports as soon as possible and may follow up for additional information.

## Scope

Security reports are especially helpful for issues involving:

- authentication or authorization flaws
- exposed credentials, tokens, or secrets
- API vulnerabilities
- SQL injection
- command injection
- cross-site scripting (XSS)
- cross-site request forgery (CSRF)
- server-side request forgery (SSRF)
- insecure deserialization
- privilege escalation
- sensitive data exposure
- AWS or Terraform misconfiguration
- GitHub Actions or CI/CD security
- S3 or CloudFront exposure
- IAM permission issues
- queue abuse or message tampering
- denial-of-service risks
- vulnerabilities that could affect the uptime-monitoring worker or scheduler

## Out of Scope

The following are generally out of scope unless they demonstrate a meaningful security impact:

- missing security headers without a working exploit
- rate-limit recommendations without evidence of abuse
- self-XSS
- clickjacking on pages without sensitive actions
- issues requiring access to the reporter's own local environment only
- vulnerabilities in unsupported third-party services that do not affect Pulse
- automated scanner output without reproduction steps
- denial-of-service testing against the live deployment

## Responsible Disclosure

Please avoid:

- accessing or modifying data that does not belong to you
- degrading or interrupting the live service
- performing destructive testing
- attempting to obtain persistence
- downloading large amounts of data
- publicly disclosing a vulnerability before it has been reviewed

Good-faith security research that follows this policy is appreciated.

## Secrets and Credentials

Pulse does not intentionally store secrets in the repository.

Production secrets and connection strings are managed through AWS Systems Manager Parameter Store and are injected into ECS tasks at runtime.

The following files and directories should never be committed:

```text
.env
.env.*
terraform.tfvars
*.tfstate
*.tfstate.*
.terraform/
