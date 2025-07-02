package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func main() {
	fmt.Println("=== Goroutine Leak Demo ===\n")

	baseline := runtime.NumGoroutine()
	fmt.Printf("🏁 Baseline goroutines: %d\n\n", baseline)

	// Demo 1: Show proper cleanup FIRST
	fmt.Println("1. Demonstrating PROPER CLEANUP:")
	properCount := demoProperCleanupWithCount(baseline)

	time.Sleep(2 * time.Second) // Give time for goroutines to settle

	// Demo 2: Show goroutine leak
	fmt.Println("\n2. Demonstrating GOROUTINE LEAK:")
	leakyCount := demoGoroutineLeakWithCount(baseline)

	time.Sleep(2 * time.Second) // Give time for goroutines to settle

	// Demo 3: Multiple leaky generators
	fmt.Println("\n3. Demonstrating MULTIPLE LEAKS:")
	multiCount := demoMultipleLeaksWithCount(baseline)

	time.Sleep(2 * time.Second)

	final := runtime.NumGoroutine()
	fmt.Printf("\n📊 SUMMARY:\n")
	fmt.Printf("Baseline: %d\n", baseline)
	fmt.Printf("After proper cleanup: %d (difference: %+d)\n", properCount, properCount-baseline)
	fmt.Printf("After single leak: %d (difference: %+d)\n", leakyCount, leakyCount-baseline)
	fmt.Printf("After multiple leaks: %d (difference: %+d)\n", multiCount, multiCount-baseline)
	fmt.Printf("Final count: %d (total leaked: %+d)\n", final, final-baseline)

	fmt.Println("\nPress Ctrl+C to exit and see final cleanup...")
	waitForAnyEvent()
}

// Demo 1: Proper generator with context
func demoProperCleanupWithCount(baseline int) int {
	before := runtime.NumGoroutine()
	fmt.Printf("Before proper cleanup: %d goroutines (vs baseline: %+d)\n", before, before-baseline)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cleanup

	for num := range properGenerator(ctx, 10) {
		fmt.Printf("Received value: %d\n", num)
		if num == 3 {
			fmt.Println("Breaking early - but with proper cleanup!")
			cancel() // This will signal the generator to stop
			break
		}
	}

	time.Sleep(200 * time.Millisecond) // Give time for cleanup
	after := runtime.NumGoroutine()
	fmt.Printf("After proper cleanup: %d goroutines (vs baseline: %+d)\n", after, after-baseline)
	fmt.Println("✅ No leak! Goroutine exited cleanly")
	return after
}

// Demo 2: Leaky generator (WITHOUT context handling)
func demoGoroutineLeakWithCount(baseline int) int {
	before := runtime.NumGoroutine()
	fmt.Printf("Before leaky function: %d goroutines (vs baseline: %+d)\n", before, before-baseline)

	// Create leaky generator
	for num := range leakyGenerator(10) {
		fmt.Printf("Received value: %d\n", num)
		if num == 3 {
			fmt.Println("Breaking early - this will cause a leak!")
			break // Generator goroutine will be stuck trying to send to channel
		}
	}

	time.Sleep(200 * time.Millisecond) // Give time for goroutine to get stuck
	after := runtime.NumGoroutine()
	fmt.Printf("After leaky function: %d goroutines (vs baseline: %+d)\n", after, after-baseline)
	fmt.Println("☠️  LEAK DETECTED! Goroutine is stuck trying to send value 4")
	return after
}

// Demo 3: Multiple leaky generators
func demoMultipleLeaksWithCount(baseline int) int {
	before := runtime.NumGoroutine()
	fmt.Printf("Before multiple leaks: %d goroutines (vs baseline: %+d)\n", before, before-baseline)

	// Create multiple leaky generators
	for i := 0; i < 5; i++ {
		go func(id int) {
			for num := range leakyGenerator(100) {
				if num == 2 {
					fmt.Printf("Generator %d: Breaking early\n", id)
					return // Causes leak
				}
			}
		}(i)
	}

	time.Sleep(500 * time.Millisecond) // Give time for leaks to accumulate
	after := runtime.NumGoroutine()
	fmt.Printf("After multiple leaks: %d goroutines (vs baseline: %+d)\n", after, after-baseline)
	fmt.Println("☠️  MULTIPLE LEAKS! Each generator left a stuck goroutine")
	return after
}

// LEAKY generator - does NOT handle early termination
func leakyGenerator(max int) <-chan int {
	generatorChan := make(chan int)
	go func() {
		defer close(generatorChan)
		defer fmt.Println("🔴 Leaky generator goroutine exiting") // This won't print if stuck

		for i := 1; i <= max; i++ {
			time.Sleep(200 * time.Millisecond)

			// This will BLOCK if no receiver - causing the leak!
			generatorChan <- i
			fmt.Printf("Leaky generator sent: %d\n", i)
		}
	}()
	return generatorChan
}

// PROPER generator - handles context cancellation
func properGenerator(ctx context.Context, max int) <-chan int {
	generatorChan := make(chan int)
	go func() {
		defer close(generatorChan)
		defer fmt.Println("✅ Proper generator goroutine exiting cleanly")

		for i := 1; i <= max; i++ {
			time.Sleep(200 * time.Millisecond)

			select {
			case generatorChan <- i:
				fmt.Printf("Proper generator sent: %d\n", i)
			case <-ctx.Done():
				fmt.Println("🛑 Proper generator received cancellation signal")
				return // Exit cleanly when context is cancelled
			}
		}
	}()
	return generatorChan
}

// Additional demo: Generator that times out
func timeoutGenerator(max int, timeout time.Duration) <-chan int {
	generatorChan := make(chan int)
	go func() {
		defer close(generatorChan)
		defer fmt.Println("⏰ Timeout generator exiting")

		for i := 1; i <= max; i++ {
			timer := time.NewTimer(timeout)
			select {
			case generatorChan <- i:
				fmt.Printf("Timeout generator sent: %d\n", i)
				timer.Stop()
			case <-timer.C:
				fmt.Println("⏰ Generator timed out - assuming receiver is gone")
				return
			}
		}
	}()
	return generatorChan
}

// Wait for interrupt signal
func waitForAnyEvent() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case <-interrupt:
		fmt.Println("\nReceived interrupt signal")
		fmt.Printf("Final goroutine count before exit: %d\n", runtime.NumGoroutine())
	}
}

// Utility function to monitor goroutines
func monitorGoroutines(label string) {
	go func() {
		for {
			fmt.Printf("[%s] Current goroutines: %d\n", label, runtime.NumGoroutine())
			time.Sleep(1 * time.Second)
		}
	}()
}
