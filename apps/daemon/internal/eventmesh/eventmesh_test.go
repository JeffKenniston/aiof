package eventmesh

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTopicMatching(t *testing.T) {
	tests := []struct {
		pattern  string
		topic    string
		expected bool
	}{
		{"code.parsed", "code.parsed", true},
		{"code.*", "code.parsed", true},
		{"code.*", "code.compiled", true},
		{"code.*", "code.parsed.ast", false},
		{"code.>", "code.parsed.ast", true},
		{"vulnerability.>", "vulnerability.detected.critical", true},
		{">", "anything.at.all", true},
		{"agent.*.status", "agent.123.status", true},
		{"agent.*.status", "agent.123.error", false},
	}

	for _, tt := range tests {
		got := matchTopic(tt.pattern, tt.topic)
		if got != tt.expected {
			t.Errorf("matchTopic(%q, %q) = %v; want %v", tt.pattern, tt.topic, got, tt.expected)
		}
	}
}

func TestPubSubBasic(t *testing.T) {
	mesh := NewEventMesh(1024, 100)
	defer mesh.Close()

	var receivedCount atomic.Uint64
	var lastEvent atomic.Pointer[Event]

	handler := func(ctx context.Context, ev *Event) error {
		receivedCount.Add(1)
		lastEvent.Store(ev)
		return nil
	}

	sub, err := mesh.Subscribe("code.parsed", handler)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}
	defer mesh.Unsubscribe(sub.id)

	evt := &Event{
		Topic:    "code.parsed",
		Priority: PriorityDefault,
		Payload:  "test_payload",
	}

	err = mesh.Publish(evt)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if receivedCount.Load() != 1 {
		t.Fatalf("expected 1 event received, got %d", receivedCount.Load())
	}

	recv := lastEvent.Load()
	if recv == nil || recv.Payload != "test_payload" {
		t.Fatalf("unexpected payload received: %v", recv)
	}
}

func TestTailDropLoadShedding(t *testing.T) {
	// Small buffer of capacity 16 to quickly trigger load shedding
	mesh := NewEventMesh(16, 50)
	defer mesh.Close()

	// Fill buffer to > 85% capacity with Default events
	for i := 0; i < 14; i++ {
		_ = mesh.Publish(&Event{
			Topic:    "telemetry.metrics",
			Priority: PriorityDefault,
			Payload:  fmt.Sprintf("item_%d", i),
		})
	}

	// Next telemetry event should be dropped (fill ratio > 70%)
	errTel := mesh.Publish(&Event{
		Topic:    "telemetry.metrics",
		Priority: PriorityTelemetry,
		Payload:  "dropped_telemetry",
	})

	if !errors.Is(errTel, ErrEventDropped) {
		t.Errorf("expected ErrEventDropped for telemetry, got %v", errTel)
	}

	// Critical event should still be accepted
	errCrit := mesh.Publish(&Event{
		Topic:    "system.alert",
		Priority: PriorityCritical,
		Payload:  "critical_alert",
	})

	if errCrit != nil {
		t.Errorf("expected Critical event to be accepted, got %v", errCrit)
	}
}

func TestDLQAndRetry(t *testing.T) {
	mesh := NewEventMesh(1024, 100)
	defer mesh.Close()

	handler := func(ctx context.Context, ev *Event) error {
		return errors.New("simulated handler failure")
	}

	_, err := mesh.Subscribe("build.failed", handler, WithMaxRetries(2))
	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	evt := &Event{
		Topic:    "build.failed",
		Priority: PriorityHigh,
		Payload:  "build_id_999",
	}

	_ = mesh.Publish(evt)

	time.Sleep(200 * time.Millisecond)

	dlqEvents := mesh.GetDLQEvents("build.failed")
	if len(dlqEvents) != 1 {
		t.Fatalf("expected 1 event in DLQ, got %d", len(dlqEvents))
	}

	if dlqEvents[0].RetryCount <= 2 {
		t.Errorf("expected retries > 2, got %d", dlqEvents[0].RetryCount)
	}

	// Test Go 1.23 Iterator over DLQ
	countIter := 0
	for item := range mesh.AllDLQ() {
		if item.ID == dlqEvents[0].ID {
			countIter++
		}
	}
	if countIter != 1 {
		t.Errorf("expected 1 iterator match for DLQ item, got %d", countIter)
	}
}

func TestConcurrentPublishing(t *testing.T) {
	mesh := NewEventMesh(4096, 500)
	defer mesh.Close()

	var received atomic.Uint64
	_, err := mesh.Subscribe("data.>", func(ctx context.Context, ev *Event) error {
		received.Add(1)
		return nil
	}, WithWorkerCount(8))

	if err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	var wg sync.WaitGroup
	numProducers := 10
	pubPerProducer := 100

	for i := 0; i < numProducers; i++ {
		wg.Add(1)
		go func(pID int) {
			defer wg.Done()
			for j := 0; j < pubPerProducer; j++ {
				_ = mesh.Publish(&Event{
					Topic:    "data.stream",
					Priority: PriorityHigh,
					Payload:  j,
				})
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(300 * time.Millisecond)

	metrics := mesh.GetMetrics()
	if metrics["published_events"] != uint64(numProducers*pubPerProducer) {
		t.Errorf("expected %d published events, got %d", numProducers*pubPerProducer, metrics["published_events"])
	}
}
