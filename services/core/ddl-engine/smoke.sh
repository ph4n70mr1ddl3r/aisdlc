#!/bin/sh
# End-to-end M1 smoke test: metadata-api defines an entity, ddl-engine creates the table.
set -eu
META=http://metadata-api:8000
DDL=http://ddl-engine:8000

SLUG=acme$$_$(date +%s)
echo "=== 1. create tenant (slug=$SLUG) ==="
TENANT=$(curl -s -X POST "$META/v1/tenants" -H 'Content-Type: application/json' \
  -d "{\"name\":\"Acme\",\"slug\":\"$SLUG\"}")
echo "$TENANT" | jq -c .
TID=$(echo "$TENANT" | jq -r .id)

APPSLUG=assets$$_$(date +%s)
echo "=== 2. create application ==="
APP=$(curl -s -X POST "$META/v1/applications" -H 'Content-Type: application/json' \
  -H "X-Tenant-ID: $TID" -d "{\"name\":\"Assets\",\"slug\":\"$APPSLUG\"}")
echo "$APP" | jq -c .
APPID=$(echo "$APP" | jq -r .id)

TABLE=assets$$_$(date +%s)
echo "=== 3. create entity '$TABLE' ==="
ENT=$(curl -s -X POST "$META/v1/entities" -H 'Content-Type: application/json' \
  -d "{\"app_id\":\"$APPID\",\"name\":\"Asset\",\"table_name\":\"$TABLE\",\"label_singular\":\"Asset\",\"label_plural\":\"Assets\"}")
echo "$ENT" | jq -c .
EID=$(echo "$ENT" | jq -r .id)

echo "=== 4. create fields ==="
curl -s -X POST "$META/v1/fields" -H 'Content-Type: application/json' \
  -d "{\"entity_id\":\"$EID\",\"name\":\"serial\",\"type\":\"text\",\"config\":{\"required\":true,\"unique\":true}}" | jq -c .
curl -s -X POST "$META/v1/fields" -H 'Content-Type: application/json' \
  -d "{\"entity_id\":\"$EID\",\"name\":\"qty\",\"type\":\"number\",\"config\":{\"required\":true}}" | jq -c .
curl -s -X POST "$META/v1/fields" -H 'Content-Type: application/json' \
  -d "{\"entity_id\":\"$EID\",\"name\":\"active\",\"type\":\"bool\"}" | jq -c .
curl -s -X POST "$META/v1/fields" -H 'Content-Type: application/json' \
  -d "{\"entity_id\":\"$EID\",\"name\":\"price\",\"type\":\"money\"}" | jq -c .
curl -s -X POST "$META/v1/fields" -H 'Content-Type: application/json' \
  -d "{\"entity_id\":\"$EID\",\"name\":\"score\",\"type\":\"computed\"}" | jq -c .

echo "=== 5. dry-run DDL preview ==="
curl -s "$DDL/v1/entities/$EID/ddl" | jq -c '.statements[] | {kind, detail, sql}'

echo "=== 6. apply (table creation) ==="
curl -s -X POST "$DDL/v1/entities/$EID/apply" | jq -c .

echo "=== 7. re-apply (idempotency: should apply 0) ==="
curl -s -X POST "$DDL/v1/entities/$EID/apply" | jq -c '{table, applied}'

echo "=== 8. tenant-wide apply ==="
curl -s -X POST "$DDL/v1/apply" -H "X-Tenant-ID: $TID" | jq -c '{count}'

echo "=== 9. migration ledger ==="
curl -s "$DDL/v1/migrations" | jq -c '.data[] | {kind, table_name, detail}'

echo "TENANT_ID=$TID ENTITY_ID=$EID"
