package gpu

import (
	"crypto/sha256"
	"fmt"

	"cos.googlesource.com/cos/tools.git/src/cmd/cos_gpu_installer/deviceinfo"
	
	"github.com/google/go-nvattest-tools/client"
	pb "github.com/google/go-nvattest-tools/proto/nvattest"

	attestationpb "github.com/GoogleCloudPlatform/confidential-space/server/proto/gen/attestation"
	"github.com/google/go-tpm-tools/proto/attest"
)

type attestationType int

const (
	// UNSUPPORTED indicates unsupported GPU attestation type like PPCIE.
	UNSUPPORTED attestationType = iota
	// SPT indicates Nvidia's single GPU passthrough attestation
	SPT
	// MPT indicates Nvidia's multi GPU secure passthrough attestation
	MPT
)

// Stub this func for testing purpose.
var getGpuTypeInfo = deviceinfo.GetGPUTypeInfo

// Attester defines the interface for GPU attestation.
type Attester interface {
	Attest(nonce []byte) (any, error)
	EnableReadyState() error
}

// NvidiaAttester is responsible for collecting GPU attestation.
type NvidiaAttester struct{}

// NewNvidiaAttester returns a new NvidiaAttester if installGpuDriver is true, otherwise nil.
func NewNvidiaAttester(installGpuDriver bool) Attester {
	if !installGpuDriver {
		return nil
	}
	return &NvidiaAttester{}
}

// Attest returns a GPU attestation.
func (a *NvidiaAttester) Attest(nonce []byte) (any, error) {
	if a == nil {
		return nil, fmt.Errorf("nil Nvidia attester")
	}
	gpuAttestation, err := a.collectAttestationEvidence(&client.LinuxGpuQuoteProvider{}, nonce)
	if err != nil {
		return nil, err
	}
	return gpuAttestation, nil
}

// EnableReadyState checks the Confidential Computing mode and transitions the GPU to a READY state if CC is enabled.
func (a *NvidiaAttester) EnableReadyState() error {
	if a == nil {
		return fmt.Errorf("nil Nvidia attester")
	}

	ccModeCmd := NvidiaSmiOutputFunc("conf-compute", "-f")
	devToolsCmd := NvidiaSmiOutputFunc("conf-compute", "-d")

	ccEnabled, err := QueryCCMode(ccModeCmd, devToolsCmd)
	if err != nil {
		return fmt.Errorf("failed to check confidential compute mode status: %v", err)
	}

	// Explicitly need to set the GPU state to READY for GPUs with confidential compute mode ON.
	if ccEnabled == attest.GPUDeviceCCMode_ON {
		setGPUStateCmd := NvidiaSmiOutputFunc("conf-compute", "-srs", "1")
		if err := setGPUStateToReady(setGPUStateCmd); err != nil {
			return fmt.Errorf("failed to set the GPU state to ready: %v", err)
		}
	}
	return nil
}

// collectAttestationEvidence assumes CC GPU devices are in place w/ driver support
// and will try to collect raw attestation evidence and convert it to known data models.
func (a *NvidiaAttester) collectAttestationEvidence(provider client.GpuQuoteProvider, nonce []byte) (*attestationpb.NvidiaAttestationReport, error) {
	nvNonce := sha256.Sum256(nonce)

	var nonce32 [32]byte
	copy(nonce32[:], nvNonce[:])

	quote, err := client.GpuQuote(provider, nonce32)
	if err != nil {
		return nil, fmt.Errorf("failed to collect GPU evidence: %v", err)
	}

	var gpuInfos []*attestationpb.GpuInfo
	for _, info := range quote.GpuInfos {
		gpuInfo := &attestationpb.GpuInfo{
			Uuid:                        info.Uuid,
			DriverVersion:               info.DriverVersion,
			VbiosVersion:                info.VbiosVersion,
			GpuArchitectureType:         convertGPUArchToPB(info.GpuArchitecture),
			AttestationReport:           info.AttestationReport,
			AttestationCertificateChain: info.AttestationCertificateChain,
		}
		gpuInfos = append(gpuInfos, gpuInfo)
	}

	switch determineAttestationType(gpuInfos) {
	case SPT:
		return &attestationpb.NvidiaAttestationReport{
			CcFeature: &attestationpb.NvidiaAttestationReport_Spt{
				Spt: &attestationpb.NvidiaAttestationReport_SinglePassthroughAttestation{
					GpuQuote: gpuInfos[0],
				},
			},
			Nonce: nvNonce[:],
		}, nil
	case MPT:
		return &attestationpb.NvidiaAttestationReport{
			CcFeature: &attestationpb.NvidiaAttestationReport_Mpt{
				Mpt: &attestationpb.NvidiaAttestationReport_MultiGpuSecurePassthroughAttestation{
					GpuQuotes: gpuInfos,
				},
			},
			Nonce: nvNonce[:],
		}, nil
	default:
		return nil, fmt.Errorf("unsupported GPU attestation")
	}
}

// determineAttesationType auto-detects the GPU attestation type.
// The current implementations "guess" the attestation type.
// Further improvement should be made to parse GPU attesation report to get the actual attestation type.
func determineAttestationType(gpuInfos []*attestationpb.GpuInfo) attestationType {
	gpuType, _ := getGpuTypeInfo()
	if gpuType != deviceinfo.H100 && gpuType != deviceinfo.B200 {
		return UNSUPPORTED
	}
	if gpuType == deviceinfo.B200 && len(gpuInfos) > 1 {
		return MPT
	}
	return SPT
}

func convertGPUArchToPB(arch pb.GpuArchitectureType) attestationpb.GpuArchitectureType {
	switch arch {
	case pb.GpuArchitectureType_GPU_ARCHITECTURE_HOPPER:
		return attestationpb.GpuArchitectureType_GPU_ARCHITECTURE_TYPE_HOPPER
	case pb.GpuArchitectureType_GPU_ARCHITECTURE_BLACKWELL:
		return attestationpb.GpuArchitectureType_GPU_ARCHITECTURE_TYPE_BLACKWELL
	default:
		return attestationpb.GpuArchitectureType_GPU_ARCHITECTURE_TYPE_UNSPECIFIED
	}
}
