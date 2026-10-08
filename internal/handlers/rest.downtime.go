package handlers

import (
	"net/http"

	"github.com/a-novel-kit/golib/downtime"
	"github.com/a-novel-kit/golib/httpf"
	"github.com/a-novel-kit/golib/otel"
)

// Downtime answers the planned downtime window, whichever services it lists, or null without one.
// The platform reads it to warn users before the window and to lock affected features during it.
type Downtime struct {
	window *downtime.Window
}

// NewDowntime returns a Downtime handler for the configured window.
func NewDowntime(window *downtime.Window) *Downtime {
	return &Downtime{window: window}
}

func (handler *Downtime) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, span := otel.Tracer().Start(r.Context(), "rest.Downtime")
	defer span.End()

	httpf.SendJSONStatus(ctx, w, span, http.StatusOK, handler.window)
}
