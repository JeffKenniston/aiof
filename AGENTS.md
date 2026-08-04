# Antigravity Global Context (AGENTS.md)

## System Architecture & Orchestration
You are operating within the Antigravity 2.0 CLI framework. Your primary directive is to seamlessly orchestrate tasks by dynamically loading specialized, on-demand skills available in your environment. Do not attempt to perform deep domain work without first relying on the relevant semantic skill.

## Universal Operational Directives
* **Absolute Determinism & Real Implementations:** Eliminate conversational filler. Strictly prohibit placeholders, dummy data, `// TODO` scaffolding, or **faked/mocked/static metrics and functions**. All code, telemetry, and features must be fully functional and compute actual data. Outputs must be ready for deployment.
* **Execution Phasing:** 1. **Discover:** Find all execution boundaries, entry points, and dependencies.
  2. **Analyze:** Evaluate edge cases and security vectors based on discovery.
  3. **Execute:** Trigger specific domain skills to implement the solution.
* **State Compaction:** Limit token-heavy tool responses. Maintain long-running autonomy by continually synthesizing and compacting knowledge into `docs/KNOWLEDGE.md`. 
* **Observability:** Assume all telemetry is handled via eBPF and OpenTelemetry (OTEL) at the kernel level. Never inject manual application-layer proxy instrumentation.
* **Enterprise-Grade Structure:** Adhere strictly to standard Go project layouts. Use `/cmd` for executable entry points, `/internal` for private business logic, `/pkg` for generic/shared modules, `/proto` for API contracts, and `/docs` for supplementary documentation. Do not clutter the root directory with non-essential files.
* **Persona Mandate:** Every AI agent must be instantiated as an actual entity/persona aware of their distinct name, gender, and personality. Generic AI responses are prohibited.
* **LLM Transparency & Control:** Always surface model attribution (which model processed the request) and reasoning/thinking steps in the UI. Ensure any user "stop/cancel" action actively halts the underlying `context` to immediately terminate execution and save resources.
* **Enterprise Semantic Versioning (SemVer 2.0.0):** Adhere strictly to standard SemVer conventions. Use `0.x.y` for initial pre-release development leading up to `v1.0.0` (First Production Release). Current project baseline version is `v0.2.0-dev`. Do not inflate version numbers.
* **User Home Data Isolation (`~/.aiof`):** All end-user runtime files, temporary data, logs, assets, and project configs created by `aiof` MUST be stored strictly in `{user_dir}\.aiof\`. NEVER write runtime user data or temporary files into the project codebase repository directory.
* **Monorepo Virtualenv Hygiene:** Standardize strictly on `.venv/` as the single canonical Python virtual environment. Exclude all variant virtualenv folders (`venv/`, `env/`) in `.gitignore`.
* **Milestone & Backlog Ledger (`docs/TODO.md`):** Maintain `docs/TODO.md` as an enterprise-grade historical ledger tracking all work accomplished from `v0.0.0` to the current version, alongside granular future phase backlogs progressing to `v1.0.0`.
* **Dual-Host Monorepo Architecture (`apps/client` & `apps/web`):** Maintain `apps/client` (`@aiof/client`) as the multi-platform Tauri v2 native host app (Win, Mac, Linux, iOS, Android) and `apps/web` (`@aiof/web`) as the browser/PWA SPA. Neither host app shall duplicate presentation or database logic.
* **Workspace Package Extraction (`packages/*`):** All shared UI primitives, layouts, and design tokens MUST reside in `packages/ui` (`@aiof/ui`). All RxDB schemas, WAL replication, Zustand stores, and reactivity hooks MUST reside in `packages/rxdb-store` (`@aiof/rxdb-store`).

## Project Documentation
The following core documents define the project's vision, architecture, and schedule. They must be referenced to ensure alignment:
* **[Blueprint.md](file:///home/jeff/Development/Projects/aiof/docs/Blueprint.md):** Defines the system component blueprints and high-level architecture layers (Ingestion, Agentic Orchestration, Sandbox Execution, LLM Gateway).
* **[ADR.md](file:///home/jeff/Development/Projects/aiof/docs/ADR.md):** Architecture Decision Records. Documents the core technical choices (e.g., Go 1.23 middleware, SGLang inference, Connect RPC, Two-Pass Deserialization) and their associated trade-offs.
* **[Roadmap.md](file:///home/jeff/Development/Projects/aiof/docs/Roadmap.md):** The agile development roadmap outlining phased implementation steps (Phase 1: Core Runtime, Phase 2: Agent Interfaces, Phase 3: Sandbox Integration).

## Custom Commands
* **"dev server" (or similar):** When the user asks to start the dev server, you MUST first kill any currently running dev servers, then concurrently start the Vite frontend server using `npm run dev` in `/web` and the Go backend server (e.g. `./run.sh` in the root directory).
