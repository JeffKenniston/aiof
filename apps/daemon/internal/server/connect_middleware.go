package server

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"runtime/debug"
	"sync/atomic"
	"time"
)

// WALWriter interface for persisting streaming token streams to Write-Ahead Log per ADR-05.
type WALWriter interface {
	WriteWAL(ctx context.Context, traceID string, payload []byte) error
}

// HandlerFunc is the core request processing function type for unary RPC endpoints.
type HandlerFunc func(ctx context.Context, rawReq []byte) ([]byte, error)

// Middleware decorates a HandlerFunc with cross-cutting concerns (logging, WAL, recovery).
type Middleware func(next HandlerFunc) HandlerFunc

// StreamChunk represents a typed chunk yielded during RPC streaming iterations.
type StreamChunk[T any] struct {
	Sequence  uint64
	Timestamp int64
	Data      *T
	Err       error
}

// TwoPassValidationMiddleware returns a middleware enforcing ADR-05 Two-Pass Deserialization & max depth validation.
func TwoPassValidationMiddleware(deserializer *TwoPassDeserializer) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, rawReq []byte) ([]byte, error) {
			if err := deserializer.CheckDepth(rawReq); err != nil {
				return nil, fmt.Errorf("connect middleware validation failed: %w", err)
			}
			return next(ctx, rawReq)
		}
	}
}

// PanicRecoveryMiddleware catches panics during RPC handler execution and returns a structured error.
func PanicRecoveryMiddleware(logger *slog.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, rawReq []byte) (resp []byte, err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					if logger != nil {
						logger.Error("Connect-RPC handler panic recovered",
							slog.Any("panic", r),
							slog.String("stack", string(stack)),
						)
					}
					err = fmt.Errorf("rpc handler panic recovered: %v", r)
				}
			}()
			return next(ctx, rawReq)
		}
	}
}

// WALLoggingMiddleware persists raw request bytes to the PostgreSQL Write-Ahead Log before response transmission (ADR-05).
func WALLoggingMiddleware(wal WALWriter, logger *slog.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, rawReq []byte) ([]byte, error) {
			start := time.Now()

			// Extract trace ID from context or assign fallback
			traceID, ok := ctx.Value("trace_id").(string)
			if !ok || traceID == "" {
				traceID = fmt.Sprintf("trace-%d", time.Now().UnixNano())
			}

			// Persist to WAL prior to processing
			if wal != nil {
				if err := wal.WriteWAL(ctx, traceID, rawReq); err != nil {
					if logger != nil {
						logger.Warn("Failed to persist request to WAL", slog.String("trace_id", traceID), slog.Any("error", err))
					}
				}
			}

			resp, err := next(ctx, rawReq)

			if logger != nil {
				logger.Info("Connect-RPC invocation complete",
					slog.String("trace_id", traceID),
					slog.Duration("duration", time.Since(start)),
					slog.Bool("success", err == nil),
				)
			}

			return resp, err
		}
	}
}

// CreateStreamAdapter transforms a channel pair into a Go 1.23 iter.Seq2[*T, error] streaming iterator.
// This supports Go 1.23 standard range-over-function syntax for token streams:
//
//	for chunk, err := range CreateStreamAdapter(ctx, ch, errCh) { ... }
func CreateStreamAdapter[T any](ctx context.Context, dataCh <-chan *T, errCh <-chan error) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		var seq uint64
		for {
			select {
			case <-ctx.Done():
				yield(nil, ctx.Err())
				return
			case err, ok := <-errCh:
				if ok && err != nil {
					yield(nil, err)
					return
				}
			case item, ok := <-dataCh:
				if !ok {
					return // Channel closed normally
				}
				atomic.AddUint64(&seq, 1)
				if !yield(item, nil) {
					return // Downstream consumer aborted iteration
				}
			}
		}
	}
}

// SliceToSeq2 converts a slice of items into a Go 1.23 iter.Seq2[*T, error] iterator.
func SliceToSeq2[T any](items []*T) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

// TransformSeq2 applies a transformation function to an incoming iter.Seq2 stream per ADR-01.
func TransformSeq2[In any, Out any](
	source iter.Seq2[*In, error],
	transform func(*In) (*Out, error),
) iter.Seq2[*Out, error] {
	return func(yield func(*Out, error) bool) {
		for in, err := range source {
			if err != nil {
				yield(nil, err)
				return
			}
			out, txErr := transform(in)
			if txErr != nil {
				yield(nil, txErr)
				return
			}
			if !yield(out, nil) {
				return
			}
		}
	}
}

// CollectSeq2 drains a Go 1.23 iter.Seq2 stream into a slice.
func CollectSeq2[T any](seq iter.Seq2[*T, error]) ([]*T, error) {
	var results []*T
	for item, err := range seq {
		if err != nil {
			return results, err
		}
		if item != nil {
			results = append(results, item)
		}
	}
	return results, nil
}
