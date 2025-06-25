package services

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"

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
	span := sentry.StartSpan(ctx, "EmailExistsService.EmailExists")
	defer span.Finish()

	span.SetData("email", request.Email)

	exists, err := service.source.ExistsCredentialsEmail(span.Context(), request.Email)
	if err != nil {
		span.SetData("dao.error", err.Error())

		return false, NewErrEmailExistsService(fmt.Errorf("check email existence: %w", err))
	}

	span.SetData("exists", exists)

	return exists, nil
}

func NewEmailExistsService(source EmailExistsSource) *EmailExistsService {
	return &EmailExistsService{source: source}
}
