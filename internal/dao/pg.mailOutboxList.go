package dao

import (
	"context"
	_ "embed"
	"fmt"
	"time"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/postgres"
)

//go:embed pg.mailOutboxList.sql
var mailOutboxListQuery string

// MailOutboxListRequest is the input to [PgMailOutboxList.Exec].
type MailOutboxListRequest struct {
	// Now is the instant the entries must be due by.
	Now time.Time
	// Limit bounds the entries one call returns.
	Limit int
}

// PgMailOutboxList returns the entries due by a given instant, the longest due first.
type PgMailOutboxList struct{}

func NewPgMailOutboxList() *PgMailOutboxList {
	return &PgMailOutboxList{}
}

func (dao *PgMailOutboxList) Exec(ctx context.Context, request *MailOutboxListRequest) ([]*MailOutboxEntry, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.PgMailOutboxList")
	defer span.End()

	db, err := postgres.GetContext(ctx)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("get postgres context: %w", err))
	}

	entities := make([]*MailOutboxEntry, 0)

	err = db.NewRaw(mailOutboxListQuery, request.Now, request.Limit).Scan(ctx, &entities)
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("execute query: %w", err))
	}

	return entities, nil
}
