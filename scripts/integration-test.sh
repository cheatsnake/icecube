#!/bin/bash
set -e

PORT=${ICECUBE_SERVER_PORT:-3331}
BASE_URL="http://localhost:$PORT"
IMAGE_FILE="docs/architecture.png"
MAX_ATTEMPTS=10
ATTEMPT_DELAY=2

echo "--- 1. Healthcheck ---"
curl -s "$BASE_URL/api/v1/health" | jq . || { echo "Healthcheck failed"; exit 1; }

echo "--- 2. Upload Image ---"
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/images" -F "file=@$IMAGE_FILE")
echo "$RESPONSE" | jq .
ORIGINAL_ID=$(echo "$RESPONSE" | jq -r '.[0].id')
echo "Original image ID: $ORIGINAL_ID"

echo "--- 3. Create Job ---"
JOB_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/job" \
  -H "Content-Type: application/json" \
  -d "{\"originalID\": \"$ORIGINAL_ID\", \"options\": [{\"format\": \"webp\", \"maxDimension\": 800, \"quality\": 80}]}")
echo "$JOB_RESPONSE" | jq .
JOB_ID=$(echo "$JOB_RESPONSE" | jq -r '.id')
echo "Job ID: $JOB_ID"

echo "--- 4. Poll Job Status ---"
for i in $(seq 1 $MAX_ATTEMPTS); do
  STATUS=$(curl -s "$BASE_URL/api/v1/job/$JOB_ID" | jq -r '.status')
  echo "Attempt $i: status=$STATUS"
  if [ "$STATUS" = "completed" ]; then
    echo "Job completed!"
    break
  elif [ "$STATUS" = "failed" ]; then
    echo "Job failed!"
    curl -s "$BASE_URL/api/v1/job/$JOB_ID" | jq .
    exit 1
  fi
  sleep $ATTEMPT_DELAY
done

echo "--- 5. Get Processed Image Metadata ---"
VARIANT_ID=$(curl -s "$BASE_URL/api/v1/job/$JOB_ID" | jq -r '.tasks[0].variantID')
echo "Variant ID: $VARIANT_ID"
curl -s "$BASE_URL/api/v1/image/$VARIANT_ID/metadata" | jq .

echo "--- 6. Download Processed Image ---"
curl -s -o /tmp/processed_image.webp "$BASE_URL/image/$VARIANT_ID"
ls -la /tmp/processed_image.webp

echo "Integration test PASSED"
