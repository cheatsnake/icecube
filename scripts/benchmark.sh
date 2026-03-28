#!/bin/bash
set -e

PORT=${ICECUBE_SERVER_PORT:-3331}
BASE_URL="http://localhost:$PORT"
IMAGE_URL="https://upload.wikimedia.org/wikipedia/commons/8/89/Durmast_House_and_Gardens.png"
NUM_JOBS=100

echo "=== Benchmark: $NUM_JOBS jobs ==="

# 1. Healthcheck
echo "--- 1. Healthcheck ---"
curl -s "$BASE_URL/api/v1/health" | jq . || { echo "Healthcheck failed"; exit 1; }

# 2. Download Image
echo "--- 2. Download Image ---"
TEMP_IMAGE=$(mktemp /tmp/benchmark_image.XXXXXX.png)
curl -s -o "$TEMP_IMAGE" "$IMAGE_URL"
echo "Downloaded to $TEMP_IMAGE"

# Cleanup temp image on exit
trap "rm -f $TEMP_IMAGE" EXIT

# 3. Upload Image
echo "--- 3. Upload Image ---"
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/images" -F "file=@$TEMP_IMAGE")
echo "$RESPONSE" | jq .
ORIGINAL_ID=$(echo "$RESPONSE" | jq -r '.[0].id')
echo "Original image ID: $ORIGINAL_ID"

# 4. Create all jobs in parallel
echo "--- 4. Create $NUM_JOBS jobs ---"
START_TIME=$(date +%s.%N)

declare -a JOB_IDS
declare -a PIDS

for i in $(seq 1 $NUM_JOBS); do
  FORMAT=$(jq -rn --argjson idx "$i" '[ "jpeg", "png", "webp" ][$idx % 3]')
  MAX_DIM=$((400 + (RANDOM % 601)))
  QUALITY=$((70 + (RANDOM % 26)))

  (
    RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/job" \
      -H "Content-Type: application/json" \
      -d "{\"originalID\": \"$ORIGINAL_ID\", \"options\": [{\"format\": \"$FORMAT\", \"maxDimension\": $MAX_DIM, \"quality\": $QUALITY, \"keepMetadata\": false}]}")
    echo "$RESPONSE" | jq -r '.id'
  ) > "/tmp/job_$i.txt" 2>&1 &

  PIDS+=($!)
done

# Wait for all job creations to complete
for pid in "${PIDS[@]}"; do
  wait $pid
done

# Collect job IDs
for i in $(seq 1 $NUM_JOBS); do
  JOB_ID=$(cat "/tmp/job_$i.txt")
  rm -f "/tmp/job_$i.txt"
  JOB_IDS+=("$JOB_ID")
  echo "Job $i: $JOB_ID"
done

JOBS_CREATED_TIME=$(date +%s.%N)
echo "All $NUM_JOBS jobs created"

# 5. Poll until all completed
echo "--- 5. Poll until all completed ---"
COMPLETED=0
FAILED=0
ATTEMPTS=0
MAX_ATTEMPTS=300
POLL_DELAY=1

while [ $COMPLETED -lt $NUM_JOBS ] && [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
  COMPLETED=0
  FAILED=0

  for JOB_ID in "${JOB_IDS[@]}"; do
    STATUS=$(curl -s "$BASE_URL/api/v1/job/$JOB_ID" 2>/dev/null | jq -r '.status')
    if [ "$STATUS" = "completed" ]; then
      COMPLETED=$((COMPLETED + 1))
    elif [ "$STATUS" = "failed" ]; then
      FAILED=$((FAILED + 1))
    fi
  done

  echo "Progress: $COMPLETED/$NUM_JOBS completed, $FAILED failed (attempt $ATTEMPTS)"
  ATTEMPTS=$((ATTEMPTS + 1))
  sleep $POLL_DELAY
done

END_TIME=$(date +%s.%N)
ELAPSED=$(echo "$END_TIME - $START_TIME" | bc)

echo ""
echo "=== Benchmark Results ==="
echo "Total jobs: $NUM_JOBS"
echo "Completed: $COMPLETED"
echo "Failed: $FAILED"
echo "Processing time: $(printf "%.1f" $(echo "$END_TIME - $JOBS_CREATED_TIME" | bc))s"
echo "Total time: $(printf "%.1f" $ELAPSED)s"
