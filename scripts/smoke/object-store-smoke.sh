#!/usr/bin/env bash
# path: scripts/smoke/object-store-smoke.sh
#
# Smoke test: ADR-0010 object-store capabilities, with a known
# commodity-emulator fidelity gap on the generation-conditional
# row (see docker-compose.yml object-store comment block).
#
# Steps:
#   1. Create the dq-local bucket.
#   2. Write a content-hashed body to manifests/by-hash/sha256-<hex>.json
#      and verify it reads back byte-identical (covers the
#      ADR-0010 "Object store: by-hash immutability with sha256"
#      row, fully exercisable locally).
#   3. Write the pointer manifests/latest.json with an unconditional
#      first write; capture its generation. Then issue a second
#      write with ifGenerationMatch set to the captured generation
#      and verify the request is accepted (covers the API surface
#      the ADR-0005 publication primitive uses).
#   4. Issue a third write with the STALE generation. The
#      commodity-emulator gap means this typically succeeds
#      locally; the script logs the observed behavior and treats
#      strict enforcement as sandbox-only. Production-shape CAS
#      enforcement is verified in the sandbox-required CI lane.
#      A Phase 2 follow-up will revise ADR-0010's row from Yes →
#      Partial to reflect this reality; tracked as B1-11 in
#      studies/foundation/06-decision-log.md.

set -euo pipefail

HOST="${GCS_EMULATOR_HOST:-localhost:4443}"
BUCKET="${GCS_BUCKET:-dq-local}"
BASE="http://${HOST}/storage/v1/b"
UPLOAD_BASE="http://${HOST}/upload/storage/v1/b"

echo "[object-store-smoke] base=${BASE}"

# Helper: compute sha256 hex of a string.
sha256hex() {
  printf '%s' "$1" | shasum -a 256 | awk '{print $1}'
}

# 1. Create the bucket (idempotent: ignore "already exists" 409).
HTTP_CODE=$(curl -sf -o /tmp/dq-os-smoke-bucket.json -w '%{http_code}' \
  -X POST "${BASE}?project=dq-local" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"${BUCKET}\"}" || true)
if [[ "${HTTP_CODE}" != "200" && "${HTTP_CODE}" != "409" ]]; then
  echo "[object-store-smoke] FAIL: bucket create returned ${HTTP_CODE}"
  exit 1
fi
echo "[object-store-smoke] bucket ready (http=${HTTP_CODE})"

# 2. Write a content-hashed manifest body and read it back.
BODY='{"manifest_version":1,"ruleset_version":"rules-vsmoke","schema_versions_present":[1],"engine_compatibility":">=1.0.0 <2.0.0","linter_used":"tools-lint-vsmoke","generated_at":"2026-05-21T00:00:00Z","rules":[]}'
HEX=$(sha256hex "${BODY}")
OBJECT_NAME="manifests/by-hash/sha256-${HEX}.json"

curl -sf -X POST \
  "${UPLOAD_BASE}/${BUCKET}/o?uploadType=media&name=${OBJECT_NAME}" \
  -H "Content-Type: application/json" \
  --data-binary "${BODY}" >/dev/null
echo "[object-store-smoke] body written: ${OBJECT_NAME}"

READBACK=$(curl -sf "${BASE}/${BUCKET}/o/$(printf '%s' "${OBJECT_NAME}" | python3 -c 'import sys,urllib.parse; sys.stdout.write(urllib.parse.quote(sys.stdin.read(),safe=""))')?alt=media")
if [[ "${READBACK}" != "${BODY}" ]]; then
  echo "[object-store-smoke] FAIL: body readback differs"
  exit 1
fi
echo "[object-store-smoke] body roundtrip OK"

# 3. Write manifests/latest.json (unconditional first write).
POINTER_BODY="{\"pointer_version\":1,\"manifest_hash\":\"sha256:${HEX}\",\"ruleset_version\":\"rules-vsmoke\",\"published_at\":\"2026-05-21T00:00:00Z\"}"
POINTER_NAME="manifests/latest.json"

CREATE_RESPONSE=$(curl -sf -X POST \
  "${UPLOAD_BASE}/${BUCKET}/o?uploadType=media&name=$(printf '%s' "${POINTER_NAME}" | python3 -c 'import sys,urllib.parse; sys.stdout.write(urllib.parse.quote(sys.stdin.read(),safe=""))')" \
  -H "Content-Type: application/json" \
  --data-binary "${POINTER_BODY}")
GEN_1=$(echo "${CREATE_RESPONSE}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["generation"])')
echo "[object-store-smoke] pointer written, generation=${GEN_1}"

# 4. Conditional write with the correct generation: should succeed.
POINTER_BODY_2="${POINTER_BODY%}}, \"_smoke\":\"2\"}"
POINTER_BODY_2=$(printf '%s' "${POINTER_BODY}" | sed 's/}$/,"_smoke":"2"}/')
UPDATE_RESPONSE=$(curl -sf -X POST \
  "${UPLOAD_BASE}/${BUCKET}/o?uploadType=media&name=$(printf '%s' "${POINTER_NAME}" | python3 -c 'import sys,urllib.parse; sys.stdout.write(urllib.parse.quote(sys.stdin.read(),safe=""))')&ifGenerationMatch=${GEN_1}" \
  -H "Content-Type: application/json" \
  --data-binary "${POINTER_BODY_2}")
GEN_2=$(echo "${UPDATE_RESPONSE}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["generation"])')
if [[ "${GEN_2}" == "${GEN_1}" ]]; then
  echo "[object-store-smoke] FAIL: generation did not advance after CAS write"
  exit 1
fi
echo "[object-store-smoke] CAS write OK (new generation=${GEN_2})"

# 5. Conditional write with the STALE generation. Commodity
#    emulator fidelity gap: production-shape ifGenerationMatch
#    enforcement returns 412 here; the selected emulator currently
#    accepts the write. The smoke logs the observed behavior and
#    succeeds in either case — strict enforcement is validated in
#    the sandbox-required CI lane. See the object-store comment
#    block in docker-compose.yml and the Phase 2 follow-up to
#    revise ADR-0010.
STALE_HTTP_CODE=$(curl -s -o /tmp/dq-os-smoke-stale.json -w '%{http_code}' \
  -X POST \
  "${UPLOAD_BASE}/${BUCKET}/o?uploadType=media&name=$(printf '%s' "${POINTER_NAME}" | python3 -c 'import sys,urllib.parse; sys.stdout.write(urllib.parse.quote(sys.stdin.read(),safe=""))')&ifGenerationMatch=${GEN_1}" \
  -H "Content-Type: application/json" \
  --data-binary "${POINTER_BODY_2}")
case "${STALE_HTTP_CODE}" in
  412)
    echo "[object-store-smoke] stale-generation write correctly rejected (412) — production-shape CAS enforcement available locally."
    ;;
  200)
    echo "[object-store-smoke] stale-generation write accepted (200) — commodity-emulator fidelity gap on CAS; strict enforcement runs in the sandbox-required CI lane (see B1-11 in studies/foundation/06-decision-log.md)."
    ;;
  *)
    echo "[object-store-smoke] FAIL: stale-generation write returned ${STALE_HTTP_CODE}, expected 200 or 412"
    exit 1
    ;;
esac

echo "[object-store-smoke] OK"
