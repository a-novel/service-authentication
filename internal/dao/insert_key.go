package dao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"

	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrInsertKeyRepository = errors.New("InsertKeyRepository.InsertKey")

func NewErrInsertKeyRepository(err error) error {
	return errors.Join(err, ErrInsertKeyRepository)
}

// InsertKeyData is the input used to perform the InsertKeyRepository.InsertKey action.
type InsertKeyData struct {
	// ID of the new key. It MUST be unique (random).
	ID uuid.UUID

	// The private key in JSON Web Key format.
	//
	// The key MUST BE encrypted, and the result of this encryption is stored as a base64 raw URL encoded string.
	PrivateKey string
	// The public key in JSON Web Key format. The key is stored as a base64 raw URL encoded string.
	//
	// This value is OPTIONAL for symmetric keys.
	PublicKey *string

	// Intended usage of the key. See the type documentation for more details.
	Usage models.KeyUsage

	// Time at which the key was created. This is important when listing keys, as the most recent keys are
	// used in priority.
	Now time.Time
	// Expiration of the key. Each key pair is REQUIRED to expire past a certain time. Once the expiration date
	// is reached, the key pair becomes invisible to the keys view.
	Expiration time.Time
}

// InsertKeyRepository is the repository used to perform the InsertKeyRepository.InsertKey action.
//
// You may create one using the NewInsertKeyRepository function.
type InsertKeyRepository struct{}

func NewInsertKeyRepository() *InsertKeyRepository {
	return &InsertKeyRepository{}
}

// InsertKey inserts a new key pair in the database.
//
// A given key pair is REQUIRED to have an expiration date, as it must be rotated on a regular basis. Only public keys
// may be exposed to the application.
func (repository *InsertKeyRepository) InsertKey(ctx context.Context, data InsertKeyData) (*KeyEntity, error) {
	span := sentry.StartSpan(ctx, "InsertKeyRepository.InsertKey")
	defer span.Finish()

	span.SetData("key.id", data.ID.String())
	span.SetData("key.usage", data.Usage)
	span.SetData("key.createdAt", data.Now.String())
	span.SetData("key.expiresAt", data.Expiration.String())

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return nil, NewErrInsertKeyRepository(fmt.Errorf("get postgres client: %w", err))
	}

	entity := &KeyEntity{
		ID:         data.ID,
		PrivateKey: data.PrivateKey,
		PublicKey:  data.PublicKey,
		Usage:      data.Usage,
		CreatedAt:  data.Now,
		ExpiresAt:  data.Expiration,
	}

	// Execute query.
	_, err = tx.NewInsert().Model(entity).Exec(span.Context())
	if err != nil {
		span.SetData("insert.error", err.Error())
		// Don't check for collision errors: this is useless, as randomly generated UUIDs have a negligible chance of
		// colliding.
		return nil, NewErrInsertKeyRepository(fmt.Errorf("insert entity: %w", err))
	}

	return entity, nil
}
