#!/usr/bin/env bash
# Regenerate internal/gen from api/proto (requires protoc + protoc-gen-go + protoc-gen-go-grpc).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
export PATH="${PATH}:$(go env GOPATH)/bin"
protoc -I "$ROOT/api/proto" \
  --go_out="$ROOT" --go_opt=module=github.com/BigRedS/coralogix-unused-metrics-finder \
  --go-grpc_out="$ROOT" --go-grpc_opt=module=github.com/BigRedS/coralogix-unused-metrics-finder \
  "$ROOT/api/proto/com/coralogix/metrics/metric-usages.proto" \
  "$ROOT/api/proto/com/coralogix/metrics/common.proto" \
  "$ROOT/api/proto/com/coralogix/metrics/priority.proto" \
  "$ROOT/api/proto/com/coralogixapis/aaa/organisations/v2/types.proto" \
  "$ROOT/api/proto/com/coralogixapis/aaa/organisations/v2/team_service.proto"
