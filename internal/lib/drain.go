package lib

import (
	"context"
	"fmt"

	"github.com/a-novel-kit/golib/otel"
)

// Waiter blocks until the work it owns has finished. The short-code services implement it over the
// detached goroutines that deliver their emails.
type Waiter interface {
	Wait()
}

// Drain waits for every waiter to finish, or for ctx to expire.
//
// Mail delivery runs on its own goroutine, detached from the request that triggered it, so a
// cancelled request leaves a half-sent email alone. Process exit then reaches whatever is still in
// flight, and the loss is silent: the handler answered 202 long before, and the short code sits in
// the database intact. The user checks an inbox that stays empty.
//
// The bound carries as much weight as the wait. Each send holds its own timeout, and ctx is the
// second bound, so a sender that stops answering delays a deploy by the shutdown budget.
func Drain(ctx context.Context, waiters ...Waiter) error {
	ctx, span := otel.Tracer().Start(ctx, "lib.Drain")
	defer span.End()

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
		// The goroutine above outlives this return, parked on sends that carry their own timeouts
		// while the process exits.
		return fmt.Errorf("drain in-flight work: %w", ctx.Err())
	}
}
