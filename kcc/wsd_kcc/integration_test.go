//go:build cgo && integration

package wsd_kcc

import (
	"testing"

	"github.com/google/uuid"
)

func TestIntegrationGenerateBindingKeypair(t *testing.T) {
	handle, err := GenerateBindingKeypair()
	if err != nil {
		t.Fatalf("GenerateBindingKeypair() failed: %v", err)
	}
	if handle == uuid.Nil {
		t.Fatal("expected non-nil UUID handle")
	}
	t.Logf("Generated binding keypair with handle: %s", handle)
}

func TestIntegrationGenerateKEMKeypair(t *testing.T) {
	// Use a dummy 32-byte X25519 public key.
	bindingPK := make([]byte, 32)
	for i := range bindingPK {
		bindingPK[i] = byte(i + 1)
	}

	handle, err := GenerateKEMKeypair(bindingPK)
	if err != nil {
		t.Fatalf("GenerateKEMKeypair() failed: %v", err)
	}
	if handle == uuid.Nil {
		t.Fatal("expected non-nil UUID handle")
	}
	t.Logf("Generated KEM keypair with handle: %s", handle)
}

func TestIntegrationGenerateKEMKeypairEmptyPK(t *testing.T) {
	_, err := GenerateKEMKeypair([]byte{})
	if err == nil {
		t.Fatal("expected error for empty binding public key")
	}
}
