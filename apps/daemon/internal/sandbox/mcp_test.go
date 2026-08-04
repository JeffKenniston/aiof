package sandbox

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMCPContainer_ConcurrentDispatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	// Create an MCP container using the host runtime to test PTY/stdio logic.
	container := NewMCPContainer(logger, RuntimeHost, ".")
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := container.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}
	defer container.Stop()

	var wg sync.WaitGroup
	numRequests := 5
	
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			
			// Execute a simple echo command to verify concurrent handling.
			// The Python script in RuntimeHost executes the code via a shell.
			code := []byte("python -c \"print('hello')\"")
			result, err := container.ExecuteCode(ctx, code)
			if err != nil {
				t.Errorf("ExecuteCode failed for request %d: %v", id, err)
			}
			if !strings.Contains(result, "hello") {
				t.Errorf("Expected result to contain 'hello', got '%s'", result)
			}
		}(i)
	}
	
	wg.Wait()
}
