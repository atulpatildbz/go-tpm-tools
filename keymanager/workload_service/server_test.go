package workload_service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// mockKeyGenerator implements keyGenerator for testing.
type mockKeyGenerator struct {
	bindingHandle uuid.UUID
	bindingErr    error
	kemHandle     uuid.UUID
	kemErr        error
}

func (m *mockKeyGenerator) GenerateBindingKeypair() (uuid.UUID, error) {
	return m.bindingHandle, m.bindingErr
}

func (m *mockKeyGenerator) GenerateKEMKeypair(bindingPK []byte) (uuid.UUID, error) {
	return m.kemHandle, m.kemErr
}

func TestHandleGenerateBindingKeypair(t *testing.T) {
	wantHandle := uuid.MustParse("12345678-1234-1234-1234-123456789abc")
	h := &handler{keygen: &mockKeyGenerator{bindingHandle: wantHandle}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	resp, err := http.Post(srv.URL+generateBindingKeypairEndpoint, "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result keyHandleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.KeyHandle != wantHandle.String() {
		t.Errorf("got keyHandle %q, want %q", result.KeyHandle, wantHandle.String())
	}
}

func TestHandleGenerateBindingKeypair_MethodNotAllowed(t *testing.T) {
	h := &handler{keygen: &mockKeyGenerator{}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + generateBindingKeypairEndpoint)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestHandleGenerateBindingKeypair_Error(t *testing.T) {
	h := &handler{keygen: &mockKeyGenerator{bindingErr: fmt.Errorf("test error")}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	resp, err := http.Post(srv.URL+generateBindingKeypairEndpoint, "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandleGenerateKEMKeypair(t *testing.T) {
	wantHandle := uuid.MustParse("abcdef01-2345-6789-abcd-ef0123456789")
	h := &handler{keygen: &mockKeyGenerator{kemHandle: wantHandle}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	body := `{"bindingPublicKey":"AQIDBA=="}`
	resp, err := http.Post(srv.URL+generateKEMKeypairEndpoint, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var result keyHandleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.KeyHandle != wantHandle.String() {
		t.Errorf("got keyHandle %q, want %q", result.KeyHandle, wantHandle.String())
	}
}

func TestHandleGenerateKEMKeypair_MethodNotAllowed(t *testing.T) {
	h := &handler{keygen: &mockKeyGenerator{}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	resp, err := http.Get(srv.URL + generateKEMKeypairEndpoint)
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestHandleGenerateKEMKeypair_MissingBindingPK(t *testing.T) {
	h := &handler{keygen: &mockKeyGenerator{}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	body := `{"bindingPublicKey":""}`
	resp, err := http.Post(srv.URL+generateKEMKeypairEndpoint, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGenerateKEMKeypair_InvalidJSON(t *testing.T) {
	h := &handler{keygen: &mockKeyGenerator{}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	resp, err := http.Post(srv.URL+generateKEMKeypairEndpoint, "application/json", strings.NewReader("{invalid"))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleGenerateKEMKeypair_Error(t *testing.T) {
	h := &handler{keygen: &mockKeyGenerator{kemErr: fmt.Errorf("test error")}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	body := `{"bindingPublicKey":"AQIDBA=="}`
	resp, err := http.Post(srv.URL+generateKEMKeypairEndpoint, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
}

func TestHandleGenerateKEMKeypair_EmptyBody(t *testing.T) {
	h := &handler{keygen: &mockKeyGenerator{}}
	srv := httptest.NewServer(h.routes())
	defer srv.Close()

	resp, err := http.Post(srv.URL+generateKEMKeypairEndpoint, "application/json", nil)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}
