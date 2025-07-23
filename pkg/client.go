package pkg

import (
	"context"
	"fmt"
	"time"

	"github.com/a-novel/service-authentication/models/api"
)

const (
	defaultPingInterval = 500 * time.Millisecond
	defaultPingTimeout  = 16 * time.Second
)

type APIClient = apimodels.Client

// NewAPIClient creates a new client to interact with a JSON keys server.
func NewAPIClient(ctx context.Context, url string, source apimodels.SecuritySource) (*apimodels.Client, error) {
	client, err := apimodels.NewClient(url, source)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	start := time.Now()
	_, err = client.Healthcheck(ctx)

	for time.Since(start) < defaultPingTimeout && err != nil {
		time.Sleep(defaultPingInterval)

		_, err = client.Healthcheck(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("ping server: %w", err)
	}

	return client, nil
}

type BearerSource struct {
	token string
}

func NewBearerSource() *BearerSource {
	return &BearerSource{token: ""}
}

func (source *BearerSource) SetToken(token string) {
	source.token = token
}

func (source *BearerSource) GetToken() string {
	return source.token
}

func (source *BearerSource) BearerAuth(_ context.Context, _ apimodels.OperationName) (apimodels.BearerAuth, error) {
	return apimodels.BearerAuth{Token: source.token}, nil
}
