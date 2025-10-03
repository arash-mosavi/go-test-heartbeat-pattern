# Go Heartbeat Pattern Demo

A demonstration of the heartbeat pattern in Go, implementing best practices for monitoring long-running goroutines.

## Overview

The heartbeat pattern allows you to monitor the health of concurrent operations by sending periodic signals. This implementation includes:

- **Worker with heartbeat**: A goroutine that performs work while sending periodic heartbeat signals
- **Health monitoring**: A monitor that detects when workers become unresponsive
- **Context-aware cancellation**: Proper cleanup and graceful shutdown using Go contexts
- **Non-blocking communication**: Safe channel operations that don't deadlock

## Project Structure

```
.
├── cmd/
│   └── heartbeat/
│       └── main.go              # Application entry point
├── internal/
│   └── heartbeat/
│       ├── heartbeat.go         # Core heartbeat implementation
│       └── heartbeat_test.go    # Comprehensive test suite
├── pkg/                         # Shared packages (for future use)
├── .gitignore
├── go.mod
└── README.md
```

## How It Works

### 1. Worker Component

The `Worker` type performs work while sending heartbeats at regular intervals:

- Returns two channels: one for results, one for heartbeats
- Uses separate tickers for heartbeats (1s) and work (500ms)
- Respects context cancellation for graceful shutdown
- Non-blocking channel sends prevent deadlocks

```go
worker := heartbeat.NewWorker(1 * time.Second)
results, heartbeatChan := worker.DoWork(ctx)
```

### 2. Monitor Component

The `Monitor` type watches for heartbeats and detects timeouts:

- Uses a timer to detect when heartbeats stop arriving
- Sends errors when workers become unresponsive
- Automatically stops when context is cancelled
- Configurable timeout duration

```go
monitor := heartbeat.NewMonitor(3 * time.Second)
errors := monitor.Watch(ctx, heartbeatChan)
```

### 3. Main Coordination Loop

The main loop coordinates results, errors, and cancellation:

- Processes work results as they arrive
- Handles monitor errors and triggers shutdown
- Ensures clean termination via context

## Usage

### Run the Application

```bash
# Run the demo
go run cmd/heartbeat/main.go
```

### Run Tests

```bash
# Run all tests with verbose output
go test ./internal/heartbeat/... -v

# Run tests with coverage
go test ./internal/heartbeat/... -cover

# Run specific test
go test ./internal/heartbeat/... -run TestWorker_DoWork_SendsHeartbeats -v
```

## Configuration

The demo uses these default settings:

- **Heartbeat interval**: 1 second
- **Monitor timeout**: 3 seconds (triggers error if no heartbeat)
- **Total runtime**: 10 seconds (context timeout)
- **Work interval**: 500ms per result

Adjust these in `cmd/heartbeat/main.go` to experiment with different scenarios.


## Best Practices Demonstrated

1. **Separation of concerns**: Worker and Monitor are independent components
2. **Context usage**: Proper propagation for cancellation and timeouts
3. **Channel patterns**: Non-blocking sends with select statements
4. **Resource cleanup**: Deferred channel closes and ticker stops
5. **Testability**: Components designed for easy testing
6. **Project layout**: Standard Go project structure with cmd/internal separation

