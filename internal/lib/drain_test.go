package lib_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// blockingWaiter is a Waiter that finishes only when released, standing in for a detached email
// send still in flight when the process is asked to stop.
type blockingWaiter struct {
	release chan struct{}
	once    sync.Once
}

func newBlockingWaiter() *blockingWaiter {
	return &blockingWaiter{release: make(chan struct{})}
}

func (w *blockingWaiter) Wait() { <-w.release }

func (w *blockingWaiter) finish() { w.once.Do(func() { close(w.release) }) }

func TestDrain(t *testing.T) {
	t.Parallel()

	t.Run("WaitsForEveryWaiter", func(t *testing.T) {
		t.Parallel()

		first, second := newBlockingWaiter(), newBlockingWaiter()

		errs := make(chan error, 1)
		go func() { errs <- lib.Drain(t.Context(), first, second) }()

		select {
		case <-errs:
			t.Fatal("Drain returned while work was still in flight")
		case <-time.After(50 * time.Millisecond):
		}

		first.finish()
		second.finish()

		require.NoError(t, <-errs)
	})

	t.Run("ReturnsWhenNothingIsInFlight", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, lib.Drain(t.Context()))
	})

	t.Run("GivesUpOnAnExpiredBudget", func(t *testing.T) {
		t.Parallel()

		stuck := newBlockingWaiter()
		t.Cleanup(stuck.finish)

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		t.Cleanup(cancel)

		start := time.Now()
		err := lib.Drain(ctx, stuck)

		// A sender that never returns delays the deploy by the budget and no more, which is what
		// lets the drain wait on sends at all.
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Less(t, time.Since(start), 5*time.Second)
	})
}
