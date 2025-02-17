package dao

import (
	"fmt"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"
)

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
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return false, fmt.Errorf(
			"(ExistsCredentialsEmailRepository.ExistsCredentialsEmail) get postgres client: %w", err,
		)
	}

	// Execute query.
	exists, err := tx.NewSelect().Model((*CredentialsEntity)(nil)).Where("email = ?", email).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("(ExistsCredentialsEmailRepository.ExistsCredentialsEmail) check database: %w", err)
	}

	return exists, nil
}

func NewExistsCredentialsEmailRepository() *ExistsCredentialsEmailRepository {
	return &ExistsCredentialsEmailRepository{}
}
