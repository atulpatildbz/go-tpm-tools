#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
KEYMANAGER_DIR="$(cd -- "${SCRIPT_DIR}/.." && pwd)"

BORINGSSL_SOURCE_DIR="${BORINGSSL_SOURCE_DIR:-${KEYMANAGER_DIR}/boringssl}"
BORINGSSL_BUILD_DIR="${BORINGSSL_BUILD_DIR:-${BORINGSSL_SOURCE_DIR}/build}"
RUST_TARGET="${RUST_TARGET:-$(rustc -vV | sed -n 's/^host: //p')}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required tool '$1' not found in PATH" >&2
    exit 1
  fi
}

require_cmd rustc
require_cmd cargo
require_cmd cmake
require_cmd bindgen

if [[ ! -f "${BORINGSSL_SOURCE_DIR}/CMakeLists.txt" ]]; then
  echo "error: BoringSSL source tree not found at '${BORINGSSL_SOURCE_DIR}'" >&2
  echo "hint: run 'git submodule update --init --recursive keymanager/boringssl'" >&2
  exit 1
fi

cmake_args=(
  -S "${BORINGSSL_SOURCE_DIR}"
  -B "${BORINGSSL_BUILD_DIR}"
  -DRUST_BINDINGS="${RUST_TARGET}"
  -DCMAKE_POSITION_INDEPENDENT_CODE=ON
  -DBUILD_TESTING=OFF
)

if [[ -n "${BORINGSSL_CMAKE_GENERATOR:-}" ]]; then
  cmake_args+=( -G "${BORINGSSL_CMAKE_GENERATOR}" )
elif [[ ! -f "${BORINGSSL_BUILD_DIR}/CMakeCache.txt" ]] && command -v ninja >/dev/null 2>&1; then
  cmake_args+=( -GNinja )
fi

cmake "${cmake_args[@]}"

build_args=(
  --build "${BORINGSSL_BUILD_DIR}"
  --target bssl_sys
)

if [[ -n "${NUM_JOBS:-}" ]]; then
  build_args+=( --parallel "${NUM_JOBS}" )
fi

cmake "${build_args[@]}"

export BORINGSSL_SOURCE_DIR
export BORINGSSL_BUILD_DIR

cd "${KEYMANAGER_DIR}"

if [[ "$#" -eq 0 ]]; then
  cargo build --workspace
else
  cargo "$@"
fi
