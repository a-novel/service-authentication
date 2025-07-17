package services

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel/golib/otel"
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

func NewEmailExistsService(source EmailExistsSource) *EmailExistsService {
	return &EmailExistsService{source: source}
}

func (service *EmailExistsService) EmailExists(ctx context.Context, request EmailExistsRequest) (bool, error) {
	ctx, span := otel.Tracer().Start(ctx, "service.EmailExists")
	defer span.End()

	span.SetAttributes(attribute.String("email", request.Email))

	exists, err := service.source.ExistsCredentialsEmail(ctx, request.Email)
	if err != nil {
		return false, otel.ReportError(span, fmt.Errorf("check email existence: %w", err))
	}

	span.SetAttributes(attribute.Bool("exists", exists))

	return otel.ReportSuccess(span, exists), nil
}
