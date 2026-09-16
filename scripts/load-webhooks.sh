#!/usr/bin/env bash
set -euo pipefail

task_url=${WEBHOOK_URL:-http://localhost:8080/webhooks/github}
task_secret=${WEBHOOK_SECRET_GITHUB:?set WEBHOOK_SECRET_GITHUB}
task_fixture=${WEBHOOK_FIXTURE:-internal/app/testdata/github.json}
task_requests=${REQUESTS:-100}
task_parallel=${PARALLEL:-20}
task_results=$(mktemp)
trap 'rm -f "$task_results"' EXIT

task_signature=$(openssl dgst -sha256 -hmac "$task_secret" -hex "$task_fixture" | sed 's/^.* //')
export task_url task_fixture task_signature task_results
seq "$task_requests" | xargs -P "$task_parallel" -n 1 sh -c '
  curl --silent --show-error --output /dev/null --write-out "%{http_code} %{time_total}\n" \
    -H "Content-Type: application/json" \
    -H "X-Hub-Signature-256: sha256=$task_signature" \
    --data-binary "@$task_fixture" "$task_url" >> "$task_results"
' sh

if ! awk '$1 == 202 { next } { exit 1 }' "$task_results"; then
  echo "at least one webhook was not accepted" >&2
  exit 1
fi
task_p95_index=$(( (task_requests * 95 + 99) / 100 ))
task_p95=$(sort -k2n "$task_results" | awk -v index="$task_p95_index" 'NR == index { print $2 }')
printf 'accepted=%s p95_seconds=%s\n' "$task_requests" "$task_p95"
