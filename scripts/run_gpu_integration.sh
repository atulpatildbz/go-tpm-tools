#!/bin/bash
# Script to run GPU integration tests on a remote CC-enabled VM.

set -e

if [ -z "$1" ]; then
    echo "Usage: $0 <vm-hostname-or-ip>"
    echo "Example: $0 patilatul-a3-vm"
    exit 1
fi

REMOTE_VM=$1
TEST_BINARY="gpu_integration_test"
TARGET_DIR="/tmp"

echo "🔨 Compiling integration test binary for linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CGO_LDFLAGS="-Wl,-z,lazy" go test -c -tags=integration_gpu ./launcher/internal/gpu -o ${TEST_BINARY}

echo "📤 Copying binary to ${REMOTE_VM}:${TARGET_DIR}..."
gcloud compute scp ${TEST_BINARY} ${REMOTE_VM}:${TARGET_DIR}/ --project=patilatul-project

echo "🚀 Running integration tests on ${REMOTE_VM}..."
gcloud compute ssh ${REMOTE_VM} --project=patilatul-project --command="${TARGET_DIR}/${TEST_BINARY} -test.v"

echo "🧹 Cleaning up local binary..."
rm ${TEST_BINARY}

echo "✅ Done!"
