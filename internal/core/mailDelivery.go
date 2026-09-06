package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"text/template"

	"go.opentelemetry.io/otel/attribute"

	"github.com/a-novel-kit/golib/otel"
	"github.com/a-novel-kit/golib/smtp"
)

const (
	mailDeliveryKindEmailUpdate   = "email-update"
	mailDeliveryKindPasswordReset = "password-reset"
	mailDeliveryKindRegister      = "register"
)

// ErrMailDeliveryUnavailable is returned when delivery capacity cannot be reserved before the
// caller's context ends or after the owner has stopped accepting work.
var ErrMailDeliveryUnavailable = errors.New("mail delivery unavailable")

// MailDeliverySMTP sends the templated message owned by [MailDelivery].
type MailDeliverySMTP interface {
	// SendMail renders and sends one message to its recipients.
	SendMail(to smtp.MailUsers, t *template.Template, tName string, data any) error
}

// MailDeliveryRequest describes one accepted message without carrying its originating request
// context. Data may contain credentials and must remain outside logs and traces.
type MailDeliveryRequest struct {
	// To identifies the message recipients.
	To smtp.MailUsers
	// Template contains the localized message definitions.
	Template *template.Template
	// TemplateName selects the localized definition to render.
	TemplateName string
	// Data contains the variables used to render the message.
	Data any
	// Kind is a stable, non-sensitive delivery category used for telemetry.
	Kind string
}

// MailDeliveryReservation represents capacity accepted by [MailDelivery]. The caller must either
// deliver one request or release the reservation.
type MailDeliveryReservation interface {
	// Deliver transfers the reserved capacity and request to the delivery owner.
	Deliver(ctx context.Context, request *MailDeliveryRequest)
	// Release returns unused capacity to the delivery owner.
	Release()
}

// MailDelivery owns the bounded lifecycle of short-code email sends.
type MailDelivery struct {
	smtp MailDeliverySMTP

	admissionClosed chan struct{}
	drained         chan struct{}
	slots           chan struct{}

	mu     sync.Mutex
	closed bool
	wg     sync.WaitGroup
}

type mailDeliveryReservation struct {
	delivery *MailDelivery
	once     sync.Once
}

// NewMailDelivery creates an owner that accepts at most limit messages at once. A limit below one
// becomes one.
func NewMailDelivery(smtp MailDeliverySMTP, limit int) *MailDelivery {
	return &MailDelivery{
		smtp:            smtp,
		admissionClosed: make(chan struct{}),
		drained:         make(chan struct{}),
		slots:           make(chan struct{}, max(limit, 1)),
	}
}

// Reserve waits for delivery capacity within ctx. A successful reservation counts as accepted work
// until its Deliver or Release method completes ownership transfer.
func (delivery *MailDelivery) Reserve(ctx context.Context) (MailDeliveryReservation, error) {
	ctx, span := otel.Tracer().Start(ctx, "core.MailDelivery(reserve)")
	defer span.End()

	select {
	case delivery.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, otel.ReportError(span, errors.Join(ctx.Err(), ErrMailDeliveryUnavailable))
	case <-delivery.admissionClosed:
		return nil, otel.ReportError(span, ErrMailDeliveryUnavailable)
	}

	delivery.mu.Lock()
	defer delivery.mu.Unlock()

	if delivery.closed {
		<-delivery.slots

		return nil, otel.ReportError(span, ErrMailDeliveryUnavailable)
	}

	delivery.wg.Add(1)

	return otel.ReportSuccess(span, MailDeliveryReservation(&mailDeliveryReservation{delivery: delivery})), nil
}

// Close stops new reservations and wakes callers waiting for capacity. Accepted work continues.
func (delivery *MailDelivery) Close() {
	delivery.mu.Lock()
	defer delivery.mu.Unlock()

	if delivery.closed {
		return
	}

	delivery.closed = true
	close(delivery.admissionClosed)

	go func() {
		delivery.wg.Wait()
		close(delivery.drained)
	}()
}

// Wait closes admission and waits for accepted work to finish within ctx. It is safe to call
// repeatedly and concurrently.
func (delivery *MailDelivery) Wait(ctx context.Context) error {
	ctx, span := otel.Tracer().Start(ctx, "core.MailDelivery(wait)")
	defer span.End()

	delivery.Close()

	select {
	case <-delivery.drained:
		otel.ReportSuccessNoContent(span)

		return nil
	case <-ctx.Done():
		return otel.ReportError(span, fmt.Errorf("drain mail delivery: %w", ctx.Err()))
	}
}

func (reservation *mailDeliveryReservation) Deliver(ctx context.Context, request *MailDeliveryRequest) {
	ctx, span := otel.Tracer().Start(ctx, "core.MailDelivery(deliver)")
	defer span.End()

	reservation.once.Do(func() {
		go func() {
			defer reservation.delivery.complete()

			reservation.delivery.send(context.WithoutCancel(ctx), request)
		}()
	})

	otel.ReportSuccessNoContent(span)
}

func (reservation *mailDeliveryReservation) Release() {
	reservation.once.Do(reservation.delivery.complete)
}

func (delivery *MailDelivery) complete() {
	<-delivery.slots
	delivery.wg.Done()
}

func (delivery *MailDelivery) send(ctx context.Context, request *MailDeliveryRequest) {
	_, span := otel.Tracer().Start(ctx, "core.MailDelivery(send)")
	defer span.End()
	defer otel.RecoverPanic(ctx, span)

	span.SetAttributes(attribute.String("mail.kind", request.Kind))

	err := delivery.smtp.SendMail(request.To, request.Template, request.TemplateName, request.Data)
	if err != nil {
		otel.Logger().ErrorContext(ctx, "mail delivery failed", "mail.kind", request.Kind, "error", err)
		_ = otel.ReportError(span, err)

		return
	}

	otel.Logger().InfoContext(ctx, "mail delivered", "mail.kind", request.Kind)
	otel.ReportSuccessNoContent(span)
}
