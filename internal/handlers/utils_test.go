package handlers_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/servicetoken/servicetokentest"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/config/configtest"
	"github.com/a-novel/service-authentication/v2/internal/core"
)

// serviceTokenAudience is the audience the handler tests' service tokens name.
const serviceTokenAudience = "authentication"

// The account a credentials service returns, and the JSON body a handler answers it with.
var (
	credentialsFixture = &core.Credentials{
		ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:     "user@provider.com",
		Role:      authconfig.RoleUser,
		CreatedAt: time.Date(2018, time.February, 2, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2020, time.February, 2, 12, 0, 0, 0, time.UTC),
	}
	credentialsFixtureJSON = map[string]any{
		"id":        "00000000-0000-0000-0000-000000000001",
		"email":     "user@provider.com",
		"role":      authconfig.RoleUser,
		"createdAt": "2018-02-02T12:00:00Z",
		"updatedAt": "2020-02-02T12:00:00Z",
	}
)

// The token pair a session service returns, and the JSON body a handler answers it with.
var (
	tokenFixture     = &core.Token{AccessToken: "access-token", RefreshToken: "refresh-token"}
	tokenFixtureJSON = map[string]any{"accessToken": "access-token", "refreshToken": "refresh-token"}
)

// serveAsCaller serves request through handler, behind the service-token middleware the server
// mounts, with a token caller holds. An empty caller sends no token to a server that verifies none,
// as a local stack does.
func serveAsCaller(t *testing.T, handler http.Handler, request *http.Request, caller string) *http.Response {
	t.Helper()

	w := httptest.NewRecorder()

	if caller == "" {
		handler.ServeHTTP(w, request)

		return w.Result()
	}

	issuer := servicetokentest.NewIssuer(t)
	tokenConfig := servicetoken.Config{Issuer: issuer.URL, Audience: serviceTokenAudience, Callers: []string{caller}}

	request.Header.Set(servicetoken.Header, issuer.Sign(t, issuer.Claims(caller, serviceTokenAudience)))

	withServiceToken, err := httpf.ServiceTokenGate(tokenConfig, &http.Client{}, configtest.Logger)
	require.NoError(t, err)

	withServiceToken(handler).ServeHTTP(w, request)

	return w.Result()
}

// requireJSONBody requires the JSON body of res to be one document equal to expect. It decodes the
// body into any, so expect spells the wire format: an object is a map[string]any, an array a []any
// and a number a float64.
func requireJSONBody(t *testing.T, res *http.Response, expect any) {
	t.Helper()

	data, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var body any

	// Unmarshal, unlike a Decoder, refuses data after the first document.
	require.NoError(t, json.Unmarshal(data, &body))
	require.Equal(t, expect, body)
}

// requireNoStore requires res to forbid every cache, as an OAuth 2.0 response that carries a code, a
// token or a refusal of one must (RFC 6749 §5.1).
func requireNoStore(t *testing.T, res *http.Response) {
	t.Helper()

	require.Equal(t, "no-store", res.Header.Get("Cache-Control"))
	require.Equal(t, "no-cache", res.Header.Get("Pragma"))
}
