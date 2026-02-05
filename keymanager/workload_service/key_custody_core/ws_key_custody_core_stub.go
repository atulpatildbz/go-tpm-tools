//go:build !cgo
// +build !cgo

package key_custody_core

import (
	"fmt"

	"github.com/google/uuid"
)

// GenerateBindingKeypair is a stub when CGO is not enabled.
func GenerateBindingKeypair() (uuid.UUID, error) {
	return uuid.Nil, fmt.Errorf("ws_key_custody_core: CGO not enabled, cannot generate binding keypair")
}

// GenerateKEMKeypair is a stub when CGO is not enabled.
func GenerateKEMKeypair(bindingPK []byte) (uuid.UUID, error) {
	return uuid.Nil, fmt.Errorf("ws_key_custody_core: CGO not enabled, cannot generate KEM keypair")
}
