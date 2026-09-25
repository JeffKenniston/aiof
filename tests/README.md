# AIOF Test Suite

This directory contains all dedicated integration and end-to-end (E2E) test suites for the AIOF platform, complementing package-level unit tests located alongside source files in `/internal`.

## Test Organization

| Tier | Directory | Files | Description | Runner |
| :--- | :--- | :--- | :--- | :--- |
| **Go Integration Tests** | `tests/integration/` | [`server_integration_test.go`](file:///home/jeff/projects/aiof/tests/integration/server_integration_test.go) | End-to-end Go HTTP server, health, and file system endpoint tests. | `go test -v ./tests/integration/...` |
| **Frontend & UI E2E** | `tests/e2e/` | [`adaptive.spec.ts`](file:///home/jeff/projects/aiof/tests/e2e/adaptive.spec.ts), [`test_ui.js`](file:///home/jeff/projects/aiof/tests/e2e/test_ui.js) | Playwright browser automation validating viewport transitions, WAL local replica hydration, and agent node catalog. | Playwright / Node runner |
| **Package Unit Tests** | `internal/*/*_test.go` | `agent/`, `api/`, `llm/`, `store/`, `orchestrator/`, `sandbox/`, `telemetry/`, `ingestion/` | Unit-level whitebox logic tests co-located with private Go packages. | `go test ./internal/...` |

## Running Tests

### All Go Tests (Unit & Integration)
```bash
go test -v ./...
# Or via the test runner script:
python3 scripts/run_tests.py
```

### Integration Test Suite Only
```bash
go test -v ./tests/integration/...
```

### E2E Browser Tests
Ensure both the backend orchestrator and frontend workstation are running (`./run.sh` & `npm run dev --prefix web`):
```bash
# Agent catalog DOM validation
node tests/e2e/test_ui.js

# Viewport boundary & WAL hydration specification
npx playwright test tests/e2e/adaptive.spec.ts
```
