#!/bin/bash
set -eux

# Use the protoc-gen-go and protoc-gen-go-grpc binaries from the current
# directory, not the ones in $PATH.
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
TOOLS_DIR="$(dirname "$DIR")/tools"
export PATH="$TOOLS_DIR:$PATH"

protoc \
  --go_out=./ \
  --go-grpc_out=./ \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  --go_opt=Msnapshot_program.proto=github.com/DataExMachina-dev/side-eye-go/internal/snapshotpb \
  --go-grpc_opt=Msnapshot_program.proto=github.com/DataExMachina-dev/side-eye-go/internal/snapshotpb \
  -I . \
  snapshot_program.proto
