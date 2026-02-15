package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/go-tpm-tools/keymanager/key_protection_service"
	kpskcc "github.com/google/go-tpm-tools/keymanager/key_protection_service/key_custody_core"
	"github.com/google/go-tpm-tools/keymanager/workload_service"
	"github.com/google/uuid"
)

type fakeBindingKeyGenerator struct{}

func (f *fakeBindingKeyGenerator) GenerateBindingKeypair(lifespanSecs uint64) (uuid.UUID, []byte, error) {
	// 32-byte fake public key (simulating X25519)
	pubKey := make([]byte, 32)
	copy(pubKey, []byte("fake-binding-key-32-bytes-long!"))
	return uuid.New(), pubKey, nil
}

func main() {
	// 1. Setup KPS Service
	log.Println("Initializing KPS Service...")
	kps := key_protection_service.NewService(
		kpskcc.GenerateKEMKeypair,
		kpskcc.EnumerateKEMKeys,
	)

	// 2. Setup WSD Server
	log.Println("Initializing WSD Server...")
	bindingGen := &fakeBindingKeyGenerator{}
	server := workload_service.NewServer(bindingGen, kps, kps)

	// 3. Start Server on Unix Socket
	tmpDir := os.TempDir()
	socketPath := filepath.Join(tmpDir, fmt.Sprintf("wsd_test_%d.sock", time.Now().UnixNano()))
	// Ensure socket doesn't exist
	os.Remove(socketPath)
	defer os.Remove(socketPath)

	log.Printf("Starting server on %s", socketPath)
	go func() {
		if err := server.Serve(socketPath); err != nil && err != http.ErrServerClosed {
			log.Printf("Server failed: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(1 * time.Second)

	// 4. Create HTTP Client with Unix Socket transport
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	baseURL := "http://localhost" // Host is ignored for unix socket dialer

	// 5. Enumerate Keys (Expect Empty)
	log.Println("Testing: GET /v1/keys (Initial State)")
	if err := enumerateKeys(client, baseURL, 0); err != nil {
		log.Fatalf("Initial enumeration failed: %v", err)
	}

	// 6. Generate Key
	log.Println("Testing: POST /v1/keys:generate_kem")
	if err := generateKey(client, baseURL); err != nil {
		log.Fatalf("Key generation failed: %v", err)
	}

	// 7. Enumerate Keys (Expect 1 Key)
	log.Println("Testing: GET /v1/keys (After Generation)")
	if err := enumerateKeys(client, baseURL, 1); err != nil {
		log.Fatalf("Final enumeration failed: %v", err)
	}

	log.Println("Manual testing completed successfully!")
}

func enumerateKeys(client *http.Client, baseURL string, expectedCount int) error {
	resp, err := client.Get(baseURL + "/v1/keys")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	log.Printf("Enumerate Response: %s", string(body))

	var result workload_service.EnumerateKeysResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	if len(result.KeyInfos) != expectedCount {
		return fmt.Errorf("expected %d keys, got %d", expectedCount, len(result.KeyInfos))
	}

	return nil
}

func generateKey(client *http.Client, baseURL string) error {
	reqBody := workload_service.GenerateKemRequest{
		Algorithm:              workload_service.KemAlgorithmDHKEMX25519HKDFSHA256,
		Lifespan:               workload_service.ProtoDuration{Seconds: 3600},
		KeyProtectionMechanism: workload_service.KeyProtectionMechanismVM,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := client.Post(baseURL+"/v1/keys:generate_kem", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d, body: %s", resp.StatusCode, body)
	}

	log.Printf("Generated Key: %s", string(body))
	return nil
}
