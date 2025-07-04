package main

import (
	"context"
	"log"
)

func main() {
	// Setting up context to prevent goroutine leaks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cleanup
	gen1 := generator(ctx, 10)
	gen2 := generator(ctx, 20)
	result := multiplexConsumer(ctx, gen1, gen2)

	for range 10 {
		log.Println("Got : ", <-result)
	}
}

func generator(ctx context.Context, maxNumber int) <-chan int {
	generatorChan := make(chan int)
	go func() {
		defer close(generatorChan)
		for i := range maxNumber {
			select {
			case generatorChan <- i:
				log.Println("Successfully sent to generator chan : ", i)
			case <-ctx.Done():
				return
			}

		}
	}()
	return generatorChan
}

func multiplexConsumer(ctx context.Context, gen1, gen2 <-chan int) <-chan int {
	multiplexChan := make(chan int)
	go func() {
		defer close(multiplexChan)
		for {
			select {
			case value, ok := <-gen1: // Fix 2: Check if channel is open
				if !ok {
					gen1 = nil // Disable this case when channel closes
				} else {
					multiplexChan <- value
					log.Println("From gen1:", value)
				}
			case value, ok := <-gen2:
				if !ok {
					gen2 = nil // Disable this case when channel closes
				} else {
					multiplexChan <- value
					log.Println("From gen2:", value)
				}
			case <-ctx.Done():
				return
			}
			// Fix 3: Exit when both channels are closed
			if gen1 == nil && gen2 == nil {
				break
			}
		}
	}()
	return multiplexChan
}
