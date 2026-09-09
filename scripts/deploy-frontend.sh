#!/usr/bin/env bash
# Build the React app with the ALB API URL and sync it to S3 + CloudFront.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TF_DIR="${ROOT}/infrastructure/terraform"

API_URL="${VITE_API_BASE_URL:-$(terraform -chdir="${TF_DIR}" output -raw api_url)}"
BUCKET="$(terraform -chdir="${TF_DIR}" output -raw frontend_bucket)"
DISTRIBUTION_ID="$(terraform -chdir="${TF_DIR}" output -raw cloudfront_distribution_id)"

if [[ -z "${API_URL}" || -z "${BUCKET}" || -z "${DISTRIBUTION_ID}" ]]; then
  echo "Missing Terraform outputs. Run terraform apply first, or set VITE_API_BASE_URL." >&2
  exit 1
fi

echo "Building dashboard with VITE_API_BASE_URL=${API_URL}"
cd "${ROOT}/frontend"
VITE_API_BASE_URL="${API_URL}" npm ci
VITE_API_BASE_URL="${API_URL}" npm run build

aws s3 sync dist "s3://${BUCKET}" --delete
aws cloudfront create-invalidation --distribution-id "${DISTRIBUTION_ID}" --paths "/*"

echo "Uploaded to s3://${BUCKET}"
echo "Open the HTTPS CloudFront URL:"
terraform -chdir="${TF_DIR}" output -raw frontend_url
echo
