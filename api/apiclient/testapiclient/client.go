package testapiclient

import (
	"fmt"
	"time"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/apiclient"
	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/config"
)

func GetServerClient() (*codegen.Client, *apiclient.SecuritySource, error) {
	security := apiclient.NewSecuritySource()

	client, err := codegen.NewClient(fmt.Sprintf("http://127.0.0.1:%v/v1", config.API.Port), security)
	if err != nil {
		return nil, nil, fmt.Errorf("create client: %w", err)
	}

	start := time.Now()
	_, err = client.Ping(context.Background())

	for time.Since(start) < 16*time.Second && err != nil {
		_, err = client.Ping(context.Background())
	}

	if err != nil {
		return nil, nil, fmt.Errorf("ping server: %w", err)
	}

	return client, security, nil
}
