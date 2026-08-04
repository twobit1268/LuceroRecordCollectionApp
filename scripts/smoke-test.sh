#!/usr/bin/env bash
# End-to-end smoke test against a running `docker compose up` stack.
# Proves the actual distributed flow works — not just that each service
# compiles in isolation — by exercising sync validation, async Pub/Sub
# publish/consume, and cross-service genre enrichment for real over HTTP.
#
# Usage: docker compose up --build -d && ./scripts/smoke-test.sh
set -euo pipefail

CATALOG_URL="http://localhost:8081"
COLLECTION_URL="http://localhost:8082"
ACTIVITY_URL="http://localhost:8083"

pass() { echo "PASS: $1"; }
fail() { echo "FAIL: $1" >&2; exit 1; }

json_field() { python3 -c "import json,sys; print(json.load(sys.stdin)$1)"; }

wait_for() {
  local name="$1" url="$2"
  for i in $(seq 1 30); do
    if curl -sf "$url/healthz" > /dev/null 2>&1; then
      pass "$name is ready"
      return 0
    fi
    sleep 2
  done
  fail "$name never became ready at $url/healthz"
}

echo "=== Waiting for services to be ready ==="
wait_for "catalog-service" "$CATALOG_URL"
wait_for "collection-service" "$COLLECTION_URL"
wait_for "activity-service" "$ACTIVITY_URL"

echo
echo "=== 1. Create a record in catalog-service ==="
CREATE_STATUS=$(curl -s -o /tmp/create_response.json -w "%{http_code}" \
  -X POST "$CATALOG_URL/records" -H 'Content-Type: application/json' \
  -d '{"title":"Kind of Blue","artist":"Miles Davis","genre":"Jazz","year":1959}')
[ "$CREATE_STATUS" = "201" ] || fail "expected 201 creating record, got $CREATE_STATUS"
RECORD_ID=$(json_field "['id']" < /tmp/create_response.json)
[ -n "$RECORD_ID" ] || fail "no id returned from record creation"
pass "created record $RECORD_ID"

echo
echo "=== 2. Reject a nonexistent recordId ==="
BAD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" \
  -X POST "$COLLECTION_URL/collections/smoke-cust/records" \
  -H 'Content-Type: application/json' -d '{"recordId":"does-not-exist"}')
[ "$BAD_STATUS" = "400" ] || fail "expected 400 for nonexistent recordId, got $BAD_STATUS"
pass "correctly rejected nonexistent recordId (400)"

echo
echo "=== 3. Add the real record to a customer's collection ==="
ADD_STATUS=$(curl -s -o /tmp/add_response.json -w "%{http_code}" \
  -X POST "$COLLECTION_URL/collections/smoke-cust/records" \
  -H 'Content-Type: application/json' -d "{\"recordId\":\"$RECORD_ID\"}")
[ "$ADD_STATUS" = "201" ] || fail "expected 201 adding record to collection, got $ADD_STATUS"
ENTRY_ID=$(json_field "['id']" < /tmp/add_response.json)
pass "added to collection, entry $ENTRY_ID"

echo
echo "=== 4. Confirm it's listed in the customer's collection ==="
LIST_COUNT=$(curl -s "$COLLECTION_URL/collections/smoke-cust/records" | python3 -c "import json,sys; print(len(json.load(sys.stdin)))")
[ "$LIST_COUNT" = "1" ] || fail "expected 1 entry in collection, got $LIST_COUNT"
pass "collection lists 1 entry"

echo
echo "=== 5. Wait for activity-service to consume the Pub/Sub event and enrich it ==="
GENRE=""
for i in $(seq 1 15); do
  GENRE=$(curl -s "$ACTIVITY_URL/activity/feed" | python3 -c "
import json,sys
entries = json.load(sys.stdin)
matches = [e['genre'] for e in entries if e['recordId'] == '$RECORD_ID']
print(matches[0] if matches else '')
" 2>/dev/null || echo "")
  [ -n "$GENRE" ] && break
  sleep 2
done
[ "$GENRE" = "Jazz" ] || fail "expected activity feed to show genre 'Jazz' enriched via catalog-service, got '$GENRE'"
pass "activity-service consumed the event and enriched it with genre=Jazz"

echo
echo "=== 6. Genre counts reflect it ==="
JAZZ_COUNT=$(curl -s "$ACTIVITY_URL/activity/genres" | python3 -c "
import json,sys
counts = {c['genre']: c['count'] for c in json.load(sys.stdin)}
print(counts.get('Jazz', 0))
")
[ "$JAZZ_COUNT" -ge "1" ] || fail "expected Jazz count >= 1, got $JAZZ_COUNT"
pass "genre counts include Jazz"

echo
echo "=== 7. Delete the entry, then confirm 404 on redelete ==="
DELETE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$COLLECTION_URL/collections/smoke-cust/records/$ENTRY_ID")
[ "$DELETE_STATUS" = "204" ] || fail "expected 204 deleting entry, got $DELETE_STATUS"
REDELETE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$COLLECTION_URL/collections/smoke-cust/records/$ENTRY_ID")
[ "$REDELETE_STATUS" = "404" ] || fail "expected 404 on redelete, got $REDELETE_STATUS"
pass "delete + idempotent 404 on redelete both correct"

echo
echo "=== All smoke tests passed ==="
