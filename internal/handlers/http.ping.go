package handlers

import (
	"net/http"

	"github.com/samber/lo"
	"go.opentelemetry.io/otel/codes"

	"github.com/a-novel/golib/otel"
)

type Ping struct{}

func NewPing() *Ping {
	return new(Ping)
}

func (handler *Ping) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer().Start(r.Context(), "api.Ping")
	defer span.End()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte("pong"))

	span.RecordError(err)
	span.SetStatus(lo.Ternary(err == nil, codes.Ok, codes.Error), "")
}
