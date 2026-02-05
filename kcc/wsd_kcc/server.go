// Package wsd_kcc implements the Workload Service Daemon (WSD) server
// for key lifecycle management via the Key Custody Core (KCC).
package wsd_kcc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/google/uuid"
)

const (
	generateBindingKeypairEndpoint = "/v1/keys:generateBindingKeypair"
	generateKEMKeypairEndpoint     = "/v1/keys:generateKEMKeypair"
)

// GenerateBindingKeypairResponse is the JSON response for binding keypair generation.
type GenerateBindingKeypairResponse struct {
	KeyHandle string `json:"keyHandle"`
}

// GenerateKEMKeypairRequest is the JSON request body for KEM keypair generation.
type GenerateKEMKeypairRequest struct {
	BindingPublicKey string `json:"bindingPublicKey"` // base64-encoded
}

// GenerateKEMKeypairResponse is the JSON response for KEM keypair generation.
type GenerateKEMKeypairResponse struct {
	KeyHandle string `json:"keyHandle"`
}

// KeyGeneratorFunc defines function signatures for key generation operations.
// This enables dependency injection for testing.
type keyGenerator interface {
	generateBindingKeypair() (uuid.UUID, error)
	generateKEMKeypair(bindingPK []byte) (uuid.UUID, error)
}

// cgoKeyGenerator calls through to the CGO FFI functions.
type cgoKeyGenerator struct{}

func (cgoKeyGenerator) generateBindingKeypair() (uuid.UUID, error) {
	return GenerateBindingKeypair()
}

func (cgoKeyGenerator) generateKEMKeypair(bindingPK []byte) (uuid.UUID, error) {
	return GenerateKEMKeypair(bindingPK)
}

// Server is the WSD HTTP server that listens on a unix socket.
type Server struct {
	server      *http.Server
	netListener net.Listener
}

// New creates a new WSD server listening on the given unix socket path.
func New(ctx context.Context, unixSock string) (*Server, error) {
	nl, err := net.Listen("unix", unixSock)
	if err != nil {
		return nil, fmt.Errorf("cannot listen to the socket [%s]: %v", unixSock, err)
	}
	return newServer(nl, cgoKeyGenerator{}), nil
}

func newServer(nl net.Listener, kg keyGenerator) *Server {
	h := &handler{keygen: kg}
	return &Server{
		netListener: nl,
		server: &http.Server{
			Handler: h.mux(),
		},
	}
}

type handler struct {
	keygen keyGenerator
}

func (h *handler) mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(generateBindingKeypairEndpoint, h.handleGenerateBindingKeypair)
	mux.HandleFunc(generateKEMKeypairEndpoint, h.handleGenerateKEMKeypair)
	return mux
}

func (h *handler) handleGenerateBindingKeypair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	handle, err := h.keygen.generateBindingKeypair()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate binding keypair: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, GenerateBindingKeypairResponse{
		KeyHandle: handle.String(),
	})
}

func (h *handler) handleGenerateKEMKeypair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GenerateKEMKeypairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	bindingPK, err := base64.StdEncoding.DecodeString(req.BindingPublicKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid base64 in bindingPublicKey: %v", err), http.StatusBadRequest)
		return
	}
	if len(bindingPK) == 0 {
		http.Error(w, "bindingPublicKey must not be empty", http.StatusBadRequest)
		return
	}

	handle, err := h.keygen.generateKEMKeypair(bindingPK)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate KEM keypair: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, GenerateKEMKeypairResponse{
		KeyHandle: handle.String(),
	})
}

func writeJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

// Serve starts the server; blocks until shutdown.
func (s *Server) Serve() error {
	return s.server.Serve(s.netListener)
}

// Shutdown gracefully terminates the server.
func (s *Server) Shutdown(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	err2 := s.netListener.Close()
	if err != nil {
		return err
	}
	if err2 != nil {
		return err2
	}
	return nil
}
