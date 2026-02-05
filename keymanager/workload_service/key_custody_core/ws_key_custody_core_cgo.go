//go:build cgo
// +build cgo

// Package key_custody_core provides Go bindings to the workload service
// Key Custody Core (Rust FFI).
package key_custody_core

/*
#cgo LDFLAGS: -L${SRCDIR}/../../../keymanager/target/release -lws_key_custody_core
#cgo LDFLAGS: -lpthread -ldl -lm
#include "include/ws_key_custody_core.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/google/uuid"
)

// GenerateBindingKeypair generates a new HPKE binding keypair and returns the
// key handle as a UUID.
func GenerateBindingKeypair() (uuid.UUID, error) {
	var handleBuf [16]byte
	rc := C.key_manager_generate_binding_keypair((*C.uint8_t)(unsafe.Pointer(&handleBuf[0])))
	if rc != 0 {
		return uuid.Nil, fmt.Errorf("key_manager_generate_binding_keypair failed with code %d", rc)
	}
	handle, err := uuid.FromBytes(handleBuf[:])
	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing key handle UUID: %w", err)
	}
	return handle, nil
}

// GenerateKEMKeypair generates a new KEM keypair associated with the given
// binding public key and returns the key handle as a UUID.
func GenerateKEMKeypair(bindingPK []byte) (uuid.UUID, error) {
	if len(bindingPK) == 0 {
		return uuid.Nil, fmt.Errorf("binding public key must not be empty")
	}
	var handleBuf [16]byte
	rc := C.key_manager_generate_kem_keypair(
		(*C.uint8_t)(unsafe.Pointer(&bindingPK[0])),
		C.size_t(len(bindingPK)),
		(*C.uint8_t)(unsafe.Pointer(&handleBuf[0])),
	)
	if rc != 0 {
		return uuid.Nil, fmt.Errorf("key_manager_generate_kem_keypair failed with code %d", rc)
	}
	handle, err := uuid.FromBytes(handleBuf[:])
	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing key handle UUID: %w", err)
	}
	return handle, nil
}
