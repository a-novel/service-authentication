package services

import (
	"context"
	"fmt"

	"github.com/go-faster/errors"
)

var ErrEmailExistsService = errors.New("EmailExistsService.EmailExists")

func NewErrEmailExistsService(err error) error {
	return errors.Join(err, ErrEmailExistsService)
}

type EmailExistsSource interface {
	ExistsCredentialsEmail(ctx context.Context, email string) (bool, error)
}

type EmailExistsRequest struct {
	Email string
}

type EmailExistsService struct {
	source EmailExistsSource
}

func (service *EmailExistsService) EmailExists(ctx context.Context, request EmailExistsRequest) (bool, error) {
	exists, err := service.source.ExistsCredentialsEmail(ctx, request.Email)
	if err != nil {
		return false, NewErrEmailExistsService(fmt.Errorf("check email existence: %w", err))
	}

	return exists, nil
}

func NewEmailExistsService(source EmailExistsSource) *EmailExistsService {
	return &EmailExistsService{source: source}
}
