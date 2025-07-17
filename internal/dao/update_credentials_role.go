package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/models"
)

//go:embed update_credentials_role.sql
var updateCredentialsRoleQuery string

type UpdateCredentialsRoleData struct {
	Role models.CredentialsRole
	Now  time.Time
}

type UpdateCredentialsRoleRepository struct{}

func NewUpdateCredentialsRoleRepository() *UpdateCredentialsRoleRepository {
	return &UpdateCredentialsRoleRepository{}
}

func (repository *UpdateCredentialsRoleRepository) UpdateCredentialsRole(
	ctx context.Context, userID uuid.UUID, data UpdateCredentialsRoleData,
) (*CredentialsEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.UpdateCredentialsRole")
	defer span.End()

	span.SetAttributes(
		attribute.String("credentials.id", userID.String()),
		attribute.String("credentials.role", data.Role.String()),
		attribute.Int64("credentials.now", data.Now.Unix()),
	)

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{}

	err = tx.NewRaw(updateCredentialsRoleQuery, data.Role, data.Now, userID).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, ErrCredentialsNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("update credentials: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
