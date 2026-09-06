package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
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
