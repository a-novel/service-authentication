package middlewares_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/golib/httpf"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/handlers/middlewares"
	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestRestLimits(t *testing.T) {
	t.Parallel()

	userA := uuid.New()
	userB := uuid.New()
	accountA := uuid.NewString()
	accountB := uuid.NewString()

	// The production defaults, so each case reads as the budget a caller really gets.
	policies := authconfig.Limits{
		Strict:     httpf.Rate{Requests: 3, Window: 5 * time.Minute},
		LoginBurst: httpf.Rate{Requests: 10, Window: time.Minute},
		Login:      httpf.Rate{Requests: 30, Window: 10 * time.Minute},
		Mailbox:    httpf.Rate{Requests: 3, Window: time.Hour},

		WaitlistMaxConcurrent:  4,
		WaitlistMaxRequestSize: 8 << 10,
	}

	anonymous := func(limits middlewares.RestLimits) func(http.Handler) http.Handler { return limits.Anonymous }
	login := func(limits middlewares.RestLimits) func(http.Handler) http.Handler { return limits.Login }
	mail := func(limits middlewares.RestLimits) func(http.Handler) http.Handler { return limits.Mail }
	mailbox := func(limits middlewares.RestLimits) func(http.Handler) http.Handler { return limits.Mailbox }
	password := func(limits middlewares.RestLimits) func(http.Handler) http.Handler { return limits.Password }
	shortCode := func(limits middlewares.RestLimits) func(http.Handler) http.Handler { return limits.ShortCode }
	signUp := func(limits middlewares.RestLimits) func(http.Handler) http.Handler { return limits.SignUp }
	waitlist := func(limits middlewares.RestLimits) func(http.Handler) http.Handler { return limits.Waitlist }

	type request struct {
		visitor string
		user    *uuid.UUID
		email   string
		account string
	}

	body := func(req request) string { return fmt.Sprintf(`{"email":%q,"userID":%q}`, req.email, req.account) }

	// times repeats one request, so a case reads as the budget it spends.
	times := func(count int, req request) []request {
		requests := make([]request, count)
		for i := range requests {
			requests[i] = req
		}

		return requests
	}

	// visitors spreads count copies of one request over as many visitor IPs.
	visitors := func(count int, req request) []request {
		requests := make([]request, count)
		for i := range requests {
			requests[i] = req
			requests[i].visitor = fmt.Sprintf("192.0.2.%d", i+1)
		}

		return requests
	}

	visitor := request{visitor: "192.0.2.1"}

	testCases := []struct {
		name string

		disabled bool
		limit    func(limits middlewares.RestLimits) func(http.Handler) http.Handler

		// allowed requests all pass; refused, when set, then gets 429 with expectRetryAfter.
		allowed []request
		refused *request

		expectRetryAfter string
	}{
		{
			name: "Success/Disabled",

			disabled: true,
			limit:    anonymous,

			allowed: times(10, visitor),
		},
		{
			name: "Success/Anonymous/OtherVisitor",

			limit: anonymous,

			allowed: append(times(3, visitor), request{visitor: "192.0.2.2"}),
		},
		{
			name: "Success/Anonymous/OtherUser",

			limit: anonymous,

			allowed: append(
				times(3, request{visitor: "192.0.2.1", user: &userA}),
				request{visitor: "192.0.2.1", user: &userB},
			),
		},
		{
			name: "Success/Mailbox/OtherEmail",

			limit: mailbox,

			allowed: append(
				visitors(3, request{email: "a@example.com"}),
				request{visitor: "192.0.2.9", email: "b@example.com"},
			),
		},
		{
			name: "Error/Anonymous",

			limit: anonymous,

			allowed: times(3, visitor),
			refused: &visitor,

			expectRetryAfter: "300",
		},
		{
			// A signed-in user keeps one budget whatever address it calls from.
			name: "Error/Anonymous/UserAcrossVisitors",

			limit: anonymous,

			allowed: []request{
				{visitor: "192.0.2.1", user: &userA},
				{visitor: "192.0.2.2", user: &userA},
				{visitor: "192.0.2.3", user: &userA},
			},
			refused: &request{visitor: "192.0.2.4", user: &userA},

			expectRetryAfter: "300",
		},
		{
			name: "Error/Waitlist",

			limit: waitlist,

			allowed: times(3, request{visitor: "192.0.2.1", user: &userA}),
			refused: &request{visitor: "192.0.2.1", user: &userA},

			expectRetryAfter: "300",
		},
		{
			name: "Error/Login/Visitor",

			limit: login,

			allowed: times(10, request{visitor: "192.0.2.1", email: "a@example.com"}),
			refused: &request{visitor: "192.0.2.1", email: "b@example.com"},

			expectRetryAfter: "60",
		},
		{
			name: "Error/Login/Email",

			limit: login,

			allowed: visitors(30, request{email: "a@example.com"}),
			refused: &request{visitor: "192.0.2.99", email: "a@example.com"},

			expectRetryAfter: "600",
		},
		{
			name: "Error/SignUp/Visitor",

			limit: signUp,

			allowed: times(10, request{visitor: "192.0.2.1", email: "a@example.com"}),
			refused: &request{visitor: "192.0.2.1", email: "b@example.com"},

			expectRetryAfter: "60",
		},
		{
			name: "Error/SignUp/Email",

			limit: signUp,

			allowed: visitors(30, request{email: "a@example.com"}),
			refused: &request{visitor: "192.0.2.99", email: "a@example.com"},

			expectRetryAfter: "600",
		},
		{
			name: "Error/ShortCode/Visitor",

			limit: shortCode,

			allowed: times(10, request{visitor: "192.0.2.1", account: accountA}),
			refused: &request{visitor: "192.0.2.1", account: accountB},

			expectRetryAfter: "60",
		},
		{
			// Many visitors cannot gang up on one account, nor dodge the limit with letter case.
			name: "Error/ShortCode/Account",

			limit: shortCode,

			allowed: visitors(30, request{account: accountA}),
			refused: &request{visitor: "192.0.2.99", account: strings.ToUpper(accountA)},

			expectRetryAfter: "600",
		},
		{
			// A password change counts against the signed-in user, wherever it comes from.
			name: "Error/Password/User",

			limit: password,

			allowed: visitors(10, request{user: &userA}),
			refused: &request{visitor: "192.0.2.99", user: &userA},

			expectRetryAfter: "60",
		},
		{
			name: "Error/Mail/Caller",

			limit: mail,

			allowed: []request{
				{visitor: "192.0.2.1", email: "a@example.com"},
				{visitor: "192.0.2.1", email: "b@example.com"},
				{visitor: "192.0.2.1", email: "c@example.com"},
			},
			refused: &request{visitor: "192.0.2.1", email: "d@example.com"},

			expectRetryAfter: "300",
		},
		{
			// Many visitors cannot gang up on one mailbox, nor dodge the limit with letter case.
			name: "Error/Mail/Email",

			limit: mail,

			allowed: []request{
				{visitor: "192.0.2.1", email: "a@example.com"},
				{visitor: "192.0.2.2", email: "A@Example.com "},
				{visitor: "192.0.2.3", email: "a@example.com"},
			},
			refused: &request{visitor: "192.0.2.4", email: "a@EXAMPLE.com"},

			expectRetryAfter: "3600",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// signIn stands in for the auth middleware, signing in the user named by X-User.
			signIn := func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					userID, err := uuid.Parse(r.Header.Get("X-User"))
					if err == nil {
						claims := &session.Claims{UserID: &userID}
						r = r.WithContext(middlewares.SetClaimsContext(r.Context(), claims))
					}

					next.ServeHTTP(w, r)
				})
			}

			router := chi.NewRouter()
			router.Use(middleware.ClientIPFromHeader("X-Client-Ip"))
			router.Group(func(api chi.Router) {
				server := httpf.ServerConfigDefault
				server.RateLimitsEnabled = !testCase.disabled

				api.Use(signIn, testCase.limit(middlewares.NewRestLimits(server, policies)))
				// The handler echoes the body, which must reach it whole after the email limit read it.
				api.Put("/", func(w http.ResponseWriter, r *http.Request) {
					_, _ = io.Copy(w, r.Body)
				})
			})

			send := func(req request) *httptest.ResponseRecorder {
				httpRequest := httptest.NewRequestWithContext(
					t.Context(), http.MethodPut, "/", strings.NewReader(body(req)),
				)
				httpRequest.Header.Set("X-Client-Ip", req.visitor)

				if req.user != nil {
					httpRequest.Header.Set("X-User", req.user.String())
				}

				response := httptest.NewRecorder()
				router.ServeHTTP(response, httpRequest)

				return response
			}

			for _, req := range testCase.allowed {
				response := send(req)
				require.Equal(t, http.StatusOK, response.Code)
				require.JSONEq(t, body(req), response.Body.String())
			}

			if testCase.refused != nil {
				response := send(*testCase.refused)
				require.Equal(t, http.StatusTooManyRequests, response.Code)
				require.Equal(t, testCase.expectRetryAfter, response.Header().Get("Retry-After"))
			}
		})
	}
}
