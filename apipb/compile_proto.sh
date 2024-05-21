#!/bin/bash
set -eux

# Use the protoc-gen-go and protoc-gen-go-grpc binaries from the current
# directory, not the ones in $PATH.
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
export PATH="$DIR:$PATH"

protoc \
  --go_out=./ \
  --go-grpc_out=./ \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  --go_opt=Mapi.proto=stackviz/api/apipb \
  --go-grpc_opt=Mapi.proto=stackviz/api/apipb \
  api.proto
