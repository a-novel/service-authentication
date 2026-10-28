package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

// ShortCodeCreateDao persists a new short code and returns the stored row.
type ShortCodeCreateDao interface {
	Exec(ctx context.Context, request *dao.ShortCodeInsertRequest) (*dao.ShortCode, error)
}

// ShortCodeCreateMailOutbox hands a stored mail to its delivery; satisfied by [MailOutbox].
type ShortCodeCreateMailOutbox interface {
	Enqueue(ctx context.Context, job *MailOutboxJob)
}

// ShortCodeCreateRequest describes a code to issue: the flow it is for, the
// subject it binds to, any flow-specific data to carry, and how long it lives.
type ShortCodeCreateRequest struct {
	Usage  string `validate:"required,usage"`
	Target string `validate:"required,max=1024"`
	Data   any
	TTL    time.Duration `validate:"required"`

	// Override expires and replaces any existing code for the same target and usage.
	// When false, a duplicate fails the request.
	Override bool

	// Mail, when set, stores the mail that carries the code in the outbox, with the code.
	Mail *ShortCodeCreateMail
}

// ShortCodeCreateMail addresses the mail that carries a new short code.
type ShortCodeCreateMail struct {
	Recipient string `validate:"required,email,max=1024"`
	Lang      string `validate:"required,langs"`
}

// ShortCodeCreate issues a short code: it generates a random plaintext code,
// stores only its Argon2id hash, and returns the plaintext once so the caller can
// deliver it to the target. [ShortCodeVerify] and [ShortCodeConsume] redeem it.
type ShortCodeCreate struct {
	dao        ShortCodeCreateDao
	mailOutbox ShortCodeCreateMailOutbox
	config     authconfig.ShortCodes
}

// NewShortCodeCreate wires the create service to its DAO, the outbox that delivers its mail, and
// the short-code configuration.
func NewShortCodeCreate(
	dao ShortCodeCreateDao, mailOutbox ShortCodeCreateMailOutbox, config authconfig.ShortCodes,
) *ShortCodeCreate {
	return &ShortCodeCreate{dao: dao, mailOutbox: mailOutbox, config: config}
}

// Exec issues a new short code and returns it with the plaintext populated in
// [ShortCode.PlainCode]; only the hash is stored. A request with a Mail hands the
// plaintext to the outbox, which sends it without storing it.
func (service *ShortCodeCreate) Exec(
	ctx context.Context, request *ShortCodeCreateRequest,
) (*ShortCode, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.ShortCodeCreate")
	defer span.End()

	err := validate.Struct(request)
	if err != nil {
		return nil, otel.ReportError(span, errors.Join(err, ErrInvalidRequest))
	}

	plainCode, encrypted, err := newShortCodeSecret(service.config.Size)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	var serializedData []byte
	if request.Data != nil {
		serializedData, err = json.Marshal(request.Data)
		if err != nil {
			return nil, otel.ReportError(span, fmt.Errorf("serialize data: %w", err))
		}
	}

	now := time.Now()
	daoRequest := &dao.ShortCodeInsertRequest{
		ID:        uuid.New(),
		Code:      encrypted,
		Usage:     request.Usage,
		Target:    request.Target,
		Data:      serializedData,
		Now:       now,
		ExpiresAt: now.Add(request.TTL),
		Override:  request.Override,
	}

	if request.Mail != nil {
		daoRequest.Mail = &dao.ShortCodeInsertMail{Recipient: request.Mail.Recipient, Lang: request.Mail.Lang}
	}

	entity, err := service.dao.Exec(ctx, daoRequest)
	if err != nil {
		return nil, otel.ReportError(span, err)
	}

	if request.Mail != nil {
		service.mailOutbox.Enqueue(ctx, &MailOutboxJob{ShortCodeID: entity.ID, PlainCode: plainCode})
	}

	return newShortCode(entity, plainCode), nil
}

// newShortCodeSecret returns a random plaintext code of size characters and its Argon2id hash. Only
// the hash reaches the database.
func newShortCodeSecret(size int) (string, string, error) {
	plainCode, err := lib.NewRandomURLString(size)
	if err != nil {
		return "", "", fmt.Errorf("generate short code: %w", err)
	}

	encrypted, err := lib.GenerateArgon2(plainCode, lib.Argon2ParamsDefault)
	if err != nil {
		return "", "", fmt.Errorf("encrypt short code: %w", err)
	}

	return plainCode, encrypted, nil
}
