package apiclient

import (
	"context"

	"github.com/a-novel/service-authentication/internal/api/codegen"
)

type SecuritySource struct {
	token string
}

func NewSecuritySource() *SecuritySource {
	return &SecuritySource{}
}

func (security *SecuritySource) BearerAuth(_ context.Context, _ codegen.OperationName) (codegen.BearerAuth, error) {
	return codegen.BearerAuth{
		Token: security.token,
	}, nil
}

func (security *SecuritySource) SetToken(token string) {
	security.token = token
}

func (security *SecuritySource) GetToken() string {
	return security.token
}
