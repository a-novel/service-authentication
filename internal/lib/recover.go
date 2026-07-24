package lib

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"go.opentelemetry.io/otel/trace"

	"github.com/a-novel-kit/golib/otel"
)

// errRecoveredPanic marks an error produced from a panic value absorbed by [RecoverPanic]. It is
// logged and span-reported only.
var errRecoveredPanic = errors.New("recovered panic")

// RecoverPanic stops a panic from escaping the calling goroutine, reporting it on span and logging
// it with a stack trace instead. Defer it directly: recover observes a panic only from a function
// the panicking goroutine deferred itself.
//
// The HTTP middleware chain recovers panics on handler goroutines only. Work detached from the
// request runs with nothing above it, and a panic that escapes a goroutine ends the whole process.
func RecoverPanic(ctx context.Context, span trace.Span) {
	rec := recover()
	if rec == nil {
		return
	}

	err := fmt.Errorf("%w: %v", errRecoveredPanic, rec)
	otel.Logger().ErrorContext(ctx, otel.ReportError(span, err).Error()+"\n"+string(debug.Stack()))
}
