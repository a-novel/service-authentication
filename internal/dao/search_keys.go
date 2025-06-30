package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/getsentry/sentry-go"

	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/models"
)

var ErrSearchKeysRepository = errors.New("SearchKeysRepository.SearchKeys")

func NewErrSearchKeysRepository(err error) error {
	return errors.Join(err, ErrSearchKeysRepository)
}

// KeysMaxBatchSize is a security used to limit the number of keys retrieved by a search operation.
//
// Normally, regular key rotation and well configured expiration should limit the number of keys per batch, so the
// search request has no pagination. Since we can't overrule an issue that would cause the number of keys in a batch
// to balloon, this value is used as a security measurement, to guarantee an upper limit of keys retrieved.
const KeysMaxBatchSize = 100

// SearchKeysRepository is the repository used to perform the SearchKeysRepository.SearchKeys action.
//
// You may create one using the NewSearchKeysRepository function.
type SearchKeysRepository struct{}

func NewSearchKeysRepository() *SearchKeysRepository {
	return &SearchKeysRepository{}
}

// SearchKeys lists keys related to a specific usage.
//
// All keys that share the same usage are called a batch. A batch only contains active keys, and is ordered by creation
// date, from the most recent to the oldest. Most recent keys must be used in priority to issue new values. Older keys
// are provided for checking older values only.
//
// While a call to SearchKeys should return every active key without exception, an upper limit is set to prevent a
// potential overhead of the response when too much active keys coexist. This limit is set to KeysMaxBatchSize. If
// a batch happens to contain more keys, an error is logged, and only the first KeysMaxBatchSize keys are returned.
func (repository *SearchKeysRepository) SearchKeys(ctx context.Context, usage models.KeyUsage) ([]*KeyEntity, error) {
	span := sentry.StartSpan(ctx, "SearchKeysRepository.SearchKeys")
	defer span.Finish()

	span.SetData("usage", usage)

	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(span.Context())
	if err != nil {
		span.SetData("postgres.context.error", err.Error())

		return nil, NewErrSearchKeysRepository(fmt.Errorf("get postgres client: %w", err))
	}

	var entities []*KeyEntity

	// Execute query.
	err = tx.NewSelect().
		Model(&entities).
		Where("usage = ?", usage).
		Order("created_at DESC").
		// Adda +1 to the limit, so we can differentiate between limit reached (which is OK) and limit exceeded
		// (which is not).
		Limit(KeysMaxBatchSize + 1).
		Scan(span.Context())
	if err != nil {
		span.SetData("scan.error", err.Error())

		return nil, NewErrSearchKeysRepository(fmt.Errorf("list keys: %w", err))
	}

	span.SetData("keys.count", len(entities))

	// Log an error when too many keys are found. This indicates a potential misconfiguration.
	if len(entities) > KeysMaxBatchSize {
		logger := sentry.NewLogger(span.Context())
		logger.Errorf(ctx, "more than %d keys found for usage %s", KeysMaxBatchSize, usage)

		// Truncate the list to the maximum allowed size.
		entities = entities[:KeysMaxBatchSize]
	}

	return entities, nil
}
