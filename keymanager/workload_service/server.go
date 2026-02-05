// Package workload_service implements the Workload Service Daemon (WSD) HTTP
// server for key management operations.
package workload_service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/google/uuid"
)

const (
	generateBindingKeypairEndpoint = "/keys:generateBindingKeypair"
	generateKEMKeypairEndpoint     = "/keys:generateKEMKeypair"
)

// keyGenerator abstracts the key generation operations for testability.
type keyGenerator interface {
	GenerateBindingKeypair() (uuid.UUID, error)
	GenerateKEMKeypair(bindingPK []byte) (uuid.UUID, error)
}

// generateKEMKeypairRequest is the JSON request body for GenerateKEMKeypair.
type generateKEMKeypairRequest struct {
	BindingPublicKey []byte `json:"bindingPublicKey"`
}

// keyHandleResponse is the JSON response containing a key handle UUID.
type keyHandleResponse struct {
	KeyHandle string `json:"keyHandle"`
}

// Server is the WSD HTTP server for key management operations.
type Server struct {
	server      *http.Server
	netListener net.Listener
}

type handler struct {
	keygen keyGenerator
}

// New creates a new WSD server listening on the given unix socket path.
func New(ctx context.Context, unixSock string, keygen keyGenerator) (*Server, error) {
	nl, err := net.Listen("unix", unixSock)
	if err != nil {
		return nil, fmt.Errorf("cannot listen on socket [%s]: %v", unixSock, err)
	}

	h := &handler{keygen: keygen}
	return &Server{
		netListener: nl,
		server: &http.Server{
			Handler: h.routes(),
		},
	}, nil
}

func (h *handler) routes() http.Handler {
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

	handle, err := h.keygen.GenerateBindingKeypair()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate binding keypair: %v", err), http.StatusInternalServerError)
		return
	}

	writeKeyHandleResponse(w, handle)
}

func (h *handler) handleGenerateKEMKeypair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req generateKEMKeypairRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(req.BindingPublicKey) == 0 {
		http.Error(w, "bindingPublicKey is required", http.StatusBadRequest)
		return
	}

	handle, err := h.keygen.GenerateKEMKeypair(req.BindingPublicKey)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to generate KEM keypair: %v", err), http.StatusInternalServerError)
		return
	}

	writeKeyHandleResponse(w, handle)
}

func writeKeyHandleResponse(w http.ResponseWriter, handle uuid.UUID) {
	resp := keyHandleResponse{KeyHandle: handle.String()}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// Serve starts the server. Blocks until the server shuts down.
func (s *Server) Serve() error {
	return s.server.Serve(s.netListener)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
