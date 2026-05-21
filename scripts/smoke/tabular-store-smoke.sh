#!/usr/bin/env bash
# path: scripts/smoke/tabular-store-smoke.sh
#
# Smoke test: ADR-0010 capability "Tabular store: append-only writes" (Yes).
# Creates a small dataset and table, inserts two rows via the
# tabledata.insertAll streaming-insert surface (append-only by
# construction; the engine never issues UPDATE/DELETE per ADR-0003),
# and verifies the resulting row count is exactly 2.
#
# The full lazy-view fidelity row in the ADR-0010 capability matrix
# is marked Partial; this smoke covers the append surface only.
# Full-view-semantics validation runs in the sandbox-required CI
# lane.

set -euo pipefail

HOST="${BIGQUERY_EMULATOR_HOST:-localhost:9050}"
PROJECT="${BIGQUERY_PROJECT_ID:-dq-local}"
DATASET="smoke_dataset_$$"
TABLE="smoke_table_$$"
BASE="http://${HOST}/bigquery/v2/projects/${PROJECT}"

echo "[tabular-store-smoke] base=${BASE}"

cleanup() {
  curl -sf -X DELETE "${BASE}/datasets/${DATASET}?deleteContents=true" >/dev/null || true
}
trap cleanup EXIT

# 1. Create dataset.
curl -sf -X POST "${BASE}/datasets" \
  -H "Content-Type: application/json" \
  -d "{\"datasetReference\":{\"projectId\":\"${PROJECT}\",\"datasetId\":\"${DATASET}\"}}" >/dev/null
echo "[tabular-store-smoke] dataset created: ${DATASET}"

# 2. Create a minimal append-only table mirroring the dq_executions
#    column shape from ADR-0003 (just three columns for the smoke).
curl -sf -X POST "${BASE}/datasets/${DATASET}/tables" \
  -H "Content-Type: application/json" \
  -d "{
    \"tableReference\":{\"projectId\":\"${PROJECT}\",\"datasetId\":\"${DATASET}\",\"tableId\":\"${TABLE}\"},
    \"schema\":{\"fields\":[
      {\"name\":\"execution_id\",\"type\":\"STRING\",\"mode\":\"REQUIRED\"},
      {\"name\":\"attempt_id\",\"type\":\"STRING\",\"mode\":\"REQUIRED\"},
      {\"name\":\"recorded_at\",\"type\":\"TIMESTAMP\",\"mode\":\"REQUIRED\"}
    ]}
  }" >/dev/null
echo "[tabular-store-smoke] table created: ${TABLE}"

# 3. Insert two rows via tabledata.insertAll.
curl -sf -X POST "${BASE}/datasets/${DATASET}/tables/${TABLE}/insertAll" \
  -H "Content-Type: application/json" \
  -d '{
    "rows":[
      {"json":{"execution_id":"smoke-exec-1","attempt_id":"attempt-1","recorded_at":"2026-05-21T00:00:00Z"}},
      {"json":{"execution_id":"smoke-exec-2","attempt_id":"attempt-2","recorded_at":"2026-05-21T00:00:01Z"}}
    ]
  }' >/dev/null
echo "[tabular-store-smoke] two rows inserted"

# 4. Query the row count.
QUERY_RESPONSE=$(curl -sf -X POST "${BASE}/queries" \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"SELECT COUNT(*) AS n FROM \`${PROJECT}.${DATASET}.${TABLE}\`\",\"useLegacySql\":false}")

# Parse the first cell of the first row. BigQuery's response shape:
# rows[0].f[0].v is the value.
COUNT=$(echo "${QUERY_RESPONSE}" | python3 -c 'import json,sys; r=json.load(sys.stdin); print(r["rows"][0]["f"][0]["v"])')

if [[ "${COUNT}" != "2" ]]; then
  echo "[tabular-store-smoke] FAIL: row count is ${COUNT}, expected 2"
  echo "[tabular-store-smoke] full response: ${QUERY_RESPONSE}"
  exit 1
fi
echo "[tabular-store-smoke] row count = 2 OK"

echo "[tabular-store-smoke] OK"
