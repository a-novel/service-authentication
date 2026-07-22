package lib

import (
	"context"
	"fmt"
)

// Waiter blocks until the work it owns has finished. The short-code services implement it over the
// detached goroutines that deliver their emails.
type Waiter interface {
	Wait()
}

// Drain waits for every waiter to finish, or for ctx to expire.
//
// Mail delivery is detached from the request that triggered it — the handler answers 202 and the
// send continues on its own goroutine, so a cancelled request does not cancel a half-sent email.
// The cost is that process exit kills whatever is still in flight, and nothing reports it: the 202
// went out long ago, the goroutine's error log sits after the call that never returned, and the
// database holds a perfectly valid unconsumed short code. The user is told to check their inbox and
// never receives anything.
//
// The bound matters as much as the wait. Each send carries its own timeout, so the drain terminates
// on its own; the context is the second belt, so a misconfigured sender delays a deploy rather than
// blocking it forever. Draining against an unbounded send would trade dropped mail for a hung
// shutdown, which is the worse failure for a rolling restart.
func Drain(ctx context.Context, waiters ...Waiter) error {
	done := make(chan struct{})

	go func() {
		defer close(done)

		for _, waiter := range waiters {
			waiter.Wait()
		}
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		// The goroutine above outlives this return. That is deliberate: it is parked on sends that
		// are themselves bounded, and the process is exiting.
		return fmt.Errorf("drain in-flight work: %w", ctx.Err())
	}
}
