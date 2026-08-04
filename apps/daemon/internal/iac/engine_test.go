package iac

import (
	"context"
	"testing"
)

func TestOPAPolicyEvaluator(t *testing.T) {
	eval := NewOPAPolicyEvaluator()

	// 1. Test Valid Secure Manifest
	validRes := IaCResource{
		ID:       "res-1",
		Name:     "web-deploy",
		Kind:     "Deployment",
		Provider: "kubernetes",
		Payload: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web-deploy
spec:
  template:
    spec:
      containers:
      - name: web
        image: nginx:alpine
        securityContext:
          runAsNonRoot: true
        resources:
          limits:
            cpu: "500m"
            memory: "512Mi"
`,
	}

	manifest := IaCManifest{
		Name:      "secure-app",
		Type:      IaCTypeKubernetes,
		Resources: []IaCResource{validRes},
	}

	report := eval.EvaluateManifest(&manifest)
	if !report.Passed {
		t.Fatalf("Expected valid manifest to pass OPA policy evaluation, got violations: %+v", report.Violations)
	}

	// 2. Test Insecure Manifest with Critical Violations (Unencrypted S3, Privileged Container, Plaintext Secret)
	insecureRes1 := IaCResource{
		ID:       "res-2",
		Name:     "insecure-bucket",
		Kind:     "aws_s3_bucket",
		Provider: "aws",
		Payload:  `resource "aws_s3_bucket" "b" { bucket = "my-bucket" encrypted = false }`,
	}

	insecureRes2 := IaCResource{
		ID:       "res-3",
		Name:     "root-pod",
		Kind:     "Deployment",
		Provider: "kubernetes",
		Payload:  `kind: Deployment metadata: name: root-pod spec: runAsUser: 0 privileged: true env: API_KEY=secret_key_12345`,
	}

	insecureManifest := IaCManifest{
		Name:      "insecure-app",
		Type:      IaCTypeTerraform,
		Resources: []IaCResource{insecureRes1, insecureRes2},
	}

	insecureReport := eval.EvaluateManifest(&insecureManifest)
	if insecureReport.Passed {
		t.Errorf("Expected insecure manifest to fail OPA evaluation")
	}

	if insecureReport.CriticalCount < 2 {
		t.Errorf("Expected at least 2 critical violations, got %d", insecureReport.CriticalCount)
	}
}

func TestCanaryObserver(t *testing.T) {
	obs := NewCanaryObserver(DefaultThresholds())

	// Healthy Telemetry Spans & Traces
	healthySpans := []OTELSpan{
		{TraceID: "t1", SpanID: "s1", ServiceName: "api", DurationMs: 120, StatusCode: "STATUS_OK"},
		{TraceID: "t2", SpanID: "s2", ServiceName: "api", DurationMs: 250, StatusCode: "STATUS_OK"},
		{TraceID: "t3", SpanID: "s3", ServiceName: "api", DurationMs: 310, StatusCode: "STATUS_OK"},
	}
	healthyTraces := []eBPFKernelTrace{
		{Syscall: "sys_enter_write", LatencyNs: 5000, DropCount: 0},
	}

	metrics, healthy := obs.EvaluateTelemetry(healthyTraces, healthySpans)
	if !healthy {
		t.Errorf("Expected healthy telemetry, got error rate %.2f%%, P99 %dms", metrics.ErrorRatePercentage, metrics.LatencyP99Ms)
	}

	// Unhealthy Telemetry (High Error Rate & High Latency)
	unhealthySpans := []OTELSpan{
		{TraceID: "t1", SpanID: "s1", ServiceName: "api", DurationMs: 850, StatusCode: "STATUS_ERROR", ErrorMessage: "500 Internal Error"},
		{TraceID: "t2", SpanID: "s2", ServiceName: "api", DurationMs: 920, StatusCode: "STATUS_ERROR", ErrorMessage: "500 Internal Error"},
		{TraceID: "t3", SpanID: "s3", ServiceName: "api", DurationMs: 150, StatusCode: "STATUS_OK"},
	}

	unhealthyMetrics, healthy := obs.EvaluateTelemetry(healthyTraces, unhealthySpans)
	if healthy {
		t.Errorf("Expected unhealthy telemetry to fail health check")
	}

	if unhealthyMetrics.ErrorRatePercentage < 50.0 {
		t.Errorf("Expected error rate > 50%%, got %.2f%%", unhealthyMetrics.ErrorRatePercentage)
	}
}

func TestDeploymentChoreographerSuccessPath(t *testing.T) {
	engine := NewIaCEngine()
	ch := engine.GetChoreographer()

	res := IaCResource{
		ID:       "r1",
		Name:     "app-service",
		Kind:     "Deployment",
		Provider: "kubernetes",
		Payload:  `kind: Deployment metadata: name: app-service spec: template: spec: containers: - name: app limits: memory: 256Mi`,
	}

	manifest, err := engine.SynthesizeManifest("prod-app", IaCTypeKubernetes, []IaCResource{res})
	if err != nil {
		t.Fatalf("Failed to synthesize manifest: %v", err)
	}

	backupManifest, _ := engine.SynthesizeManifest("prod-app-v0", IaCTypeKubernetes, []IaCResource{res})

	record, err := ch.CreateDeploymentRecord(*manifest, backupManifest)
	if err != nil {
		t.Fatalf("Failed to create deployment record: %v", err)
	}

	ctx := context.Background()

	// 1. Dry Run Verification
	err = ch.ExecuteDryRun(ctx, record.ID)
	if err != nil {
		t.Fatalf("Dry run failed: %v", err)
	}

	if record.CurrentState != StateDryRunVerified {
		t.Errorf("Expected state %s, got %s", StateDryRunVerified, record.CurrentState)
	}

	// 2. Canary Deployment & Promotion
	healthySpans := []OTELSpan{{TraceID: "t1", SpanID: "s1", ServiceName: "svc", DurationMs: 80, StatusCode: "STATUS_OK"}}
	err = ch.ExecuteCanaryDeploy(ctx, record.ID, nil, healthySpans)
	if err != nil {
		t.Fatalf("Canary deploy failed: %v", err)
	}

	if record.CurrentState != StateCommitted {
		t.Errorf("Expected state %s, got %s", StateCommitted, record.CurrentState)
	}
}

func TestDeploymentChoreographerAutomatedRollback(t *testing.T) {
	engine := NewIaCEngine()
	ch := engine.GetChoreographer()

	res := IaCResource{
		ID:       "r1",
		Name:     "app-service",
		Kind:     "Deployment",
		Provider: "kubernetes",
		Payload:  `kind: Deployment metadata: name: app-service spec: template: spec: containers: - name: app limits: memory: 256Mi`,
	}

	manifest, _ := engine.SynthesizeManifest("prod-app-v2", IaCTypeKubernetes, []IaCResource{res})
	backupManifest, _ := engine.SynthesizeManifest("prod-app-v1", IaCTypeKubernetes, []IaCResource{res})

	record, _ := ch.CreateDeploymentRecord(*manifest, backupManifest)
	ctx := context.Background()

	_ = ch.ExecuteDryRun(ctx, record.ID)

	// Inject Unhealthy Spans -> Triggers Automated Rollback
	unhealthySpans := []OTELSpan{
		{TraceID: "t1", SpanID: "s1", ServiceName: "svc", DurationMs: 1500, StatusCode: "STATUS_ERROR"},
		{TraceID: "t2", SpanID: "s2", ServiceName: "svc", DurationMs: 1200, StatusCode: "STATUS_ERROR"},
	}

	err := ch.ExecuteCanaryDeploy(ctx, record.ID, nil, unhealthySpans)
	if err == nil {
		t.Errorf("Expected canary deploy error due to telemetry failure and rollback")
	}

	if record.CurrentState != StateRolledBack {
		t.Errorf("Expected state %s after canary health failure, got %s", StateRolledBack, record.CurrentState)
	}
}
