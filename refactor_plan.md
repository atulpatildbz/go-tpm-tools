# Refactor Plan: `go-nvtrust` to `go-nvattest-tools`

## Objective
Safely replace the deprecated/older `go-nvtrust` library with `github.com/google/go-nvattest-tools` for NVIDIA GPU attestation, while maintaining near 100% confidence that the exact same attestation payload is generated.

## What We've Done So Far

1. **Identified the Testing Gap:** 
   We realized that standard unit tests mocking the NVML driver wouldn't survive the refactor because the interface boundaries between the two libraries are fundamentally different.
2. **Established an Integration Anchor:** 
   Decided the only way to prove the refactor is mathematically identical is to write a physical hardware integration test (`attestation_integration_test.go`) and anchor it to a specific A3 VM (`jetski-test-a3-ubuntu24`).
3. **Built Deployment Tooling:** 
   Created `scripts/run_gpu_integration.sh` to compile the binary and SCP/SSH it to the remote VM.
4. **Overcame Dynamic Linking Issues:** 
   Discovered that the VM's NVIDIA driver (610.43.02) was missing a symbol expected by the `go-nvml` wrapper. Bypassed this by compiling the Go binary with CGO and Lazy Binding (`CGO_LDFLAGS="-Wl,-z,lazy"`) so the program could boot and attest successfully.
5. **Locked in Baseline Assertions:** 
   Successfully ran the test on `jetski-test-a3-ubuntu24` and extracted its actual hardware properties:
   - **UUID**: `GPU-a8da6e1e-55b3-3291-a383-8f7dba2bb0a6`
   - **Driver Version**: `610.43.02`
   - **VBIOS Version**: `96.00.D9.00.01`
   We hardcoded these into `attestation_integration_test.go`. The baseline test is now **100% passing**.

## Next Steps

### 1. Refactor Production Code (`launcher/internal/gpu/attestation.go`)
- Remove the `github.com/confidentsecurity/go-nvtrust/pkg/gonvtrust/gpu` dependency.
- Import `github.com/google/go-nvattest-tools/client` and `github.com/google/go-nvattest-tools/proto/nvattest`.
- Update `NvidiaAttester` to utilize `client.LinuxGpuQuoteProvider{}`.
- Replace the manual `deviceInfo` iteration loop with a single call to `client.GpuQuote(provider, nonce)`.
- Write a mapping function to convert the returned `pb.GpuInfo` (from nvattest-tools) directly into `attestationpb.GpuInfo` (the confidential-space format).

### 2. Update Unit Tests (`launcher/internal/gpu/attestation_test.go`)
- Remove the deprecated `NVMLHandlerMock`.
- Update the unit test to inject a custom `client.GpuQuoteProvider` interface for fast, local unit testing.

### 3. Verify on Hardware
- Run `./scripts/run_gpu_integration.sh jetski-test-a3-ubuntu24`.
- The script must pass completely green against our exact anchored UUID and versions. If it passes, it proves the refactored code correctly interacts with physical hardware identically to the old code.

### 4. Cleanup
- `go mod tidy` to fully remove `go-nvtrust` from the module.
- Commit the changes.
