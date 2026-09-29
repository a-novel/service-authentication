package dao_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

// waitlistTransport replaces only HTTP I/O, leaving request signing and http.Client redirects intact.
type waitlistTransport func(*http.Request) (*http.Response, error)

func (transport waitlistTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestGoogleWaitlist(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		status       int
		body         string
		redirect     string
		transportErr bool
		disabled     bool
		action       string
		expectErr    error
	}{
		{name: "SignedJoin", status: http.StatusOK, body: `{"status":"accepted"}`},
		{name: "ContentRedirect", status: http.StatusFound, redirect: "https://script.googleusercontent.com/response"},
		{
			name: "RejectForeignRedirect", status: http.StatusFound, redirect: "https://example.com/leak",
			expectErr: dao.ErrWaitlistUnavailable,
		},
		{
			name: "RejectPostReplay", status: http.StatusTemporaryRedirect,
			redirect: "https://script.googleusercontent.com/response", expectErr: dao.ErrWaitlistUnavailable,
		},
		{name: "Busy", status: http.StatusOK, body: `{"status":"busy"}`, expectErr: dao.ErrWaitlistBusy},
		{
			name: "BadSignature", status: http.StatusOK, body: `{"status":"unauthorized"}`,
			expectErr: dao.ErrWaitlistUnavailable,
		},
		{
			name: "HTTPError", status: http.StatusServiceUnavailable, body: `{"status":"accepted"}`,
			expectErr: dao.ErrWaitlistUnavailable,
		},
		{name: "Malformed", status: http.StatusOK, body: `{`, expectErr: dao.ErrWaitlistUnavailable},
		{
			name: "Oversized", status: http.StatusOK, body: strings.Repeat(" ", 128<<10) + `{"status":"accepted"}`,
			expectErr: dao.ErrWaitlistUnavailable,
		},
		{name: "NetworkErrorSanitized", transportErr: true, expectErr: dao.ErrWaitlistUnavailable},
		{name: "DisabledJoin", disabled: true, expectErr: dao.ErrWaitlistUnavailable},
		{name: "DisabledCleanup", disabled: true, action: "remove"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := authconfig.Waitlist{
				URL:     "https://script.google.com/macros/s/test-deployment/exec",
				Secret:  "test-only-key-not-a-deployed-secret",
				Timeout: time.Second,
			}
			if testCase.disabled {
				cfg = authconfig.Waitlist{}
			}

			calls := 0
			transport := waitlistTransport(func(request *http.Request) (*http.Response, error) {
				calls++

				if testCase.transportErr {
					return nil, errors.New("private response URL")
				}

				if calls == 1 {
					require.Equal(t, http.MethodPost, request.Method)
					require.Equal(t, "application/json", request.Header.Get("Content-Type"))

					var envelope struct {
						Payload   string `json:"payload"`
						Signature string `json:"signature"`
					}
					require.NoError(t, json.NewDecoder(request.Body).Decode(&envelope))
					require.NoError(t, request.Body.Close())
					require.NotContains(t, envelope.Payload, cfg.Secret)
					signature, err := base64.RawURLEncoding.DecodeString(envelope.Signature)
					require.NoError(t, err)

					mac := hmac.New(sha256.New, []byte(cfg.Secret))
					_, err = mac.Write([]byte(envelope.Payload))
					require.NoError(t, err)
					require.True(t, hmac.Equal(mac.Sum(nil), signature))

					var payload struct {
						Action    string `json:"action"`
						Email     string `json:"email"`
						Lang      string `json:"lang"`
						Timestamp int64  `json:"timestamp"`
					}
					require.NoError(t, json.Unmarshal([]byte(envelope.Payload), &payload))
					require.Equal(t, "join", payload.Action)
					require.Equal(t, "member@example.com", payload.Email)
					require.Equal(t, "fr", payload.Lang)
					require.WithinDuration(t, time.Now(), time.Unix(payload.Timestamp, 0), time.Second)

					return &http.Response{
						StatusCode: testCase.status, Header: http.Header{"Location": {testCase.redirect}},
						Body: io.NopCloser(strings.NewReader(testCase.body)), Request: request,
					}, nil
				}

				require.Equal(t, "ContentRedirect", testCase.name, "unsafe redirects must not be followed")
				require.Equal(t, http.MethodGet, request.Method)
				require.Nil(t, request.Body)

				return &http.Response{
					StatusCode: http.StatusOK, Header: make(http.Header),
					Body: io.NopCloser(strings.NewReader(`{"status":"accepted"}`)), Request: request,
				}, nil
			})
			writer, err := dao.NewGoogleWaitlist(cfg, transport)
			require.NoError(t, err)

			action := testCase.action
			if action == "" {
				action = "join"
			}

			err = writer.Exec(t.Context(), &dao.WaitlistRequest{
				Action: action, Email: "member@example.com", Lang: "fr",
			})
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr != nil {
				require.EqualError(t, err, testCase.expectErr.Error(), "Google details must stay out of errors and traces")
			}

			if testCase.disabled {
				require.Zero(t, calls)
			}
		})
	}

	t.Run("Cancelled", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		writer, err := dao.NewGoogleWaitlist(authconfig.Waitlist{
			URL: "https://script.google.com/macros/s/test/exec", Secret: strings.Repeat("x", 32), Timeout: time.Second,
		}, waitlistTransport(func(request *http.Request) (*http.Response, error) {
			return nil, request.Context().Err()
		}))
		require.NoError(t, err)
		err = writer.Exec(ctx, &dao.WaitlistRequest{Action: "join"})
		require.ErrorIs(t, err, dao.ErrWaitlistUnavailable)
	})
}
