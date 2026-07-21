//go:build integration_gpu
// +build integration_gpu

package gpu

import (
	"crypto/sha256"
	"testing"

	attestationpb "github.com/GoogleCloudPlatform/confidential-space/server/proto/gen/attestation"
)

// TODO: Replace these placeholders with the actual values of your reusable A3 VM
const (
	ExpectedUUID         = "GPU-a8da6e1e-55b3-3291-a383-8f7dba2bb0a6" // Actual VM GPU UUID
	ExpectedDriverVer    = "610.43.02"                                // Actual driver version
	ExpectedVbiosVer     = "96.00.D9.00.01"                           // Actual VBIOS version
)

func TestAttest_RealHardware(t *testing.T) {
	// Enable attester for driver checks
	attester := NewNvidiaAttester(true)
	if attester == nil {
		t.Fatal("Expected non-nil NvidiaAttester, got nil")
	}

	// 1. Act: Call Attest() on real hardware
	nonce := []byte("integration-test-nonce")
	reportAny, err := attester.Attest(nonce)
	if err != nil {
		t.Fatalf("Attest() failed on real hardware: %v", err)
	}

	// 2. Assert: Check Types
	report, ok := reportAny.(*attestationpb.NvidiaAttestationReport)
	if !ok {
		t.Fatalf("Expected *attestationpb.NvidiaAttestationReport, got %T", reportAny)
	}

	// 3. Assert: Validate Nonce match
	nvNonce := sha256.Sum256(nonce)
	if string(report.Nonce) != string(nvNonce[:]) {
		t.Errorf("Expected nonce %x, got %x", nvNonce[:], report.Nonce)
	}

	// 4. Assert: Validate CC Feature and retrieve GPU Quotes
	var gpuQuotes []*attestationpb.GpuInfo
	
	switch cc := report.CcFeature.(type) {
	case *attestationpb.NvidiaAttestationReport_Spt:
		if cc.Spt == nil || cc.Spt.GpuQuote == nil {
			t.Fatal("SPT report is missing GpuQuote")
		}
		gpuQuotes = append(gpuQuotes, cc.Spt.GpuQuote)
	case *attestationpb.NvidiaAttestationReport_Mpt:
		if cc.Mpt == nil || len(cc.Mpt.GpuQuotes) == 0 {
			t.Fatal("MPT report is missing GpuQuotes")
		}
		gpuQuotes = cc.Mpt.GpuQuotes
	default:
		t.Fatalf("Unknown or missing CcFeature: %T", report.CcFeature)
	}

	// 5. Assert: Validate exact values against our anchored VM
	for i, quote := range gpuQuotes {
		// Exact Match Anchors
		if quote.Uuid != ExpectedUUID {
			t.Errorf("GPU[%d]: Expected UUID %q, got %q", i, ExpectedUUID, quote.Uuid)
		}
		if quote.DriverVersion != ExpectedDriverVer {
			t.Errorf("GPU[%d]: Expected Driver %q, got %q", i, ExpectedDriverVer, quote.DriverVersion)
		}
		if quote.VbiosVersion != ExpectedVbiosVer {
			t.Errorf("GPU[%d]: Expected VBIOS %q, got %q", i, ExpectedVbiosVer, quote.VbiosVersion)
		}

		// Structural Size Anchors (Must not be empty)
		if len(quote.AttestationReport) == 0 {
			t.Errorf("GPU[%d]: AttestationReport is empty", i)
		}
		if len(quote.AttestationCertificateChain) == 0 {
			t.Errorf("GPU[%d]: AttestationCertificateChain is empty", i)
		}
		if quote.GpuArchitectureType == attestationpb.GpuArchitectureType_GPU_ARCHITECTURE_TYPE_UNSPECIFIED {
			t.Errorf("GPU[%d]: Architecture type is unspecified", i)
		}
	}
}
