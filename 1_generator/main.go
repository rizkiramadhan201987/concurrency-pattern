package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

func main() {

	// Setting up context to prevent goroutine leaks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cleanup

	for num := range generator(ctx, 10) {
		fmt.Println("receive value : ", num)
		if num == 5 {
			cancel() // Stop generator early
			break
		}
	}
}

func generator(ctx context.Context, max int) <-chan int {
	generatorChan := make(chan int)
	go func() {
		defer close(generatorChan)
		for i := range max {
			time.Sleep(1 * time.Second) // to simulate process
			value := i + 1
			select {
			case generatorChan <- value:
				log.Println("Sent value : ", value)
			case <-ctx.Done():
				// Context cancelled, exit cleanly
				return
			}
		}
	}()
	return generatorChan
}
