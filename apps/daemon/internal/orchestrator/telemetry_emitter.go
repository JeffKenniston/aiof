package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type TelemetryData struct {
	ID        string  `json:"id"`
	Time      string  `json:"time"`
	ReqPerSec float64 `json:"reqPerSec"`
	ErrorPct  float64 `json:"errorPct"`
	P50       float64 `json:"p50"`
	P95       float64 `json:"p95"`
	P99       float64 `json:"p99"`
	Timestamp int64   `json:"timestamp"`
}

type TelemetryEmitter struct {
	o            *Orchestrator
	mu           sync.Mutex
	latencies    []float64
	reqCount     int
	errCount     int
	lastEmitTime time.Time
}

func NewTelemetryEmitter(o *Orchestrator) *TelemetryEmitter {
	return &TelemetryEmitter{
		o:            o,
		latencies:    make([]float64, 0),
		lastEmitTime: time.Now(),
	}
}

func (te *TelemetryEmitter) RecordRequest(latencyMs float64, isError bool) {
	te.mu.Lock()
	defer te.mu.Unlock()
	te.latencies = append(te.latencies, latencyMs)
	te.reqCount++
	if isError {
		te.errCount++
	}
}

func (te *TelemetryEmitter) Start(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			te.emit(ctx, now)
		}
	}
}

func (te *TelemetryEmitter) emit(ctx context.Context, now time.Time) {
	te.mu.Lock()
	
	elapsed := now.Sub(te.lastEmitTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1
	}

	reqs := float64(te.reqCount)
	errs := float64(te.errCount)
	reqPerSec := reqs / elapsed
	errorPct := 0.0
	if reqs > 0 {
		errorPct = (errs / reqs) * 100
	}

	var p50, p95, p99 float64
	if len(te.latencies) > 0 {
		// Calculate percentiles
		sort.Float64s(te.latencies)
		p50 = percentile(te.latencies, 50)
		p95 = percentile(te.latencies, 95)
		p99 = percentile(te.latencies, 99)
	}

	// Reset window
	te.latencies = te.latencies[:0]
	te.reqCount = 0
	te.errCount = 0
	te.lastEmitTime = now
	te.mu.Unlock()

	data := TelemetryData{
		ID:        fmt.Sprintf("tel-%d", now.UnixNano()),
		Time:      now.Format("15:04:05"),
		ReqPerSec: math.Round(reqPerSec*10) / 10,
		ErrorPct:  math.Round(errorPct*100) / 100,
		P50:       math.Round(p50),
		P95:       math.Round(p95),
		P99:       math.Round(p99),
		Timestamp: now.UnixMilli(),
	}

	payload, err := json.Marshal(data)
	if err == nil {
		te.o.emitState(ctx, "telemetry", data.ID, string(payload))
	}
}

func percentile(sorted []float64, pct float64) float64 {
	index := (pct / 100.0) * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	weight := index - float64(lower)

	if lower < 0 {
		return sorted[0]
	}
	if upper >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	if lower == upper {
		return sorted[lower]
	}

	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
