# AIOF Enterprise Monorepo & Workstation Knowledge Base (KNOWLEDGE.md)

## Monorepo Architecture & Directory Taxonomy
The repository is structured as a polyglot monorepo governed by **Go Workspaces (`go.work`)** for backend Go modules, **pnpm workspaces + Turborepo** for TypeScript/React packages, and **Buf** for gRPC/ConnectRPC contract generation.

```text
aiof/
├── .agents/                        # Agent system prompts, capabilities, & manifests
├── apps/                           # Runnable Host Applications & Daemon Services
│   ├── client/                     # Tauri v2 Native Host Application (@aiof/client)
│   ├── web/                        # React + Vite + RxDB Web/PWA SPA (@aiof/web)
│   ├── daemon/                     # Go 1.26 Agent Orchestration Daemon (aiof daemon)
│   │   ├── cmd/                    # Daemon executable entry points
│   │   ├── internal/               # Core private business logic
│   │   ├── pkg/                    # Exportable Go packages
│   │   └── migrations/             # SQLite WAL schema migrations
│   └── graphrag/                   # Python GraphRAG Sidecar Service (.venv, requirements.txt)
├── packages/                       # Shared Cross-Runtime Workspace Packages
│   ├── ui/                         # Shared React UI Component System & Tokens (@aiof/ui)
│   ├── rxdb-store/                 # Shared RxDB Storage Engine & Zustand Stores (@aiof/rxdb-store)
│   └── contracts/                  # Polyglot API Schemas & Generated Clients (@aiof/contracts)
├── infra/                          # Infrastructure & Orchestration Configs
│   ├── docker/                     # Container definitions
│   └── deployments/                # Kubernetes & Helm deployment specs
├── bin/                            # Compiled executable output binaries (git-ignored)
├── docs/                           # Documentation, AGENTS.md, KNOWLEDGE.md, & TODO.md
├── scripts/                        # Maintenance & ML Training Scripts (scripts/dev, scripts/ml)
├── go.work                         # Go 1.26 Multi-Module Workspace (targets ./apps/daemon)
├── pnpm-workspace.yaml             # TypeScript Workspace (targets apps/* and packages/*)
├── turbo.json                      # Turborepo task pipeline configuration
└── Makefile                        # Unified Developer Task Orchestration
```

---

## Universal Project Engine Architecture (`apps/daemon/internal/project`)
The v0.2.0 Universal Multi-Modal Project Engine provides high-performance, enterprise-grade project orchestration handling 100% of workload domains across 10 distinct operational modalities:
- **Software Engineering (`software_engineering`)**
- **Web Applications (`web_fullstack`)**
- **Data Science & ML (`datascience_ml`)**
- **Media & Video (`media_creative`)**
- **Design & 3D (`composable_hybrid`)**
- **Writing & Academic (`academic_research`)**
- **Business & Strategy (`enterprise_business`)**
- **DevOps & IaC (`devops_iac`)**
- **Hardware & IoT (`documentation_writing`)**
- **Composable Hybrid (`education_learning`)**

---

## Monorepo Layout & Environment Hygiene
- **Go Project Structure:** Managed via root `go.work` linking `apps/daemon`.
- **Python Virtual Environments:** Isolated strictly within `apps/graphrag/.venv/`. Root directory remains clean of runtime environments.
- **Clean Binaries:** Output binaries target `bin/` (`bin/aiof.exe`).
- **User Home Storage Standard:** All end-user runtime files, caches, assets, and project configs route strictly to `{user_dir}\.aiof\` (`~/.aiof`).

---

## v0.3.0-dev Stateless & Governance Architecture
- **Stateless MCP 2026-07-28 Protocol Core (`apps/daemon/internal/mcp`):** Supports multi-round-trip tool elicitation (`ElicitationChallenge` / `ElicitationResponse`) and header capability pass-through without session socket locking.
- **OPA Policy-Gated Swarm Plasticity (`apps/daemon/internal/agent`):** Subagent nodes carry `TenantID` and `SecurityLabel` tags. Cross-tenant prompt fusion attempts are rejected with `ErrTenantIsolationViolation` regardless of message volume.
- **$Pass^k$ BFT Trajectory Verification (`apps/daemon/internal/consensus`):** BFT consensus rounds compute unbiased $pass^k$ statistical consistency scores across $k$ concurrent sandbox trials, rejecting flaky proposals below `MinPassKThreshold` (default `0.80`).

