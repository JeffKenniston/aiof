# AIOF Enterprise Monorepo Roadmap & Milestone Backlog

**Current Framework Version:** `v0.3.0-dev`  
**Versioning Strategy:** Enterprise Semantic Versioning (SemVer 2.0.0 — `0.x.y` initial development stage progressing toward `v1.0.0` first stable release).

---

## 1. Historical Milestone Ledger (v0.0.0 to Present)

### Milestone v0.0.0 -> v0.1.0: Core AI Orchestration Foundation
- [x] **Dual-Index Ingestion Engine (`internal/ingestion`)**
  - Built vector embeddings + AST symbol graph indexing for real-time codebase context retrieval.
  - Implemented DualIndexMemory with high-performance cosine similarity lookup.
- [x] **LLM Gateway & Asymmetric Routing (`internal/llm`)**
  - Integrated Gemini API client (Gemini 3.1 Pro, Flash, FlashLite) with failover routing.
  - Built Matrix Factorization (`RouteLLMClassifier`) non-autoregressive complexity classifier.
  - Implemented LMCache KV-pool manager with INT4 quantization & PCIe DMA page swapping.
  - Implemented speculative decoding with draft verification.
- [x] **Lock-Free A2A EventMesh & State Pipeline (`internal/eventmesh`, `internal/store`)**
  - Built lock-free ring-buffer EventMesh for sub-millisecond agent-to-agent pub/sub messaging.
  - Created SQLite Write-Ahead Log (WAL) persistent storage layer (`internal/store`).
- [x] **Decoupled Server & Web Workstation (`internal/api`, `/web`)**
  - Implemented REST / Connect RPC server handling prompt execution, project sync, and settings.
  - Built multi-viewport responsive React workstation UI (`/web`) supporting 4 workspace presets (Casual, Dev, Work, Design).
- [x] **Financial Cost & Policy Governance (`internal/governance`)**
  - Created `CostGovernanceEngine` enforcing hard dollar spending caps and token transaction ledgers.

---

### Milestone v0.1.0 -> v0.2.0: Universal Multi-Modal Project Engine & Monorepo Hardening
- [x] **Universal Multi-Modal Project Engine (`internal/project`)**
  - Designed core domain model, metadata, state transitions, and modality strategies.
  - Implemented 10-Modality Taxonomy handling 100% of workload domains:
    1. Software Engineering (`software_engineering`)
    2. Web Applications (`web_fullstack`)
    3. Data Science & ML (`datascience_ml`)
    4. Media & Video (`media_creative`)
    5. Design & 3D (`composable_hybrid`)
    6. Writing & Academic (`academic_research`)
    7. Business & Strategy (`enterprise_business`)
    8. DevOps & IaC (`devops_iac`)
    9. Hardware & IoT (`documentation_writing`)
    10. Composable Hybrid (`education_learning`)
  - Integrated thread-safe `project.Engine` into `internal/api/project.go` endpoints.
