package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	kps "github.com/google/go-tpm-tools/keymanager/key_protection_service"
	kpskcc "github.com/google/go-tpm-tools/keymanager/key_protection_service/key_custody_core"
	algorithms "github.com/google/go-tpm-tools/keymanager/km_common/proto"
	workload_service "github.com/google/go-tpm-tools/keymanager/workload_service"
	wskcc "github.com/google/go-tpm-tools/keymanager/workload_service/key_custody_core"
)

// realWorkloadService wraps the actual WSD KCC FFI.
type realWorkloadService struct{}

func (r *realWorkloadService) GenerateBindingKeypair(algo *algorithms.HpkeAlgorithm, lifespanSecs uint64) (uuid.UUID, []byte, error) {
	return wskcc.GenerateBindingKeypair(algo, lifespanSecs)
}

func (r *realWorkloadService) Open(bindingUUID uuid.UUID, enc, ciphertext, aad []byte) ([]byte, error) {
	return wskcc.Open(bindingUUID, enc, ciphertext, aad)
}

func main() {
	// Create server.log file
	logFile, err := os.OpenFile("server.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		log.Fatalf("Failed to open server.log: %v", err)
	}
	defer func() {
		_ = logFile.Close()
	}()

	// Write logs to both standard error and server.log
	mw := io.MultiWriter(os.Stderr, logFile)
	log.SetOutput(mw)

	log.Println("Initializing WSD server components...")

	// Wire up real FFI calls for generation and decap/open to test E2E functionality.
	kpsSvc := kps.NewService(kpskcc.GenerateKEMKeypair, kpskcc.DecapAndSeal)

	socketPath := "/tmp/wsd.sock"
	if err := os.RemoveAll(socketPath); err != nil {
		log.Printf("Failed to remove old socket (ignored): %v", err)
	}

	srv, err := workload_service.NewServer(
		kpsSvc,
		&realWorkloadService{},
		socketPath,
	)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := srv.Serve(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for socket to be created
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

	log.Println("Server ready. You can now use the following curl commands to test:")
	log.Printf("curl --unix-socket %s http://unix/v1/capabilities", socketPath)
	log.Printf("curl --unix-socket %s http://unix/v1/keys:generate_kem -H \"Content-Type: application/json\" -d '{\"algorithm\":\"DHKEM_X25519_HKDF_SHA256\", \"key_protection_mechanism\":\"KEY_PROTECTION_VM\", \"lifespan\":3600}'", socketPath)
	log.Println("To test unsupported algorithm:")
	log.Printf("curl -v --unix-socket %s http://unix/v1/keys:generate_kem -H \"Content-Type: application/json\" -d '{\"algorithm\":\"KEM_ALGORITHM_UNSPECIFIED\", \"key_protection_mechanism\":\"KEY_PROTECTION_VM\", \"lifespan\":3600}'", socketPath)
	log.Println("To test destroy (replace HANDLE with actual handle):")
	log.Printf("curl --unix-socket %s http://unix/v1/keys:destroy -H \"Content-Type: application/json\" -d '{\"key_handle\": {\"handle\": \"HANDLE\"}}'", socketPath)

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
