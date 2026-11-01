package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
