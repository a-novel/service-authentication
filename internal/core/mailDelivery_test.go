package core_test

import (
	"context"
	"errors"
	"fmt"
	"net/textproto"
	"testing"
	"text/template"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/a-novel-kit/golib/smtp"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
)

func TestMailDelivery(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string
		exec func(t *testing.T)
	}{
		{
			name: "Success/BoundsAcceptedWork",
			exec: func(t *testing.T) {
				t.Helper()

				started := make(chan struct{}, 2)
				release := make(chan struct{}, 2)

				t.Cleanup(func() {
					for range 2 {
						select {
						case release <- struct{}{}:
						default:
						}
					}
				})

				smtpService := coremocks.NewMockMailDeliverySMTP(t)
				smtpService.EXPECT().
					SendMail(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(smtp.MailUsers, *template.Template, string, any) {
						started <- struct{}{}

						<-release
					}).
					Return(nil).
					Times(2)

				delivery := core.NewMailDelivery(smtpService, 2)

				for range 2 {
					reservation := reserveMailDelivery(t, delivery, t.Context())
					reservation.Deliver(t.Context(), newMailDeliveryRequest())
				}

				awaitMailDeliverySignal(t, started)
				awaitMailDeliverySignal(t, started)

				admissionCtx, cancel := context.WithCancel(t.Context())
				cancel()

				reservation, err := delivery.Reserve(admissionCtx)
				require.ErrorIs(t, err, context.Canceled)
				require.ErrorIs(t, err, core.ErrMailDeliveryUnavailable)
				require.Nil(t, reservation)

				release <- struct{}{}

				release <- struct{}{}

				waitForMailDelivery(t, delivery)
				smtpService.AssertExpectations(t)
			},
		},
		{
			name: "Success/SurvivesRequestCancellation",
			exec: func(t *testing.T) {
				t.Helper()

				started := make(chan struct{}, 1)
				release := make(chan struct{}, 1)

				t.Cleanup(func() {
					select {
					case release <- struct{}{}:
					default:
					}
				})

				smtpService := coremocks.NewMockMailDeliverySMTP(t)
				smtpService.EXPECT().
					SendMail(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(smtp.MailUsers, *template.Template, string, any) {
						started <- struct{}{}

						<-release
					}).
					Return(nil).
					Once()

				delivery := core.NewMailDelivery(smtpService, 1)
				requestCtx, cancelRequest := context.WithCancel(t.Context())
				reservation := reserveMailDelivery(t, delivery, requestCtx)
				reservation.Deliver(requestCtx, newMailDeliveryRequest())

				awaitMailDeliverySignal(t, started)
				cancelRequest()

				waitCtx, cancelWait := context.WithTimeout(t.Context(), 20*time.Millisecond)
				err := delivery.Wait(waitCtx)

				cancelWait()

				require.ErrorIs(t, err, context.DeadlineExceeded)

				release <- struct{}{}

				waitForMailDelivery(t, delivery)
				smtpService.AssertExpectations(t)
			},
		},
		{
			name: "Success/CloseWakesAdmission",
			exec: func(t *testing.T) {
				t.Helper()

				smtpService := coremocks.NewMockMailDeliverySMTP(t)
				delivery := core.NewMailDelivery(smtpService, 1)
				reservation := reserveMailDelivery(t, delivery, t.Context())

				admissionErr := make(chan error, 1)

				go func() {
					_, err := delivery.Reserve(t.Context())
					admissionErr <- err
				}()

				delivery.Close()
				delivery.Close()

				require.ErrorIs(t, awaitMailDeliveryError(t, admissionErr), core.ErrMailDeliveryUnavailable)

				reservation.Release()

				waitErrs := make(chan error, 2)
				go func() { waitErrs <- delivery.Wait(t.Context()) }()
				go func() { waitErrs <- delivery.Wait(t.Context()) }()

				require.NoError(t, awaitMailDeliveryError(t, waitErrs))
				require.NoError(t, awaitMailDeliveryError(t, waitErrs))
				smtpService.AssertExpectations(t)
			},
		},
		{
			name: "Success/RecoversSenderPanic",
			exec: func(t *testing.T) {
				t.Helper()

				smtpService := coremocks.NewMockMailDeliverySMTP(t)
				smtpService.EXPECT().
					SendMail(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Run(func(smtp.MailUsers, *template.Template, string, any) {
						panic("mail delivery exploded")
					}).
					Return(nil).
					Once()

				delivery := core.NewMailDelivery(smtpService, 1)
				reservation := reserveMailDelivery(t, delivery, t.Context())
				reservation.Deliver(t.Context(), newMailDeliveryRequest())
				reservation.Deliver(t.Context(), newMailDeliveryRequest())
				reservation.Release()

				waitForMailDelivery(t, delivery)
				smtpService.AssertExpectations(t)
			},
		},
		{
			name: "Error/ReportsSenderError",
			exec: func(t *testing.T) {
				t.Helper()

				smtpService := coremocks.NewMockMailDeliverySMTP(t)
				smtpService.EXPECT().
					SendMail(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(errFoo).
					Once()

				delivery := core.NewMailDelivery(smtpService, 1)
				reservation := reserveMailDelivery(t, delivery, t.Context())
				reservation.Deliver(t.Context(), newMailDeliveryRequest())

				err := delivery.Wait(t.Context())
				require.ErrorIs(t, err, errFoo)
				smtpService.AssertExpectations(t)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			testCase.exec(t)
		})
	}
}

//nolint:paralleltest // Replaces the process-wide tracer provider while mail goes out.
func TestMailDeliveryTelemetry(t *testing.T) {
	testCases := []struct {
		name    string
		smtpErr error

		expectStatus    codes.Code
		expectErrorType string
	}{
		{name: "Success", expectStatus: codes.Unset},
		{
			name:            "Error/Smtp",
			smtpErr:         fmt.Errorf("send mail: %w", &textproto.Error{Code: 451, Msg: "try again later"}),
			expectStatus:    codes.Error,
			expectErrorType: "*textproto.Error",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			previous := otel.GetTracerProvider()

			otel.SetTracerProvider(provider)
			t.Cleanup(func() {
				otel.SetTracerProvider(previous)
				require.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context())))
			})

			smtpService := coremocks.NewMockMailDeliverySMTP(t)
			smtpService.EXPECT().
				SendMail(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(testCase.smtpErr).
				Once()

			ctx, request := provider.Tracer("test").Start(t.Context(), "request")
			delivery := core.NewMailDelivery(smtpService, 1)
			reservation := reserveMailDelivery(t, delivery, ctx)
			reservation.Deliver(ctx, newMailDeliveryRequest())
			request.End()

			require.ErrorIs(t, delivery.Wait(t.Context()), testCase.smtpErr)
			smtpService.AssertExpectations(t)

			spans := make(map[string]sdktrace.ReadOnlySpan)
			for _, span := range recorder.Ended() {
				spans[span.Name()] = span
			}

			deliver, send := spans["core.MailDelivery(deliver)"], spans["core.MailDelivery(send)"]
			require.NotNil(t, deliver)
			require.NotNil(t, send)

			// The send roots its own trace, linked to the request's.
			require.Equal(t, request.SpanContext(), deliver.Parent())
			require.False(t, send.Parent().IsValid())
			require.NotEqual(t, request.SpanContext().TraceID(), send.SpanContext().TraceID())
			require.Len(t, send.Links(), 1)
			require.Equal(t, deliver.SpanContext(), send.Links()[0].SpanContext)

			attributes := attribute.NewSet(send.Attributes()...)
			errorType, _ := attributes.Value("error.type")

			require.Equal(t, testCase.expectStatus, send.Status().Code)
			require.Equal(t, testCase.expectErrorType, errorType.AsString())
		})
	}
}

func reserveMailDelivery(
	t *testing.T, delivery *core.MailDelivery, ctx context.Context,
) core.MailDeliveryReservation {
	t.Helper()

	reservation, err := delivery.Reserve(ctx)
	require.NoError(t, err)

	return reservation
}

func newMailDeliveryRequest() *core.MailDeliveryRequest {
	return &core.MailDeliveryRequest{
		To:           smtp.MailUsers{{Email: "user@provider.com"}},
		Template:     template.Must(template.New("test").Parse("test")),
		TemplateName: "test",
		Data:         map[string]any{"value": "test"},
		Kind:         "test",
	}
}

func awaitMailDeliverySignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("mail delivery did not reach the sender")
	}
}

func awaitMailDeliveryError(t *testing.T, errs <-chan error) error {
	t.Helper()

	select {
	case err := <-errs:
		return err
	case <-time.After(time.Second):
		t.Fatal("mail delivery operation did not return")

		return nil
	}
}
