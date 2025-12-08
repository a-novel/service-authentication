package dao

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.shortCodeSelect.sql
var shortCodeSelectQuery string

var ErrShortCodeSelectNotFound = errors.New("short code not found")

type ShortCodeSelectRequest struct {
	Usage  string
	Target string
}

type ShortCodeSelect struct{}

func NewShortCodeSelect() *ShortCodeSelect {
	return new(ShortCodeSelect)
}

func (repository *ShortCodeSelect) Exec(ctx context.Context, request *ShortCodeSelectRequest) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.ShortCodeSelect")
	defer span.End()

	span.SetAttributes(
		attribute.String("data", request.Target),
		attribute.String("usage", request.Usage),
	)

	tx, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get transaction: %w", err))
	}

	entity := new(ShortCode)

	err = tx.NewRaw(shortCodeSelectQuery, request.Target, request.Usage).Scan(ctx, entity)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = errors.Join(err, ErrShortCodeSelectNotFound)
		}

		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return otel.ReportSuccess(span, entity), nil
}
