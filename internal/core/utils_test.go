package core_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/lib"
)

const (
	mockUnsignedRefreshToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.eyJqdGkiOiI0MTI5OGE3ZC1jYWJkLTQ0ZmQtODJhZS1jYWViN" +
		"zM1OWI5YzgiLCJ1c2VySUQiOiJkMGY4YzkwNS1jOWYwLTRmYmUtYTBkOC01YzUxY2U4NWNlNzkifQ." +
		"wedontcareaboutthesignaturebutweneedonesohereweare"
	mockUnsignedJTI = "41298a7d-cabd-44fd-82ae-caeb7359b9c8"
)

// requireNoHash runs exec while every Argon2 hash waits forever, and returns its error. It fails
// the test when exec is stuck waiting to hash. The gate is process-wide, so tests calling it cannot
// run in parallel with tests that hash.
func requireNoHash(t *testing.T, exec func() error) error {
	t.Helper()

	lib.LimitArgon2(0)
	defer lib.LimitArgon2(runtime.GOMAXPROCS(0))

	done := make(chan error, 1)

	go func() { done <- exec() }()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		require.FailNow(t, "hashed where no hash may run")

		return nil
	}
}

// noHashInTx is a transactor mock body that runs the transaction while no hash may run, since a
// transaction holds a pooled connection for as long as it lasts.
func noHashInTx(t *testing.T) func(ctx context.Context, fn func(ctx context.Context) error) error {
	t.Helper()

	return func(ctx context.Context, fn func(ctx context.Context) error) error {
		return requireNoHash(t, func() error { return fn(ctx) })
	}
}
