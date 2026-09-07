// This program exercises the public authentication API from an external module.
package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	loggingpresets "github.com/a-novel-kit/golib/logging/presets"
	serviceauthentication "github.com/a-novel/service-authentication/v2/pkg/go"
	servicejsonkeys "github.com/a-novel/service-json-keys/v2/pkg/go"
)

func main() {
	client, err := servicejsonkeys.NewClient("localhost:1", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	verifier, err := servicejsonkeys.NewClaimsVerifier[serviceauthentication.Claims](client)
	if err != nil {
		panic(err)
	}

	permissions := serviceauthentication.Permissions{Roles: map[string]serviceauthentication.Role{
		"reader": {Permissions: []string{"read"}, Priority: 1},
	}}
	priority, err := permissions.Priority("reader")
	if err != nil || priority != 1 {
		panic("permissions alias lost its priority method")
	}
	if _, err := permissions.Priority("unknown"); err == nil {
		panic("unknown role was accepted")
	}

	withAuth := serviceauthentication.NewAuthHandler(verifier, permissions, &loggingpresets.LogLocal{Out: io.Discard})
	router := chi.NewRouter()
	withAuth(router).Get("/", func(w http.ResponseWriter, r *http.Request) {
		claims, err := serviceauthentication.GetClaimsContext(r.Context())
		if err != nil || claims != nil {
			panic("optional authentication changed")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusNoContent {
		panic("optional authentication rejected an unauthenticated request")
	}

	claims := &serviceauthentication.Claims{Roles: []string{"reader"}}
	ctx := serviceauthentication.SetClaimsContext(context.Background(), claims)
	resolved, err := serviceauthentication.MustGetClaimsContext(ctx)
	if err != nil || resolved != claims {
		panic("claims context round-trip failed")
	}
}
