# Contributing to Pulse

Thanks for your interest in contributing to Pulse.

Pulse is an open-source uptime and API monitoring platform built with Go, React, PostgreSQL, Amazon SQS, Terraform, and AWS.

Contributions are welcome for bug fixes, documentation improvements, tests, developer tooling, reliability improvements, and well-scoped features.

## Before You Start

Before making a contribution:

1. Check existing issues and pull requests to avoid duplicate work.
2. For larger changes, open an issue first so the approach can be discussed.
3. Keep changes focused and avoid unrelated refactors in the same pull request.
4. Do not include secrets, credentials, Terraform state, or environment files in commits.

## Development Setup

### Requirements

You will need:

- Go 1.25+
- Node.js 22+
- Docker
- Docker Compose
- PostgreSQL
- RabbitMQ for local queue development
- Redis for optional local statistics caching

Terraform and AWS CLI are only required if you are working on infrastructure.

## Clone the Repository

```bash
git clone https://github.com/sreenidhbonagiri/pulse.git
cd pulse
