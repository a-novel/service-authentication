package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/internal/lib"
)

var ErrExistsCredentialsEmailRepository = errors.New("ExistsCredentialsEmailRepository.ExistsCredentialsEmail")

func NewErrExistsCredentialsEmailRepository(err error) error {
	return errors.Join(err, ErrExistsCredentialsEmailRepository)
}

// ExistsCredentialsEmailRepository is the repository used to perform the
// ExistsCredentialsEmailRepository.ExistsCredentialsEmail action.
//
// You may create one using the NewExistsCredentialsEmailRepository function.
type ExistsCredentialsEmailRepository struct{}

// ExistsCredentialsEmail returns whether a credential with the given email exists in the database.
//
// Emails MUST be unique, and this method should be used to verify the availability of an address before creating it.
func (repository *ExistsCredentialsEmailRepository) ExistsCredentialsEmail(
	ctx context.Context, email string,
) (bool, error) {
	span := sentry.StartSpan(ctx, "ExistsCredentialsEmailRepository.ExistsCredentialsEmail")
	defer span.Finish()

	span.SetData("email", email)

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return false, NewErrExistsCredentialsEmailRepository(fmt.Errorf("get postgres client: %w", err))
	}

	// Execute query.
	exists, err := tx.NewSelect().
		Model((*CredentialsEntity)(nil)).
		Where("email = ?", email).
		Order("email DESC").
		Exists(span.Context())
	if err != nil {
		span.SetData("exists.error", err.Error())

		return false, NewErrExistsCredentialsEmailRepository(fmt.Errorf("check database: %w", err))
	}

	return exists, nil
}

func NewExistsCredentialsEmailRepository() *ExistsCredentialsEmailRepository {
	return &ExistsCredentialsEmailRepository{}
}