- [x] **End-User Home Storage Standard (`~/.aiof`)**
  - Standardized all runtime files, caches, asset previews, logs, and project configs to `{user_dir}\.aiof\` (`~/.aiof`).
  - Purged codebase repository of all temporary/runtime `.aiof/` folders.
- [x] **Monorepo Virtualenv & Directory Hygiene**
  - Standardized strictly on `.venv/` as the single canonical Python virtual environment across the repo.
  - Standardized project directory layout (`/cmd`, `/internal`, `/pkg`, `/proto`, `/docs`, `/web`).
- [x] **100% Monorepo Test Verification**
  - Resolved test edge-cases across 25+ packages (`internal/ast`, `internal/graphrag`, `internal/iac`, `internal/llm`, `internal/sandbox`, `internal/plugin`).
  - Verified 100% test pass rate (`go test ./...`) and zero-error binary build (`go build -o bin/aiof.exe ./cmd/aiof/main.go`).
- [x] **Polyglot Monorepo Dual-Host & Design System Extraction**
  - Restructured monorepo topology (`services/daemon`, `services/graphrag`, `apps/client`, `apps/web`, `packages/ui`, `packages/rxdb-store`, `packages/contracts`).
  - Extracted shared design system (`@aiof/ui`), HSL tokens, and primitive components into `packages/ui`.
  - Extracted RxDB offline sync and Zustand state stores into `packages/rxdb-store` (`@aiof/rxdb-store`).
  - Scaffolded `apps/web` (`@aiof/web`) as a Browser / PWA SPA with service worker offline support.
  - Transformed `apps/client` (`@aiof/client`) into a Tauri v2 native host application (Windows, macOS, Linux, iOS, Android).

---

### Milestone v0.2.0 -> v0.3.0: Ephemeral Swarms, Consensus, & July 2026 Stateless Architecture
- [x] **Stateless MCP 2026-07-28 Protocol Core (`apps/daemon/internal/mcp`)**
  - Implemented stateless multi-round-trip elicitation (`ElicitationChallenge` / `ElicitationResponse`) without transport session locking.
  - Added expiration TTLs and header capability token pass-through for serverless edge workers.
- [x] **Multi-Tenant OPA Policy Gating for Swarm Plasticity (`apps/daemon/internal/agent`)**
  - Added `TenantID` and `SecurityLabel` fields to `NodeState` in Autonomic Swarm Plasticity (`plasticity.go`).
  - Implemented zero-trust OPA tenant boundary checks in `executeCellFusionLocked` preventing cross-department prompt fusion.
- [x] **$Pass^k$ Multi-Attempt Trajectory Verification in BFT Quorums (`apps/daemon/internal/consensus`)**
  - Integrated statistical execution sampling ($pass^k$) into `BFTConsensusEngine`.
  - Added unbiased pass^k estimator calculation (`CalculatePassK`) and concurrent verification trials (`EvaluatePassK`) requiring consistent pass rates before vote commit validity.
- [x] **Fabricator Ephemeral Swarm Pipeline (ADR-21)**
  - Wired `FabricatorPipeline` into `SetValuedRouter` for runtime synthesis of task-tailored subagent manifests.
  - Implemented TTL auto-reclamation and CoW state isolation for shadow agent branches.
- [x] **Byzantine Fault Tolerant (BFT) Multi-Agent Consensus (ADR-14)**
  - Implemented dynamic quorum threshold adjustments based on agent confidence weights.
  - Added VRF-based verifiable random function leader selection for arbiter voting.

---

## 2. Active Engineering Backlog (v0.3.0 -> v0.4.0: MicroVMs & Wasm)
- [ ] **Firecracker MicroVM Provisioning (ADR-04)**
  - Build automated minimal Linux rootfs builder in `scripts/build_rootfs.sh`.
  - Implement fast-boot snapshot restoration for sub-10ms microVM execution.
- [ ] **Wasmtime Resource Metering & Fault Isolation (ADR-08)**
  - Add epoch-based fuel consumption limits for WebAssembly plugin drivers.
  - Implement memory boundary enforcement per plugin sandbox container.

---

## 4. Hardware Acceleration Backlog (v0.4.0 -> v0.5.0: Heterogeneous Silicon)
- [ ] **OpenVINO Heterogeneous Switchboard Tuning (ADR-16)**
  - Implement dynamic workload distribution between Intel Arc 140V iGPU (`GPU.0`) and NPU (`NPU.0`).
  - Add PCIe DMA bandwidth monitoring for LMCache KV-pool page swapping.
- [ ] **Speculative Decoding KV-Cache Compression (ADR-03)**
  - Benchmark INT4 quantization MSE accuracy on complex code refactoring tasks.
  - Optimize block table allocator chunk sizes for high-throughput batching.

---

## 5. First Stable Release Target (v0.5.0 -> v1.0.0: Production Readiness)
- [ ] **Multimodal Project Asset Preview Panel**
  - Expand React `/web` workstation with native previewers for Media/Video, Design/3D, and Data Science project assets.
- [ ] **eBPF Syscall Telemetry Visualizer**
  - Integrate kernel trace stream with OpenTelemetry span waterfall chart in the Expanded workstation viewport.
- [ ] **gRPC / Connect RPC Production Gateway**
  - Generate full Proto contract bindings (`/proto`) with Buf CLI for external client SDK integration.
