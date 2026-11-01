package core_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

const (
	mockUnsignedRefreshToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0.eyJqdGkiOiI0MTI5OGE3ZC1jYWJkLTQ0ZmQtODJhZS1jYWViN" +
		"zM1OWI5YzgiLCJ1c2VySUQiOiJkMGY4YzkwNS1jOWYwLTRmYmUtYTBkOC01YzUxY2U4NWNlNzkifQ." +
		"wedontcareaboutthesignaturebutweneedonesohereweare"
	mockUnsignedJTI = "41298a7d-cabd-44fd-82ae-caeb7359b9c8"
)

const (
	// accountPlatformCaller is the account platform's caller in the authorization code tests.
	accountPlatformCaller = "system:serviceaccount:account:web"
	// studioCaller is the caller of the studio client.
	studioCaller = "system:serviceaccount:studio:web"
	// studioRedirectURI is the redirect URI of the studio client.
	studioRedirectURI = "https://studio.example.com/auth/callback"
	// forumRedirectURI is the redirect URI of the forum client.
	forumRedirectURI = "https://forum.example.com/auth/callback"
)

// oauthPreset is the client registry of the authorization code tests: studio, and forum beside it.
var oauthPreset = authconfig.OAuth{
	AccountPlatformCaller: accountPlatformCaller,
	Clients: map[string]authconfig.Client{
		"studio": {RedirectURI: studioRedirectURI, Caller: studioCaller},
		"forum":  {RedirectURI: forumRedirectURI, Caller: "system:serviceaccount:forum:web"},
	},
}

// changed returns a copy of value, changed by update.
func changed[T any](value T, update func(value *T)) *T {
	update(&value)

	return &value
}

// mockRefreshToken returns an unsigned refresh token carrying claims, as the signer would sign it.
// The tests decode it without verifying it.
func mockRefreshToken(claims session.RefreshClaims) string {
	payload, err := json.Marshal(claims)
	if err != nil {
		panic(err)
	}

	return "eyJ0eXAiOiJKV1QiLCJhbGciOiJub25lIn0." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

func waitForMailDelivery(t *testing.T, delivery *core.MailDelivery) {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	require.NoError(t, delivery.Wait(ctx))
}

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
