package wsd_kcc

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// fakeKeyGenerator implements keyGenerator for testing.
type fakeKeyGenerator struct {
	bindingHandle uuid.UUID
	bindingErr    error
	kemHandle     uuid.UUID
	kemErr        error
	lastBindingPK []byte
}

func (f *fakeKeyGenerator) generateBindingKeypair() (uuid.UUID, error) {
	return f.bindingHandle, f.bindingErr
}

func (f *fakeKeyGenerator) generateKEMKeypair(bindingPK []byte) (uuid.UUID, error) {
	f.lastBindingPK = bindingPK
	return f.kemHandle, f.kemErr
}

func newTestHandler(kg keyGenerator) http.Handler {
	h := &handler{keygen: kg}
	return h.mux()
}

func TestGenerateBindingKeypair(t *testing.T) {
	wantHandle := uuid.MustParse("01020304-0506-0708-090a-0b0c0d0e0f10")
	fake := &fakeKeyGenerator{bindingHandle: wantHandle}
	mux := newTestHandler(fake)

	req := httptest.NewRequest(http.MethodPost, generateBindingKeypairEndpoint, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp GenerateBindingKeypairResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.KeyHandle != wantHandle.String() {
		t.Errorf("got keyHandle %q, want %q", resp.KeyHandle, wantHandle.String())
	}
}

func TestGenerateBindingKeypairWrongMethod(t *testing.T) {
	fake := &fakeKeyGenerator{}
	mux := newTestHandler(fake)

	req := httptest.NewRequest(http.MethodGet, generateBindingKeypairEndpoint, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGenerateBindingKeypairError(t *testing.T) {
	fake := &fakeKeyGenerator{bindingErr: fmt.Errorf("vault creation failed")}
	mux := newTestHandler(fake)

	req := httptest.NewRequest(http.MethodPost, generateBindingKeypairEndpoint, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestGenerateKEMKeypair(t *testing.T) {
	wantHandle := uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	bindingPK := make([]byte, 32)
	for i := range bindingPK {
		bindingPK[i] = byte(i)
	}

	fake := &fakeKeyGenerator{kemHandle: wantHandle}
	mux := newTestHandler(fake)

	body := fmt.Sprintf(`{"bindingPublicKey":"%s"}`, base64.StdEncoding.EncodeToString(bindingPK))
	req := httptest.NewRequest(http.MethodPost, generateKEMKeypairEndpoint, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp GenerateKEMKeypairResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.KeyHandle != wantHandle.String() {
		t.Errorf("got keyHandle %q, want %q", resp.KeyHandle, wantHandle.String())
	}

	// Verify the binding PK was passed through correctly.
	if len(fake.lastBindingPK) != 32 {
		t.Fatalf("expected 32-byte binding PK, got %d bytes", len(fake.lastBindingPK))
	}
	for i, b := range fake.lastBindingPK {
		if b != byte(i) {
			t.Errorf("binding PK byte %d: got %d, want %d", i, b, i)
		}
	}
}

func TestGenerateKEMKeypairWrongMethod(t *testing.T) {
	fake := &fakeKeyGenerator{}
	mux := newTestHandler(fake)

	req := httptest.NewRequest(http.MethodGet, generateKEMKeypairEndpoint, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestGenerateKEMKeypairInvalidBody(t *testing.T) {
	fake := &fakeKeyGenerator{}
	mux := newTestHandler(fake)

	req := httptest.NewRequest(http.MethodPost, generateKEMKeypairEndpoint, strings.NewReader("not json"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGenerateKEMKeypairInvalidBase64(t *testing.T) {
	fake := &fakeKeyGenerator{}
	mux := newTestHandler(fake)

	body := `{"bindingPublicKey":"not-valid-base64!!!"}`
	req := httptest.NewRequest(http.MethodPost, generateKEMKeypairEndpoint, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGenerateKEMKeypairEmptyBindingPK(t *testing.T) {
	fake := &fakeKeyGenerator{}
	mux := newTestHandler(fake)

	// base64 of empty slice
	body := fmt.Sprintf(`{"bindingPublicKey":"%s"}`, base64.StdEncoding.EncodeToString([]byte{}))
	req := httptest.NewRequest(http.MethodPost, generateKEMKeypairEndpoint, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestGenerateKEMKeypairBackendError(t *testing.T) {
	fake := &fakeKeyGenerator{kemErr: fmt.Errorf("KEM generation failed")}
	mux := newTestHandler(fake)

	bindingPK := make([]byte, 32)
	body := fmt.Sprintf(`{"bindingPublicKey":"%s"}`, base64.StdEncoding.EncodeToString(bindingPK))
	req := httptest.NewRequest(http.MethodPost, generateKEMKeypairEndpoint, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
