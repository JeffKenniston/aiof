# AIOF Workstation Knowledge Base (KNOWLEDGE.md)

## Workspace Architecture
The workstation uses a dynamic layout grid driven by the active workspace preset. Rather than storing static cell arrays in state, layouts are computed on the fly by querying the preset registry using the active workspace ID and the resolved view viewport.

### Workspace Types
Workspaces are defined in [types.ts](file:///home/jeff/projects/aiof/web/src/workspace/types.ts) and configured in [presets.ts](file:///home/jeff/projects/aiof/web/src/workspace/presets.ts):
- **Casual (`casual`):** Everyday AI chatbot tasks.
- **Dev (`dev`):** Programming environment containing code editors, terminals, and file explorer.
- **Work (`work`):** Document compilation and research.
- **Design (`design`):** Visual assets and preview canvases.

---

## Responsive Viewports & Layouts
The layout system maps viewport widths into four view mode states:
1.  **Condensed (Phone, `< 640px`):** Single panel active at 100% dimensions, supported by bottom tab navigation bar containing primary panels.
2.  **Semi-full (Tablet, `640px - 1023px`):** Simpler 2-column or 3-panel split.
3.  **Full (Laptop/Desktop, `1024px - 1439px`):** Standard 4-panel dashboard.
4.  **Expanded (Immersive Workstation, `>= 1440px`):** Comprehensive 5-6 panel grid showing full telemetry and logging.

### Viewport Override
Users can force override viewport sizes (Auto, Condensed, Semi-full, Full, Expanded) using the dropdown selector in the [CommandBar](file:///home/jeff/projects/aiof/web/src/layout/CommandBar.tsx).

---

## Performance Tuning (60 FPS / 120 FPS)
To achieve target frame rates:
- **GPU Promotion:** The `.gpu-accelerated` style forces separate composite layers using `transform: translate3d(0, 0, 0)`.
- **Layout Thrashing Defense:** Layout sizing transitions animate strictly using `transform` and `opacity` properties instead of standard box attributes (`width`, `height`, `top`, `left`) which trigger browser document reflows.
- **Fluid Typography:** Scale variables use CSS `clamp()` combined with viewport-relative units to maintain optimal density across multiple resolutions without visual noise.

---

## E2E Spec Assertions
E2E testing suites assert layout structure using these specific DOM markers:
- **Layout Selector:** `.adaptive-pane`
- **Class Matchers:** `/compact-stack/` (condensed) and `/medium-split/` (semi-full).
- **Text Indicator:** `Status: WAL Hydrated & RxDB Initialized`
- **Mutation Dispatcher:** Button named `Dispatch Local State Mutation` which inserts local events into the RxDB WAL.

---

## Environment & Runtime Baseline
- **Go Version:** `1.26.4 linux/amd64`
- **Node.js / npm:** `Node v22.22.1`, `npm 9.2.0`
- **Backend Build:** `go build -o aiof_server cmd/aiof/*.go` (or via `./run.sh`)
- **Backend Test Status:** 100% pass across all packages (`aiof/internal/agent`, `aiof/internal/api`, `aiof/internal/ingestion`, `aiof/internal/llm`, `aiof/internal/orchestrator`, `aiof/internal/sandbox`, `aiof/internal/store`, `aiof/internal/telemetry`)
- **Frontend Workstation:** In `web/`, built with Vite v8 + React 19 + TypeScript + Tailwind CSS. Production bundle compiled via `npm run build`.
- **Architecture Documentation:** Consolidated at [ARCHITECTURE.md](file:///home/jeff/projects/aiof/docs/ARCHITECTURE.md) covering ADRs 1–6, System Component Blueprints (1–4), and Agile Development Roadmap (Phases 1–6).

---

## Repository Layout & Testing Architecture
- **`/cmd`**: Executable entry points (`cmd/aiof`).
- **`/internal`**: Domain packages, business logic, and co-located Go unit tests (`*_test.go`).
- **`/proto`**: Protobuf schema contracts (`buf.gen.yaml`).
- **`/docs`**: High-level system architecture, ADRs, and roadmap.
- **`/scripts`**: Utility and training scripts (`train_router.py`, `run_tests.py`).
- **`/tests`**: Root test suite ([README.md](file:///home/jeff/projects/aiof/tests/README.md)):
  - `tests/e2e/`: Multi-tier and browser validation ([`adaptive.spec.ts`](file:///home/jeff/projects/aiof/tests/e2e/adaptive.spec.ts), [`test_ui.js`](file:///home/jeff/projects/aiof/tests/e2e/test_ui.js)).
  - `tests/integration/`: Service and database integration suites ([`server_integration_test.go`](file:///home/jeff/projects/aiof/tests/integration/server_integration_test.go)).
- **`/web`**: React 19 / Vite Adaptive Workstation.


