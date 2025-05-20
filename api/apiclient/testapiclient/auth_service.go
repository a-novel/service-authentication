package testapiclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/apiclient"
	"github.com/a-novel/service-authentication/api/codegen"
	"github.com/a-novel/service-authentication/config"
	"github.com/a-novel/service-authentication/models"
)

var AuthAPIPort = 4001

var ErrTokenNotFound = errors.New("not found")

type mockAuthService struct {
	pool sync.Map
}

func (service *mockAuthService) Authenticate(
	_ context.Context, accessToken string,
) (*models.AccessTokenClaims, error) {
	if val, ok := service.pool.Load(accessToken); ok {
		return val.(*models.AccessTokenClaims), nil //nolint:forcetypeassert
	}

	return nil, ErrTokenNotFound
}

func (service *mockAuthService) AddPool(accessToken string, claims *models.AccessTokenClaims) {
	service.pool.Store(accessToken, claims)
}

var authServiceInstance = &mockAuthService{}

func AddPool(accessToken string, claims *models.AccessTokenClaims) {
	authServiceInstance.AddPool(accessToken, claims)
}

func Authenticate(ctx context.Context, accessToken string) (*models.AccessTokenClaims, error) {
	return authServiceInstance.Authenticate(ctx, accessToken)
}

func InitAuthServer() {
	securityHandler, err := api.NewSecurity(
		config.Permissions,
		authServiceInstance,
	)
	if err != nil {
		panic(err)
	}

	authAPI, err := codegen.NewServer(new(api.API), securityHandler)
	if err != nil {
		panic(err)
	}

	router := chi.NewRouter()
	router.Mount("/v1/", http.StripPrefix("/v1", authAPI))

	httpServer := &http.Server{ //nolint:gosec
		Addr:    ":" + strconv.Itoa(AuthAPIPort),
		Handler: router,
	}

	go func() {
		if err = httpServer.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	// Wait for the auth server to start, otherwise the mains server won't be able to connect to it.
	security := apiclient.NewSecuritySource()

	client, err := codegen.NewClient(fmt.Sprintf("http://127.0.0.1:%v/v1", AuthAPIPort), security)
	if err != nil {
		panic(err)
	}

	start := time.Now()
	_, err = client.Ping(context.Background())

	for time.Since(start) < 16*time.Second && err != nil {
		_, err = client.Ping(context.Background())
	}

	if err != nil {
		panic(err)
	}
}
