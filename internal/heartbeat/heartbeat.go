package heartbeat

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Worker struct {
	heartbeatInterval time.Duration
}

func NewWorker(heartbeatInterval time.Duration) *Worker {
	return &Worker{
		heartbeatInterval: heartbeatInterval,
	}
}

func (w *Worker) DoWork(ctx context.Context) (<-chan int, <-chan struct{}) {
	heartbeat := make(chan struct{})
	results := make(chan int)

	go func() {
		defer close(heartbeat)
		defer close(results)

		heartbeatTicker := time.NewTicker(w.heartbeatInterval)
		defer heartbeatTicker.Stop()

		workTicker := time.NewTicker(500 * time.Millisecond)
		defer workTicker.Stop()

		workCounter := 0

		for {
			select {
			case <-ctx.Done():
				log.Println("Worker: context cancelled, stopping work")
				return

			case <-heartbeatTicker.C:
				select {
				case heartbeat <- struct{}{}:
				case <-ctx.Done():
					return
				}

			case <-workTicker.C:
				select {
				case <-ctx.Done():
					return
				case results <- workCounter:
					workCounter++
				}
			}
		}
	}()

	return results, heartbeat
}

type Monitor struct {
	timeout time.Duration
}

func NewMonitor(timeout time.Duration) *Monitor {
	return &Monitor{
		timeout: timeout,
	}
}

func (m *Monitor) Watch(ctx context.Context, heartbeat <-chan struct{}) <-chan error {
	errors := make(chan error)

	go func() {
		defer close(errors)

		timer := time.NewTimer(m.timeout)
		defer timer.Stop()

		for {
			timer.Reset(m.timeout)

			select {
			case <-ctx.Done():
				log.Println("Monitor: context cancelled, stopping monitoring")
				return

			case <-heartbeat:
				log.Println("Monitor: heartbeat received")

			case <-timer.C:
				select {
				case errors <- fmt.Errorf("worker unresponsive: no heartbeat received within %v", m.timeout):
				case <-ctx.Done():
					return
				}
				return
			}
		}
	}()

	return errors
}
