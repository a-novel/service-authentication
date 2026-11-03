package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/transaction"

	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// CredentialsUpdateLocationConsentDao records the caller's consent.
type CredentialsUpdateLocationConsentDao interface {
	Exec(ctx context.Context, request *dao.CredentialsUpdateLocationConsentRequest) (*dao.Credentials, error)
}

// CredentialsUpdateLocationConsentDaoClearLocations erases the locations of the caller's sessions.
type CredentialsUpdateLocationConsentDaoClearLocations interface {
	Exec(ctx context.Context, request *dao.SessionClearLocationsRequest) error
}

// CredentialsUpdateLocationConsentRequest names the caller and their choice.
type CredentialsUpdateLocationConsentRequest struct {
	// UserID is the caller, taken from their verified access token.
	UserID uuid.UUID
	// Enabled is whether the account records the location of its sessions.
	Enabled *bool `validate:"required"`
}

// CredentialsUpdateLocationConsent sets whether the caller's sessions record their location: the IP
// address, the region and the city of each device. The consent is off until the user turns it on.
// Turning it off erases the stored location of every session of the account, in the same transaction.
type CredentialsUpdateLocationConsent struct {
	dao               CredentialsUpdateLocationConsentDao
	daoClearLocations CredentialsUpdateLocationConsentDaoClearLocations
	transactor        transaction.Transactor
}

func NewCredentialsUpdateLocationConsent(
	dao CredentialsUpdateLocationConsentDao,
	daoClearLocations CredentialsUpdateLocationConsentDaoClearLocations,
	transactor transaction.Transactor,
) *CredentialsUpdateLocationConsent {
	return &CredentialsUpdateLocationConsent{dao: dao, daoClearLocations: daoClearLocations, transactor: transactor}
}

// Exec returns [ErrCredentialsUpdateLocationConsentNotFound] when the account no longer exists.
func (service *CredentialsUpdateLocationConsent) Exec(
	ctx context.Context, request *CredentialsUpdateLocationConsentRequest,
) error {
	ctx, span := otel.Tracer().Start(ctx, "core.CredentialsUpdateLocationConsent")
	defer span.End()

	span.SetAttributes(attribute.String("user.id", request.UserID.String()))

	err := validate.Struct(request)
	if err != nil {
		return otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	enabled := *request.Enabled

	span.SetAttributes(attribute.Bool("credentials.location_consent", enabled))

	err = service.transactor.WithinTx(ctx, func(ctx context.Context) error {
		daoRequest := &dao.CredentialsUpdateLocationConsentRequest{ID: request.UserID, LocationConsent: enabled}

		_, err := service.dao.Exec(ctx, daoRequest)
		if err != nil {
			return fmt.Errorf("update location consent: %w", err)
		}

		if enabled {
			return nil
		}

		err = service.daoClearLocations.Exec(ctx, &dao.SessionClearLocationsRequest{UserID: request.UserID})
		if err != nil {
			return fmt.Errorf("clear session locations: %w", err)
		}

		return nil
	})
	if err != nil {
		return otel.ReportError(span, fmt.Errorf("run transaction: %w", err))
	}

	return nil
}
