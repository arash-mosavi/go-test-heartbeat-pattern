package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arash-mosavi/go-test-heartbeat-pattern.git/internal/heartbeat"
)

func main() {
	fmt.Println("Heartbeat Pattern\n")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	worker := heartbeat.NewWorker(1 * time.Second)
	results, heartbeatChan := worker.DoWork(ctx)

	monitor := heartbeat.NewMonitor(3 * time.Second)
	errors := monitor.Watch(ctx, heartbeatChan)

	for {
		select {
		case result, ok := <-results:
			if !ok {
				fmt.Println("\nWorker finished")
				return
			}
			fmt.Printf("Result: %d\n", result)

		case err := <-errors:
			if err != nil {
				log.Printf("ERROR: %v\n", err)
				cancel()
				return
			}

		case <-ctx.Done():
			fmt.Println("\nContext cancelled, shutting down...")
			return
		}
	}
}
