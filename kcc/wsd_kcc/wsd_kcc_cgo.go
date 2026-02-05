//go:build cgo
// +build cgo

// Package wsd_kcc provides Go bindings to the WSD Key Custody Core (Rust FFI).
package wsd_kcc

/*
#cgo LDFLAGS: -L${SRCDIR}/../target/release -lwsd_kcc
#cgo LDFLAGS: -lpthread -ldl -lm
#include "include/wsd_kcc.h"
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
