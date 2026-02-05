//go:build integration
// +build integration

package key_custody_core

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
	t.Logf("binding key handle: %s", handle)
}

func TestIntegrationGenerateKEMKeypair(t *testing.T) {
	bindingPK := make([]byte, 32)
	for i := range bindingPK {
		bindingPK[i] = byte(i)
	}

	handle, err := GenerateKEMKeypair(bindingPK)
	if err != nil {
		t.Fatalf("GenerateKEMKeypair() failed: %v", err)
	}
	if handle == uuid.Nil {
		t.Fatal("expected non-nil UUID handle")
	}
	t.Logf("KEM key handle: %s", handle)
}

func TestIntegrationGenerateKEMKeypairEmptyPK(t *testing.T) {
	_, err := GenerateKEMKeypair(nil)
	if err == nil {
		t.Fatal("expected error for empty binding PK, got nil")
	}
	t.Logf("got expected error: %v", err)
}
