package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
	"github.com/a-novel/golib/postgres"

	"github.com/a-novel/service-authentication/models"
)

//go:embed select_short_code_by_params.sql
var selectShortCodeByParamsQuery string

// SelectShortCodeByParamsData is the input used to perform the
// SelectShortCodeByParamsRepository.SelectShortCodeByParams action.
type SelectShortCodeByParamsData struct {
	// Information about the resource the short code grants access to.
	Usage models.ShortCodeUsage
	// The target that is allowed to access the resource. Only this target can retrieve the short code.
	Target string
}

// SelectShortCodeByParamsRepository is the repository used to perform the
// SelectShortCodeByParamsRepository.SelectShortCodeByParams action.
//
// You may create one using the NewSelectShortCodeByParamsRepository function.
type SelectShortCodeByParamsRepository struct{}

func NewSelectShortCodeByParamsRepository() *SelectShortCodeByParamsRepository {
	return &SelectShortCodeByParamsRepository{}
}

// SelectShortCodeByParams returns a short code based on its unique combined identifier (target and usage).
//
// This method only returns active short codes.
//
// The code returned by this method is encrypted. The clear code must be sent and retrieved through a secure channel.
// Once a short code is successfully verified, it should be expired using the DeleteShortCodeRepository.
func (repository *SelectShortCodeByParamsRepository) SelectShortCodeByParams(
	ctx context.Context, data SelectShortCodeByParamsData,
) (*ShortCodeEntity, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.SelectShortCodeByParams")
	defer span.End()

	span.SetAttributes(
		attribute.String("data", data.Target),
		attribute.String("usage", data.Usage.String()),
	)

	// Retrieve a connection to postgres from the context.
	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := &ShortCodeEntity{}

	// Execute query.
	err = tx.NewRaw(selectShortCodeByParamsQuery, data.Target, data.Usage).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, otel.ReportError(span, ErrShortCodeNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("select short code: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
