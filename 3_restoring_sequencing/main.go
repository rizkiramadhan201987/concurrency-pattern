package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Message struct {
	Ordering int
	Value    string
}

type MessageResult struct {
	Ordering int
	Result   string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cleanup
	// Define the tasks
	numberOfTasks := 10
	resultChan := make(chan *MessageResult, numberOfTasks)
	printerChan := make(chan *MessageResult, numberOfTasks)
	// Proccess the task
	go func() {
		defer close(resultChan)
		for message := range processMessage(ctx, numberOfTasks) {
			resultChan <- message
		}
	}()
	go func() {
		defer func() {
			fmt.Println("Closing printerChan")
			close(printerChan)
		}()
		buffer := make(map[int]*MessageResult)
		nextExpected := 0

		for result := range resultChan {
			fmt.Printf("Resequencer received: Task %d\n", result.Ordering)
			if result.Ordering == nextExpected {
				fmt.Printf("Sending Task %d (expected)\n", result.Ordering)
				printerChan <- result
				nextExpected++

				// Check buffer
				for {
					if bufferedResult, exists := buffer[nextExpected]; exists {
						fmt.Printf("Sending Task %d (from buffer)\n", bufferedResult.Ordering)
						printerChan <- bufferedResult
						delete(buffer, nextExpected)
						nextExpected++
					} else {
						break
					}
				}
			} else {
				fmt.Printf("Buffering Task %d (expected %d)\n", result.Ordering, nextExpected)
				buffer[result.Ordering] = result
			}
		}
		fmt.Printf("Resequencer finished, buffer: %v\n", buffer)
	}()
	for result := range printerChan {
		fmt.Printf("Received in order: %s\n", result.Result)
	}

	// Keep program running until Ctrl+C
	fmt.Println("All tasks completed. Press Ctrl+C to exit...")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Received quit signal, exiting...")

}

func createMessage(numberTask int) *Message {
	return &Message{
		Ordering: numberTask,
		Value:    fmt.Sprintf("TASK %d", numberTask),
	}
}

func processMessage(
	ctx context.Context,
	numberOfTask int) <-chan *MessageResult {
	// Buffered channel acts as semaphore - limits to 5 concurrent workers
	semaphoreChan := make(chan struct{}, 5)
	messageResultChan := make(chan *MessageResult)
	doneChan := make(chan struct{}, numberOfTask)

	go func() {
		defer close(messageResultChan) // Close when this goroutine exits
		for numberTask := 0; numberTask < numberOfTask; numberTask++ {
			// Acquire semaphore
			semaphoreChan <- struct{}{}
			// message creation
			go func(numberTask int) {
				defer func() {
					<-semaphoreChan
					doneChan <- struct{}{} // Signal completion
				}() // Release semaphore when done
				message := createMessage(numberTask)
				time.Sleep(time.Millisecond * time.Duration(message.Ordering))
				data := &MessageResult{
					Ordering: message.Ordering,
					Result:   fmt.Sprintf("FINISH TASK %d", message.Ordering),
				}
				select {
				case messageResultChan <- data:
				case <-ctx.Done():
					// Context cancelled, exit cleanly
					return
				}
			}(numberTask)
		}
		// Wait for all workers to complete
		for i := 0; i < numberOfTask; i++ {
			<-doneChan
		}
	}()
	return messageResultChan
}
