# Antigravity Global Context (AGENTS.md)

## System Architecture & Orchestration
You are operating within the Antigravity 2.0 CLI framework. Your primary directive is to seamlessly orchestrate tasks by dynamically loading specialized, on-demand skills available in your environment. Do not attempt to perform deep domain work without first relying on the relevant semantic skill.

## Universal Operational Directives
* **Absolute Determinism:** Eliminate conversational filler. Prohibit the use of placeholders, dummy data, or `// TODO` scaffolding. All outputs must be logically sound, syntactically complete, structurally verified, and ready for deployment.
* **Execution Phasing:** 1. **Discover:** Find all execution boundaries, entry points, and dependencies.
  2. **Analyze:** Evaluate edge cases and security vectors based on discovery.
  3. **Execute:** Trigger specific domain skills to implement the solution.
* **State Compaction:** Limit token-heavy tool responses. Maintain long-running autonomy by continually synthesizing and compacting knowledge into a project-scoped 'KNOWLEDGE.md' file placed directly in each project's root folder. 
* **Observability:** Assume all telemetry is handled via eBPF and OpenTelemetry (OTEL) at the kernel level. Never inject manual application-layer proxy instrumentation.
* **Enterprise-Grade Structure:** Adhere strictly to standard Go project layouts. Use `/cmd` for executable entry points, `/internal` for private business logic, `/pkg` for generic/shared modules, `/proto` for API contracts, and `/docs` for supplementary documentation. Do not clutter the root directory with non-essential files.

## Project Documentation
The following core documents define the project's vision, architecture, and schedule. They must be referenced to ensure alignment:
* **[Blueprint.md](file:///home/jeff/Development/Projects/aiof/docs/Blueprint.md):** Defines the system component blueprints and high-level architecture layers (Ingestion, Agentic Orchestration, Sandbox Execution, LLM Gateway).
* **[ADR.md](file:///home/jeff/Development/Projects/aiof/docs/ADR.md):** Architecture Decision Records. Documents the core technical choices (e.g., Go 1.23 middleware, SGLang inference, Connect RPC, Two-Pass Deserialization) and their associated trade-offs.
* **[Roadmap.md](file:///home/jeff/Development/Projects/aiof/docs/Roadmap.md):** The agile development roadmap outlining phased implementation steps (Phase 1: Core Runtime, Phase 2: Agent Interfaces, Phase 3: Sandbox Integration).

## Custom Commands
* **"dev server" (or similar):** When the user asks to start the dev server, you MUST first kill any currently running dev servers, then concurrently start the Vite frontend server using `npm run dev` in `/web` and the Go backend server (e.g. `./run.sh` in the root directory).
