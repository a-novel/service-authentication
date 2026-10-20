package handlers

import (
	"net/http"

	"github.com/a-novel-kit/golib/otel"
)

// RestPing is the REST handler that responds with "pong" for liveness checks.
type RestPing struct{}

func NewRestPing() *RestPing {
	return &RestPing{}
}

func (handler *RestPing) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, span := otel.Tracer().Start(r.Context(), "rest.Ping")
	defer span.End()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	_, err := w.Write([]byte("pong"))
	if err != nil {
		_ = otel.ReportError(span, err)
	}
}
