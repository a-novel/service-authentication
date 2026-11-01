package middlewares_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"strings"
	"sync"
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

	// send serves req on router, from the visitor and the user it names.
	send := func(t *testing.T, router http.Handler, req request) *httptest.ResponseRecorder {
		t.Helper()

		httpRequest := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/", strings.NewReader(body(req)))
		httpRequest.Header.Set("X-Client-Ip", req.visitor)

		if req.user != nil {
			httpRequest.Header.Set("X-User", req.user.String())
		}

		response := httptest.NewRecorder()
		router.ServeHTTP(response, httpRequest)

		return response
	}

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

			server := httpf.ServerConfigDefault
			server.RateLimitsEnabled = !testCase.disabled

			router := limitRouter(testCase.limit(middlewares.NewRestLimits(server, policies)))

			for _, req := range testCase.allowed {
				response := send(t, router, req)
				require.Equal(t, http.StatusOK, response.Code)
				require.JSONEq(t, body(req), response.Body.String())
			}

			if testCase.refused != nil {
				response := send(t, router, *testCase.refused)
				require.Equal(t, http.StatusTooManyRequests, response.Code)
				require.Equal(t, testCase.expectRetryAfter, response.Header().Get("Retry-After"))
			}
		})
	}

	// The test env runs the counter store on VALKEY_PORT, with the limiter's production ACL.
	t.Run("Success/Store", func(t *testing.T) {
		t.Parallel()

		port := os.Getenv("VALKEY_PORT")
		require.NotEmpty(t, port, "VALKEY_PORT names the test env's Valkey")

		proxy := newStoreProxy(t, "localhost:"+port)

		server := httpf.ServerConfigDefault
		server.RateLimitStore = httpf.RateLimitStore{Addr: proxy.addr(), Username: "limiter", Password: "limiter"}

		// Two instances, each with the limits cmd/rest builds as it starts.
		instances := []http.Handler{
			limitRouter(middlewares.NewRestLimits(server, policies).Login),
			limitRouter(middlewares.NewRestLimits(server, policies).Login),
		}

		// spendsOneBudget reports whether the instances spend one budget of 10 logins per minute for req.
		spendsOneBudget := func(req request) bool {
			for i := range 10 {
				if send(t, instances[i%2], req).Code != http.StatusOK {
					return false
				}
			}

			for _, instance := range instances {
				if send(t, instance, req).Code != http.StatusTooManyRequests {
					return false
				}
			}

			return true
		}

		// A store that answers after 100 ms sends an instance to its own counters, which a loaded runner
		// can cause, so the test retries. Each attempt takes a visitor and an email no earlier attempt or
		// run counted. A store the instances cannot use fails every attempt.
		var req request

		require.Eventually(t, func() bool {
			req = request{visitor: netip.AddrFrom16(uuid.New()).String(), email: uuid.NewString() + "@example.com"}

			return spendsOneBudget(req)
		}, 10*time.Second, 300*time.Millisecond)

		// Once the store stops, each instance counts the budget again, on its own.
		proxy.stop()

		for _, instance := range instances {
			for range 10 {
				require.Equal(t, http.StatusOK, send(t, instance, req).Code)
			}

			require.Equal(t, http.StatusTooManyRequests, send(t, instance, req).Code)
		}
	})
}

// limitRouter serves PUT / under limit, to the visitor the X-Client-Ip header names. The handler
// echoes the body, which must reach it whole after the email limit read it.
func limitRouter(limit func(http.Handler) http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.ClientIPFromHeader("X-Client-Ip"))
	router.Group(func(api chi.Router) {
		api.Use(signIn, limit)
		api.Put("/", func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.Copy(w, r.Body)
		})
	})

	return router
}

// signIn stands in for the auth middleware, signing in the user named by X-User.
func signIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(r.Header.Get("X-User"))
		if err == nil {
			claims := &session.Claims{UserID: &userID}
			r = r.WithContext(session.SetClaimsContext(r.Context(), claims))
		}

		next.ServeHTTP(w, r)
	})
}

// storeProxy forwards connections to a counter store until it stops. Stopping closes the proxy and
// every connection open through it, which the instances see as the store stopping.
type storeProxy struct {
	listener net.Listener
	store    string

	mu      sync.Mutex
	stopped bool
	conns   []net.Conn
}

// newStoreProxy forwards connections to the store at addr until the test ends.
func newStoreProxy(t *testing.T, addr string) *storeProxy {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	proxy := &storeProxy{listener: listener, store: addr}
	t.Cleanup(proxy.stop)

	go proxy.serve(t.Context())

	return proxy
}

// addr is the address the instances reach the store through.
func (proxy *storeProxy) addr() string {
	return proxy.listener.Addr().String()
}

// serve pipes each connection the proxy accepts to the store, until the proxy stops.
func (proxy *storeProxy) serve(ctx context.Context) {
	for {
		client, err := proxy.listener.Accept()
		if err != nil {
			return
		}

		store, err := (&net.Dialer{}).DialContext(ctx, "tcp", proxy.store)
		if err != nil {
			_ = client.Close()

			continue
		}

		if proxy.track(client, store) {
			go func() { _, _ = io.Copy(store, client) }()
			go func() { _, _ = io.Copy(client, store) }()
		}
	}
}

// track keeps conns for stop to close, and reports whether the proxy still runs. Once it stopped,
// track closes them instead.
func (proxy *storeProxy) track(conns ...net.Conn) bool {
	proxy.mu.Lock()
	defer proxy.mu.Unlock()

	if proxy.stopped {
		for _, conn := range conns {
			_ = conn.Close()
		}

		return false
	}

	proxy.conns = append(proxy.conns, conns...)

	return true
}

// stop closes the proxy and every connection open through it.
func (proxy *storeProxy) stop() {
	_ = proxy.listener.Close()

	proxy.mu.Lock()
	defer proxy.mu.Unlock()

	proxy.stopped = true

	for _, conn := range proxy.conns {
		_ = conn.Close()
	}
}
