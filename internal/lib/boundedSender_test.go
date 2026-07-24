package lib_test

import (
	"errors"
	"sync"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// countingSender tracks how many deliveries run at once, holding each one open until released.
type countingSender struct {
	mu      sync.Mutex
	current int
	peak    int
	release chan struct{}
	err     error
	pingErr error
}

func (sender *countingSender) SendMail(_ smtp.MailUsers, _ *template.Template, _ string, _ any) error {
	sender.mu.Lock()
	sender.current++
	sender.peak = max(sender.peak, sender.current)
	sender.mu.Unlock()

	if sender.release != nil {
		<-sender.release
	}

	sender.mu.Lock()
	sender.current--
	sender.mu.Unlock()

	return sender.err
}

func (sender *countingSender) Ping() error { return sender.pingErr }

func (sender *countingSender) running() int {
	sender.mu.Lock()
	defer sender.mu.Unlock()

	return sender.current
}

func TestBoundedSender(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	t.Run("CapsConcurrentDeliveries", func(t *testing.T) {
		t.Parallel()

		inner := &countingSender{release: make(chan struct{})}
		sender := lib.NewBoundedSender(inner, 2)

		errs := make(chan error, 6)

		var wg sync.WaitGroup

		for range 6 {
			wg.Add(1)

			go func() {
				defer wg.Done()

				errs <- sender.SendMail(nil, nil, "", nil)
			}()
		}

		// Two deliveries hold the slots; the four others are parked waiting for one.
		require.Eventually(t, func() bool { return inner.running() == 2 }, time.Second, time.Millisecond)

		close(inner.release)
		wg.Wait()
		close(errs)

		for err := range errs {
			require.NoError(t, err)
		}

		require.Equal(t, 2, inner.peak)
	})

	t.Run("PingSkipsDeliverySlots", func(t *testing.T) {
		t.Parallel()

		inner := &countingSender{release: make(chan struct{})}
		sender := lib.NewBoundedSender(inner, 1)

		errs := make(chan error, 1)

		go func() { errs <- sender.SendMail(nil, nil, "", nil) }()

		require.Eventually(t, func() bool { return inner.running() == 1 }, time.Second, time.Millisecond)

		// The only slot is busy; Ping returning at all is the assertion.
		require.NoError(t, sender.Ping())

		close(inner.release)
		require.NoError(t, <-errs)
	})

	t.Run("PropagatesDeliveryErrors", func(t *testing.T) {
		t.Parallel()

		sender := lib.NewBoundedSender(&countingSender{err: errFoo}, 2)

		require.ErrorIs(t, sender.SendMail(nil, nil, "", nil), errFoo)
	})

	t.Run("PropagatesPingErrors", func(t *testing.T) {
		t.Parallel()

		sender := lib.NewBoundedSender(&countingSender{pingErr: errFoo}, 2)

		require.ErrorIs(t, sender.Ping(), errFoo)
	})

	t.Run("RaisesALimitBelowOne", func(t *testing.T) {
		t.Parallel()

		sender := lib.NewBoundedSender(&countingSender{}, 0)

		// A zero-capacity slot channel never grants a slot; completing at all is the assertion.
		require.NoError(t, sender.SendMail(nil, nil, "", nil))
	})
}
