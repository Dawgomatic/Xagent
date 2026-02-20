# New File Requests - SWE100821

## pkg/hwprofile/hwprofile.go
- **Purpose**: Runtime hardware fingerprinting and compute-tier classification
- **Searched for duplicates**:
  - `llm-checker/src/hardware/detector.js` — JS/Node, not reusable in Go binary
  - `llm-checker/src/hardware/pc-optimizer.js` — JS systeminformation lib, not Go
  - `llm-checker/src/hardware/unified-detector.js` — JS multi-backend, not Go
  - `llm-checker/src/hardware/backends/cuda-detector.js` — JS nvidia-smi wrapper
  - `start.sh detect_system()` — Bash, install-time only, no runtime use
  - No existing Go hardware detection package found in `pkg/`
- **Decision**: New package required. Go stdlib + Linux proc/sys (zero deps).
