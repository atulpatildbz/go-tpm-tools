#!/bin/bash
# Integration test: builds Rust wsd_kcc, then runs Go CGO integration tests.
# Usage: ./kcc/integration_test.sh
# Runs inside Docker (Linux) since Vault uses memfd_secret (Linux-only).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"

echo "=== Building and running integration tests in Docker ==="

CONTAINER_RT="${CONTAINER_RT:-podman}"
"${CONTAINER_RT}" run --rm \
  -v "${REPO_DIR}:/src/go-tpm-tools" \
  -w /src/go-tpm-tools \
  rust:latest \
  bash -c '
set -euo pipefail

echo "--- Installing build dependencies ---"
apt-get update -qq && apt-get install -y -qq cmake ninja-build golang-go git pkg-config clang libclang-dev > /dev/null 2>&1
cargo install bindgen-cli --quiet 2>&1
echo "Go version: $(go version)"
echo "Rust version: $(rustc --version)"

echo "--- Cloning and building BoringSSL ---"
if [ ! -d /src/boringssl ]; then
  git clone --depth 1 https://boringssl.googlesource.com/boringssl /src/boringssl
fi
cd /src/boringssl
if [ ! -f build/crypto/libcrypto.a ]; then
  mkdir -p build && cd build
  cmake -GNinja -DCMAKE_BUILD_TYPE=Release -DRUST_BINDINGS="$(rustc -vV | sed -n "s|host: ||p")" ..
  ninja
fi

echo "--- Building Rust workspace (kcc) ---"
cd /src/go-tpm-tools/kcc
BORINGSSL_BUILD_DIR=/src/boringssl/build cargo build --release --workspace 2>&1

echo "--- Verifying libwsd_kcc.a ---"
ls -la /src/go-tpm-tools/kcc/target/release/libwsd_kcc.a

echo "--- Running Go integration tests ---"
cd /src/go-tpm-tools
CGO_ENABLED=1 CGO_LDFLAGS="-L/src/boringssl/build/crypto -lcrypto" \
  go test -tags integration -v ./kcc/wsd_kcc/ -run TestIntegration
'
