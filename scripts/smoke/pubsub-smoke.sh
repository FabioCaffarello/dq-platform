#!/usr/bin/env bash
# path: scripts/smoke/pubsub-smoke.sh
#
# Smoke test: ADR-0010 capability "Pub/Sub publish/subscribe" (Yes).
# Round-trips one message through the local Pub/Sub emulator:
# create topic → create subscription → publish → pull → ack.
# Exits non-zero on any failure (the Pub/Sub emulator is mandatory
# for Phase 2 closing per ADR-0010).

set -euo pipefail

HOST="${PUBSUB_EMULATOR_HOST:-localhost:8085}"
PROJECT="${PUBSUB_PROJECT_ID:-dq-local}"
TOPIC="smoke-topic-$$"
SUB="smoke-sub-$$"
BASE="http://${HOST}/v1/projects/${PROJECT}"

echo "[pubsub-smoke] base=${BASE}"

cleanup() {
  curl -sf -X DELETE "${BASE}/subscriptions/${SUB}" >/dev/null || true
  curl -sf -X DELETE "${BASE}/topics/${TOPIC}" >/dev/null || true
}
trap cleanup EXIT

# 1. Create topic.
curl -sf -X PUT "${BASE}/topics/${TOPIC}" >/dev/null
echo "[pubsub-smoke] topic created: ${TOPIC}"

# 2. Create subscription on that topic.
curl -sf -X PUT "${BASE}/subscriptions/${SUB}" \
  -H "Content-Type: application/json" \
  -d "{\"topic\":\"projects/${PROJECT}/topics/${TOPIC}\"}" >/dev/null
echo "[pubsub-smoke] subscription created: ${SUB}"

# 3. Publish one message (base64-encoded "hello").
curl -sf -X POST "${BASE}/topics/${TOPIC}:publish" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"data":"aGVsbG8="}]}' >/dev/null
echo "[pubsub-smoke] message published"

# 4. Pull. Use maxMessages=1 and returnImmediately so a missing
#    message exits cleanly rather than blocking.
PULL_RESPONSE=$(curl -sf -X POST "${BASE}/subscriptions/${SUB}:pull" \
  -H "Content-Type: application/json" \
  -d '{"maxMessages":1,"returnImmediately":true}')

# Parse the ackId out of the pull response. python is used (instead
# of grep) because the emulator emits pretty-printed JSON with
# whitespace, and a tolerant JSON parse is more robust than a
# brittle regex.
ACK_ID=$(printf '%s' "${PULL_RESPONSE}" | python3 -c '
import json, sys
data = json.load(sys.stdin)
msgs = data.get("receivedMessages", [])
print(msgs[0]["ackId"] if msgs else "")
')

if [[ -z "${ACK_ID}" ]]; then
  echo "[pubsub-smoke] FAIL: no message returned from pull"
  echo "[pubsub-smoke] full response: ${PULL_RESPONSE}"
  exit 1
fi
echo "[pubsub-smoke] message pulled (ackId=${ACK_ID})"

# 5. Ack so the round-trip is complete.
curl -sf -X POST "${BASE}/subscriptions/${SUB}:acknowledge" \
  -H "Content-Type: application/json" \
  -d "{\"ackIds\":[\"${ACK_ID}\"]}" >/dev/null
echo "[pubsub-smoke] message acked"

echo "[pubsub-smoke] OK"
