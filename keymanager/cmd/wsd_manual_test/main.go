package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	kps "github.com/google/go-tpm-tools/keymanager/key_protection_service"
	kpskcc "github.com/google/go-tpm-tools/keymanager/key_protection_service/key_custody_core"
	"github.com/google/go-tpm-tools/keymanager/workload_service"
	wskcc "github.com/google/go-tpm-tools/keymanager/workload_service/key_custody_core"
)

// realBindingKeyGen wraps the actual WSD KCC FFI.
type realBindingKeyGen struct{}

func (r *realBindingKeyGen) GenerateBindingKeypair(lifespanSecs uint64) (uuid.UUID, []byte, error) {
	return wskcc.GenerateBindingKeypair(lifespanSecs)
}

func main() {
	log.Println("Initializing WSD server components...")

	// Wire up real FFI calls: WSD KCC for binding, KPS KCC (via KPS KOL) for KEM.
	kpsSvc := kps.NewService(kpskcc.GenerateKEMKeypair)
	srv := workload_service.NewServer(&realBindingKeyGen{}, kpsSvc)

	socketPath := "/tmp/wsd.sock"
	// Ensure the socket does not already exist
	_ = os.Remove(socketPath)

	log.Printf("Starting server on %s", socketPath)

	go func() {
		if err := srv.Serve(socketPath); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for socket
	ready := false
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			ready = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !ready {
		log.Fatal("Timed out waiting for socket")
	}

	log.Println("Server ready. You can now use the following curl command to test:")
	log.Printf("curl --unix-socket %s http://unix/v1/keys:generate_kem -H \"Content-Type: application/json\" -d '{\"algorithm\":\"DHKEM_X25519_HKDF_SHA256\", \"key_protection_mechanism\":\"KEY_PROTECTION_VM\", \"lifespan\":\"3600s\"}'", socketPath)

	// Wait for interrupt signal to gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
