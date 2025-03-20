package dao

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun/driver/pgdriver"

	"github.com/a-novel-kit/context"
	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/models"
)

var ErrInsertCredentialsRepository = errors.New("InsertCredentialsRepository.InsertCredentials")

func NewErrInsertCredentialsRepository(err error) error {
	return errors.Join(err, ErrInsertCredentialsRepository)
}

// InsertCredentialsData is the input used to perform the InsertCredentialsRepository.InsertCredentials action.
type InsertCredentialsData struct {
	// ID of the credentials. It must be unique (random).
	ID uuid.UUID

	// Email of the credentials.
	//
	// Each email MUST be unique among all credentials. You may use the ExistsCredentialsEmailRepository to ensure
	// a given address is not already in use.
	Email string
	// The user's password, encrypted using scrypt. This will be used when creating a session.
	//
	// This value is sensitive, and MUST always be transmitted using a secure channel / encryption. It MUST NEVER be
	// stored clear in the database.
	//
	// This value may be empty when creating a shadow user. Shadow Users are users whose account is created indirectly,
	// and so their password must be set post-creation.
	// While technically valid, shadow users accounts cannot be used to create sessions, as this process requires a
	// password.
	Password string

	// Time at which the credentials were created.
	Now time.Time
}

// InsertCredentialsRepository is the repository used to perform the
// InsertCredentialsRepository.InsertCredentials action.
//
// You may create one using the NewInsertCredentialsRepository function.
type InsertCredentialsRepository struct{}

// InsertCredentials inserts a new set of credentials in the database.
//
// The new credentials MUST have a unique email. This email is used to reach the user, and to uniquely identify them.
// The ExistsCredentialsEmailRepository can be used beforehand to ensure an email is available.
//
// A shadow user may be created by providing an empty password. When this happens, the new user MUST have a way to
// set its password post creation, by using a ShortCodeEntity for example.
func (repository *InsertCredentialsRepository) InsertCredentials(
	ctx context.Context, data InsertCredentialsData,
) (*CredentialsEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := pgctx.Context(ctx)
	if err != nil {
		return nil, NewErrInsertCredentialsRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entity := &CredentialsEntity{
		ID:        data.ID,
		Email:     data.Email,
		Password:  data.Password,
		CreatedAt: data.Now,
		UpdatedAt: data.Now,
		Role:      models.CredentialsRoleUser,
	}

	// Execute query.
	if _, err = tx.NewInsert().Model(entity).Exec(ctx); err != nil {
		var pgErr pgdriver.Error
		if errors.As(err, &pgErr) && pgErr.Field('C') == "23505" {
			return nil, NewErrInsertCredentialsRepository(errors.Join(err, ErrCredentialsAlreadyExists))
		}

		return nil, NewErrInsertCredentialsRepository(fmt.Errorf("insert credentials: %w", err))
	}

	return entity, nil
}

func NewInsertCredentialsRepository() *InsertCredentialsRepository {
	return &InsertCredentialsRepository{}
}
