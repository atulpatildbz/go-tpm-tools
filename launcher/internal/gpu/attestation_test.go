package gpu

import (
	"fmt"
	"testing"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/deviceinfo"

	attestationpb "github.com/GoogleCloudPlatform/confidential-space/server/proto/gen/attestation"
	pb "github.com/google/go-nvattest-tools/proto/nvattest"
)

type mockGpuQuoteProvider struct {
	quote *pb.GpuAttestationQuote
	err   error
}

func (m *mockGpuQuoteProvider) CollectGpuEvidence(nonce [32]byte) (*pb.GpuAttestationQuote, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.quote, nil
}

func TestCollectAttestationEvidence(t *testing.T) {
	validQuote := &pb.GpuAttestationQuote{
		GpuInfos: []*pb.GpuInfo{
			{
				Uuid:                        "GPU-1234",
				DriverVersion:               "550.00.00",
				VbiosVersion:                "99.00.00.00",
				GpuArchitecture:             pb.GpuArchitectureType_GPU_ARCHITECTURE_HOPPER,
				AttestationCertificateChain: []byte("cert-chain"),
				AttestationReport:           []byte("report"),
			},
		},
	}

	testCases := []struct {
		name     string
		nonce    []byte
		gpuType  deviceinfo.GPUType
		provider *mockGpuQuoteProvider
		wantPass bool
		wantSPT  bool
		wantMPT  bool
	}{
		{
			name:    "success w/ H100 SPT",
			nonce:   []byte("nonce"),
			gpuType: deviceinfo.H100,
			provider: &mockGpuQuoteProvider{
				quote: validQuote,
			},
			wantPass: true,
			wantSPT:  true,
		},
		{
			name:    "failed due to unsupported GPU attestation type",
			nonce:   []byte("nonce"),
			gpuType: deviceinfo.Others,
			provider: &mockGpuQuoteProvider{
				quote: validQuote,
			},
			wantPass: false,
		},
		{
			name:    "provider error",
			nonce:   []byte("nonce"),
			gpuType: deviceinfo.H100,
			provider: &mockGpuQuoteProvider{
				err: fmt.Errorf("provider error"),
			},
			wantPass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fn := &getGpuTypeInfo
			getGpuTypeInfo = func() (deviceinfo.GPUType, error) {
				return tc.gpuType, nil
			}
			// Restore to original func after testing
			t.Cleanup(func() { getGpuTypeInfo = *fn })

			attester := &NvidiaAttester{}
			attesation, err := attester.collectAttestationEvidence(tc.provider, tc.nonce)
			if gotPass := (err == nil); gotPass != tc.wantPass {
				t.Errorf("CollectAttestationEvidence() pass = %v, want %v", gotPass, tc.wantPass)
			}
			if tc.wantPass {
				if tc.wantSPT {
					if _, ok := attesation.CcFeature.(*attestationpb.NvidiaAttestationReport_Spt); !ok {
						t.Errorf("CollectAttestationEvidence() CcFeature = %T, want *attestationpb.NvidiaAttestationReport_Spt", attesation.CcFeature)
					}
				}
			}
		})
	}
}

func TestDetermineAttestationType(t *testing.T) {
	testCases := []struct {
		name     string
		gpuInfos []*attestationpb.GpuInfo
		gpuType  deviceinfo.GPUType
		want     attestationType
	}{
		{
			name: "UNSUPPORTED GPU type",
			gpuInfos: []*attestationpb.GpuInfo{
				{Uuid: "gpu-0"},
			},
			gpuType: deviceinfo.Others,
			want:    UNSUPPORTED,
		},
		{
			name: "SPT attestation type (H100)",
			gpuInfos: []*attestationpb.GpuInfo{
				{Uuid: "gpu-0"},
			},
			gpuType: deviceinfo.H100,
			want:    SPT,
		},
		{
			name: "SPT attestation type (B200 with single GPU)",
			gpuInfos: []*attestationpb.GpuInfo{
				{Uuid: "gpu-0"},
			},
			gpuType: deviceinfo.B200,
			want:    SPT,
		},
		{
			name: "MPT attestation type (B200 with multiple GPUs)",
			gpuInfos: []*attestationpb.GpuInfo{
				{Uuid: "gpu-0"},
				{Uuid: "gpu-1"},
			},
			gpuType: deviceinfo.B200,
			want:    MPT,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fn := &getGpuTypeInfo
			getGpuTypeInfo = func() (deviceinfo.GPUType, error) {
				return tc.gpuType, nil
			}
			// Restore to original func after testing
			t.Cleanup(func() { getGpuTypeInfo = *fn })

			if got := determineAttestationType(tc.gpuInfos); got != tc.want {
				t.Errorf("determineAttestationType() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConvertGPUArchToPB(t *testing.T) {
	testCases := []struct {
		name     string
		arch     pb.GpuArchitectureType
		wantArch attestationpb.GpuArchitectureType
	}{
		{
			name:     "HOPPER",
			arch:     pb.GpuArchitectureType_GPU_ARCHITECTURE_HOPPER,
			wantArch: attestationpb.GpuArchitectureType_GPU_ARCHITECTURE_TYPE_HOPPER,
		},
		{
			name:     "BLACKWELL",
			arch:     pb.GpuArchitectureType_GPU_ARCHITECTURE_BLACKWELL,
			wantArch: attestationpb.GpuArchitectureType_GPU_ARCHITECTURE_TYPE_BLACKWELL,
		},
		{
			name:     "UNSPECIFIED",
			arch:     pb.GpuArchitectureType_GPU_ARCHITECTURE_UNSPECIFIED,
			wantArch: attestationpb.GpuArchitectureType_GPU_ARCHITECTURE_TYPE_UNSPECIFIED,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := convertGPUArchToPB(tc.arch); got != tc.wantArch {
				t.Errorf("convertGPUArchToPB() = %v, want %v", got, tc.wantArch)
			}
		})
	}
}
