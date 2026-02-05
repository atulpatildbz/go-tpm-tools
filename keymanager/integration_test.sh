#!/bin/bash
# Integration test script for WSD key custody core.
# Builds the Rust library (with BoringSSL) and runs Go integration tests
# inside a Linux container via podman.
#
# Usage: ./keymanager/integration_test.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "=== Building and testing in Linux container ==="
podman run --rm \
  -v "${REPO_ROOT}:/src:Z" \
  -w /src \
  docker.io/library/rust:latest \
  bash -c '
    set -euo pipefail

    echo "--- Installing dependencies ---"
    apt-get update -qq
    apt-get install -y -qq golang cmake ninja-build libclang-dev > /dev/null 2>&1
    cargo install bindgen-cli 2>/dev/null

    echo "--- Building BoringSSL ---"
    cd /src/boringssl
    mkdir -p build && cd build
    cmake -GNinja -DRUST_BINDINGS=1 -DCMAKE_POSITION_INDEPENDENT_CODE=ON .. > /dev/null 2>&1
    ninja > /dev/null 2>&1
    export BORINGSSL_BUILD_DIR=/src/boringssl/build

    echo "--- Building Rust workspace ---"
    cd /src/keymanager
    cargo build --release --workspace 2>&1

    echo "--- Running Go integration tests ---"
    cd /src/keymanager
    CGO_ENABLED=1 \
    CGO_LDFLAGS="-L/src/boringssl/build/crypto -lcrypto" \
    go test -tags integration -v ./workload_service/key_custody_core/
  '

echo "=== All integration tests passed ==="
