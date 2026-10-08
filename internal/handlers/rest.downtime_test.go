package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/downtime"

	"github.com/a-novel/service-authentication/v2/internal/handlers"
)

func TestDowntime(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		window *downtime.Window

		expectBody string
	}{
		{
			name:       "NoWindow",
			expectBody: "null",
		},
		{
			name: "Window",
			window: &downtime.Window{
				Services: []string{"json-keys"},
				Start:    time.Date(2026, 10, 12, 6, 0, 0, 0, time.UTC),
				End:      time.Date(2026, 10, 12, 7, 0, 0, 0, time.UTC),
			},
			expectBody: `{"services":["json-keys"],"start":"2026-10-12T06:00:00Z","end":"2026-10-12T07:00:00Z"}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			handlers.NewDowntime(testCase.window).
				ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v2/downtime", nil))

			require.Equal(t, http.StatusOK, w.Code)
			require.JSONEq(t, testCase.expectBody, w.Body.String())
		})
	}
}
