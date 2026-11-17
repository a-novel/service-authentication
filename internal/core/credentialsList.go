package core

import (
	"cmp"
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// CredentialsListDao lists a page of accounts.
type CredentialsListDao interface {
	Exec(ctx context.Context, request *dao.CredentialsListRequest) ([]*dao.Credentials, error)
}

// CredentialsListLimitDefault is the page size of a request that sets no limit.
const CredentialsListLimitDefault = 100

// CredentialsListRequest selects a page of accounts, and the roles to narrow it to. A zero Limit
// takes [CredentialsListLimitDefault].
type CredentialsListRequest struct {
	Limit  int      `validate:"omitempty,min=1,max=100"`
	Offset int      `validate:"min=0"`
	Roles  []string `validate:"min=0,max=10,dive,role"`
}

// CredentialsList returns a paginated page of accounts, optionally narrowed to a
// set of roles.
type CredentialsList struct {
	dao CredentialsListDao
}

func NewCredentialsList(dao CredentialsListDao) *CredentialsList {
	return &CredentialsList{dao: dao}
}

func (service *CredentialsList) Exec(
	ctx context.Context, request *CredentialsListRequest,
) ([]*Credentials, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.CredentialsList")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	daoRequest := &dao.CredentialsListRequest{
		Limit:  cmp.Or(request.Limit, CredentialsListLimitDefault),
		Offset: request.Offset,
		Roles:  request.Roles,
	}

	entities, err := service.dao.Exec(ctx, daoRequest)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("list credentials: %w", err))
	}

	span.SetAttributes(attribute.Int("credentials.count", len(entities)))

	credentials := make([]*Credentials, len(entities))
	for i, entity := range entities {
		credentials[i] = newCredentials(entity)
	}

	return credentials, nil
}
