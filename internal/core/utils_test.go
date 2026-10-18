package core_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

const (
	mockUnsignedRefreshToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.eyJqdGkiOiI0MTI5OGE3ZC1jYWJkLTQ0ZmQtODJhZS1jYWViN" +
		"zM1OWI5YzgiLCJ1c2VySUQiOiJkMGY4YzkwNS1jOWYwLTRmYmUtYTBkOC01YzUxY2U4NWNlNzkifQ." +
		"wedontcareaboutthesignaturebutweneedonesohereweare"
	mockUnsignedJTI = "41298a7d-cabd-44fd-82ae-caeb7359b9c8"
)

func waitForMailDelivery(t *testing.T, delivery *core.MailDelivery) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	require.NoError(t, delivery.Wait(ctx))
}

// closeArgon2Gate makes every hash wait until the test ends, so [requireNoHash] can tell whether
// an operation hashed. Tests that close it cannot run in parallel with tests that hash.
func closeArgon2Gate(t *testing.T) {
	t.Helper()

	lib.LimitArgon2(0)
	t.Cleanup(func() { lib.LimitArgon2(runtime.GOMAXPROCS(0)) })
}

// requireNoHash runs exec behind the closed Argon2 gate and returns its error, failing the test
// when exec is stuck waiting to hash.
func requireNoHash(t *testing.T, exec func() error) error {
	t.Helper()

	done := make(chan error, 1)

	go func() { done <- exec() }()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		require.FailNow(t, "hashed before checking the proof")

		return nil
	}
}
