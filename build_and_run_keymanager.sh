#!/bin/bash
set -e

WORKLOAD_DIR="launcher/image/testworkloads/keymanager"
IMAGE_NAME="keymanager-test-workload"
SOCKET_PATH="/tmp/kmaserver.sock"

echo "=== Building KeyManager Rust library ==="
(cd keymanager && cargo build --release)

echo "=== Building KeyManager test server ==="
go build -o test_server test_server.go

echo "=== Starting KeyManager test server ==="
./test_server &
SERVER_PID=$!

# Give the server a moment to start and create the socket
sleep 2

echo "=== Building the keymanager workload container ==="
docker build -t "$IMAGE_NAME" "$WORKLOAD_DIR"

echo "=== Running the keymanager workload container ==="
# We mount the local socket to the path expected by the workload inside the container.
docker run --rm -v "$SOCKET_PATH:/run/container_launcher/kmaserver.sock" "$IMAGE_NAME"

echo "=== Cleaning up ==="
kill $SERVER_PID
wait $SERVER_PID 2>/dev/null || true
echo "Done."
