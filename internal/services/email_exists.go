package services

import (
	"fmt"

	"github.com/a-novel-kit/context"
)

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
		return false, fmt.Errorf("(EmailExistsService.EmailExists) check email existence: %w", err)
	}

	return exists, nil
}

func NewEmailExistsService(source EmailExistsSource) *EmailExistsService {
	return &EmailExistsService{source: source}
}
