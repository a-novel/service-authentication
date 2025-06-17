package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/a-novel/service-authentication/internal/lib"

	"github.com/a-novel/service-authentication/models"
)

var ErrSelectShortCodeByParamsRepository = errors.New("SelectShortCodeByParamsRepository.SelectShortCodeByParams")

func NewErrSelectShortCodeByParamsRepository(err error) error {
	return errors.Join(err, ErrSelectShortCodeByParamsRepository)
}

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

// SelectShortCodeByParams returns a short code based on its unique combined identifier (target and usage).
//
// This method only returns active short codes.
//
// The code returned by this method is encrypted. The clear code must be sent and retrieved through a secure channel.
// Once a short code is successfully verified, it should be expired using the DeleteShortCodeRepository.
func (repository *SelectShortCodeByParamsRepository) SelectShortCodeByParams(
	ctx context.Context, data SelectShortCodeByParamsData,
) (*ShortCodeEntity, error) {
	// Retrieve a connection to postgres from the context.
	tx, err := lib.PostgresContext(ctx)
	if err != nil {
		return nil, NewErrSelectShortCodeByParamsRepository(fmt.Errorf("get transaction: %w", err))
	}

	var entity ShortCodeEntity

	// Execute query.
	err = tx.NewSelect().
		Model(&entity).
		Where("target = ?", data.Target).
		Where("usage = ?", data.Usage).
		Order("id DESC").
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, NewErrSelectShortCodeByParamsRepository(ErrShortCodeNotFound)
		}

		return nil, NewErrSelectShortCodeByParamsRepository(fmt.Errorf("select short code: %w", err))
	}

	return &entity, nil
}

func NewSelectShortCodeByParamsRepository() *SelectShortCodeByParamsRepository {
	return &SelectShortCodeByParamsRepository{}
}
