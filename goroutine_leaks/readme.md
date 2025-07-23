Goroutine leaks occur when goroutines are created but never properly terminated, causing them to accumulate in memory and consume system resources indefinitely. This is similar to memory leaks in other programming languages, but specifically involves Go's lightweight threads.
Common Causes
Blocked goroutines are the most frequent culprit. This happens when a goroutine is waiting for an operation that never completes, such as:

Reading from a channel that no other goroutine will ever write to
Writing to a channel that no other goroutine will ever read from
Waiting on a mutex, condition variable, or other synchronization primitive indefinitely

Missing termination conditions occur when goroutines are designed to run indefinitely without proper shutdown mechanisms. For example, a worker goroutine that processes tasks but has no way to receive a "stop" signal.
Context misuse can also lead to leaks when goroutines don't properly respect context cancellation or timeouts.

Detection and Prevention
You can detect goroutine leaks by monitoring runtime.NumGoroutine() or using tools like go tool trace. To prevent leaks, always ensure goroutines have clear termination paths, use contexts for cancellation, properly close channels, and implement timeouts for potentially blocking operations.
The key is designing your concurrent code so that every goroutine has a guaranteed way to exit, either through completing its work or receiving a cancellation signal.