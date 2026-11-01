package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/servicetoken"
	"github.com/a-novel-kit/golib/servicetoken/servicetokentest"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
)

// serviceTokenAudience is the audience the handler tests' service tokens name.
const serviceTokenAudience = "authentication"

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
	middlewares.RestServiceToken(tokenConfig, &http.Client{}, config.LoggerDev)(handler).ServeHTTP(w, request)

	return w.Result()
}

// requireJSONBody decodes the JSON body of res into a value of expect's type, a pointer, and
// requires it to equal expect.
func requireJSONBody(t *testing.T, res *http.Response, expect any) {
	t.Helper()

	body := reflect.New(reflect.TypeOf(expect).Elem()).Interface()

	require.NoError(t, json.NewDecoder(res.Body).Decode(body))
	require.Equal(t, expect, body)
}

// requireNoStore requires res to forbid every cache, as an OAuth 2.0 response that carries a code, a
// token or a refusal of one must (RFC 6749 §5.1).
func requireNoStore(t *testing.T, res *http.Response) {
	t.Helper()

	require.Equal(t, "no-store", res.Header.Get("Cache-Control"))
	require.Equal(t, "no-cache", res.Header.Get("Pragma"))
}
