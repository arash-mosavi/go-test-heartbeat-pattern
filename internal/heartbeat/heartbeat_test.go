package heartbeat

import (
	"context"
	"testing"
	"time"
)

func TestWorker_DoWork_SendsHeartbeats(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	worker := NewWorker(100 * time.Millisecond)
	_, heartbeat := worker.DoWork(ctx)

	heartbeatCount := 0
	timeout := time.After(300 * time.Millisecond)

	for heartbeatCount < 2 {
		select {
		case <-heartbeat:
			heartbeatCount++
		case <-timeout:
			t.Fatalf("Expected at least 2 heartbeats, got %d", heartbeatCount)
		case <-ctx.Done():
			t.Fatal("Context cancelled before receiving heartbeats")
		}
	}

	if heartbeatCount < 2 {
		t.Errorf("Expected at least 2 heartbeats, got %d", heartbeatCount)
	}
}

func TestWorker_DoWork_SendsResults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	worker := NewWorker(1 * time.Second)
	results, _ := worker.DoWork(ctx)

	select {
	case result, ok := <-results:
		if !ok {
			t.Fatal("Results channel closed unexpectedly")
		}
		if result != 0 {
			t.Errorf("Expected first result to be 0, got %d", result)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("Timeout waiting for result")
	case <-ctx.Done():
		t.Fatal("Context cancelled before receiving result")
	}
}

func TestWorker_DoWork_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	worker := NewWorker(100 * time.Millisecond)
	results, heartbeat := worker.DoWork(ctx)

	cancel()

	timeout := time.After(500 * time.Millisecond)

	select {
	case _, ok := <-results:
		if ok {

			for range results {
			}
		}
	case <-timeout:
		t.Fatal("Results channel did not close after context cancellation")
	}

	select {
	case _, ok := <-heartbeat:
		if ok {

			for range heartbeat {
			}
		}
	case <-timeout:
		t.Fatal("Heartbeat channel did not close after context cancellation")
	}
}

func TestWorker_DoWork_IncrementingResults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	worker := NewWorker(2 * time.Second)
	results, _ := worker.DoWork(ctx)

	receivedResults := make([]int, 0, 3)

	for len(receivedResults) < 3 {
		select {
		case result, ok := <-results:
			if !ok {
				t.Fatalf("Results channel closed unexpectedly after %d results", len(receivedResults))
			}
			receivedResults = append(receivedResults, result)
		case <-time.After(1 * time.Second):
			t.Fatalf("Timeout waiting for result, got %d results so far", len(receivedResults))
		case <-ctx.Done():
			t.Fatal("Context cancelled before receiving all results")
		}
	}

	for i, result := range receivedResults {
		if result != i {
			t.Errorf("Expected result %d, got %d", i, result)
		}
	}
}

func TestMonitor_Watch_DetectsHealthyHeartbeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	heartbeat := make(chan struct{})
	monitor := NewMonitor(500 * time.Millisecond)
	errors := monitor.Watch(ctx, heartbeat)

	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				close(heartbeat)
				return
			case <-ticker.C:
				select {
				case heartbeat <- struct{}{}:
				case <-ctx.Done():
					close(heartbeat)
					return
				}
			}
		}
	}()

	select {
	case err := <-errors:
		if err != nil {
			t.Errorf("Unexpected error from monitor: %v", err)
		}
	case <-time.After(1 * time.Second):

	}
}

func TestMonitor_Watch_DetectsTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	heartbeat := make(chan struct{})
	monitor := NewMonitor(200 * time.Millisecond)
	errors := monitor.Watch(ctx, heartbeat)

	select {
	case err := <-errors:
		if err == nil {
			t.Error("Expected timeout error, got nil")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Monitor did not detect timeout")
	case <-ctx.Done():
		t.Fatal("Context cancelled before monitor timeout")
	}

	close(heartbeat)
}

func TestMonitor_Watch_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	heartbeat := make(chan struct{})
	monitor := NewMonitor(1 * time.Second)
	errors := monitor.Watch(ctx, heartbeat)

	cancel()

	timeout := time.After(500 * time.Millisecond)
	select {
	case _, ok := <-errors:
		if ok {

			for range errors {
			}
		}
	case <-timeout:
		t.Fatal("Error channel did not close after context cancellation")
	}

	close(heartbeat)
}

func TestIntegration_WorkerAndMonitor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	worker := NewWorker(100 * time.Millisecond)
	results, heartbeat := worker.DoWork(ctx)

	monitor := NewMonitor(500 * time.Millisecond)
	errors := monitor.Watch(ctx, heartbeat)

	resultCount := 0
	errorReceived := false

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case result, ok := <-results:
				if !ok {
					return
				}
				if result >= 0 {
					resultCount++
				}
			case err := <-errors:
				if err != nil {
					errorReceived = true
					t.Logf("Received error: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(4 * time.Second):
		t.Fatal("Test did not complete in time")
	}

	if resultCount == 0 {
		t.Error("Expected to receive results, got 0")
	}

	if errorReceived {
		t.Error("Did not expect to receive errors from monitor")
	}
}

func TestNewWorker(t *testing.T) {
	interval := 1 * time.Second
	worker := NewWorker(interval)

	if worker == nil {
		t.Fatal("NewWorker returned nil")
	}

	if worker.heartbeatInterval != interval {
		t.Errorf("Expected heartbeat interval %v, got %v", interval, worker.heartbeatInterval)
	}
}

func TestNewMonitor(t *testing.T) {
	timeout := 2 * time.Second
	monitor := NewMonitor(timeout)

	if monitor == nil {
		t.Fatal("NewMonitor returned nil")
	}

	if monitor.timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, monitor.timeout)
	}
}
