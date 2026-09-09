#!/usr/bin/env bash
# Build linux/amd64 images and push them to the Pulse ECR repositories.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AWS_REGION="${AWS_REGION:-us-east-1}"
PROJECT="${PROJECT:-pulse}"
ENVIRONMENT="${ENVIRONMENT:-dev}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
PREFIX="${PROJECT}-${ENVIRONMENT}"

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
REGISTRY="${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"

aws ecr get-login-password --region "${AWS_REGION}" \
  | docker login --username AWS --password-stdin "${REGISTRY}"

build_push() {
  local service="$1"
  local dockerfile="$2"
  local image="${REGISTRY}/${PREFIX}/${service}:${IMAGE_TAG}"

  docker build \
    --platform linux/amd64 \
    -f "${ROOT}/backend/${dockerfile}" \
    -t "${image}" \
    "${ROOT}/backend"

  docker push "${image}"
  echo "Pushed ${image}"
}

build_push api Dockerfile.api
build_push worker Dockerfile.worker
build_push scheduler Dockerfile.scheduler
