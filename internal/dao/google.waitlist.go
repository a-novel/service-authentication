package dao

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/a-novel-kit/golib/otel"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
)

var (
	// ErrWaitlistUnavailable is returned when the sheet cannot acknowledge an operation.
	ErrWaitlistUnavailable = errors.New("waitlist unavailable")
	// ErrWaitlistBusy is returned when the shared writer cannot admit more work.
	ErrWaitlistBusy = errors.New("waitlist busy")
)

const waitlistMaxResponseSize = 128 << 10

// WaitlistRequest is a signed operation for the repository-owned Google writer.
type WaitlistRequest struct {
	// Action selects join or remove; these are private writer operations.
	Action string `json:"action"`
	// Email identifies the row. Comparison follows the auth service's case-sensitive contract.
	Email string `json:"email,omitempty"`
	// Lang is the preferred invitation language.
	Lang string `json:"lang,omitempty"`
}

// WaitlistResult acknowledges a mutation.
type WaitlistResult struct {
	// Status is the writer's acknowledgement, independent of its HTTP transport status.
	Status string `json:"status"`
}

// GoogleWaitlist calls the private sheet writer without exposing its secret or response data in errors.
type GoogleWaitlist struct {
	config authconfig.Waitlist
	client http.Client
}

// NewGoogleWaitlist validates the writer configuration and restricts response redirects to Google's content host.
// transport may be nil to use the standard HTTP transport.
func NewGoogleWaitlist(cfg authconfig.Waitlist, transport http.RoundTripper) (*GoogleWaitlist, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	return &GoogleWaitlist{
		config: cfg,
		client: http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
			CheckRedirect: func(request *http.Request, via []*http.Request) error {
				if len(via) > 1 || request.Method != http.MethodGet || request.URL.Scheme != "https" ||
					request.URL.Host != "script.googleusercontent.com" || request.URL.User != nil {
					return ErrWaitlistUnavailable
				}

				return nil
			},
		},
	}, nil
}

// Exec signs one idempotent writer operation. A disabled writer permits cleanup but rejects joins.
func (writer *GoogleWaitlist) Exec(ctx context.Context, request *WaitlistRequest) (*WaitlistResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.GoogleWaitlist")
	defer span.End()

	if writer.config.URL == "" {
		if request.Action == "remove" {
			return otel.ReportSuccess(span, &WaitlistResult{Status: "accepted"}), nil
		}

		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}

	payload, err := json.Marshal(struct {
		*WaitlistRequest

		Timestamp int64 `json:"timestamp"`
	}{request, time.Now().Unix()})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encode waitlist request: %w", err))
	}

	mac := hmac.New(sha256.New, []byte(writer.config.Secret))
	_, _ = mac.Write(payload) // hash.Hash.Write never returns an error.

	body, err := json.Marshal(struct {
		Payload   string `json:"payload"`
		Signature string `json:"signature"`
	}{string(payload), base64.RawURLEncoding.EncodeToString(mac.Sum(nil))})
	if err != nil {
		return nil, otel.ReportError(span, fmt.Errorf("encode waitlist envelope: %w", err))
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, writer.config.URL, bytes.NewReader(body))
	if err != nil {
		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}

	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := writer.client.Do(httpRequest)
	if err != nil {
		// HTTP errors can contain Google's one-time response URL. Keep it out of logs and traces.
		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}
	defer func() { _ = response.Body.Close() }()

	var result WaitlistResult

	decodeErr := json.NewDecoder(io.LimitReader(response.Body, waitlistMaxResponseSize)).Decode(&result)
	if response.StatusCode != http.StatusOK || decodeErr != nil {
		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}

	if result.Status == "busy" {
		return nil, otel.ReportError(span, ErrWaitlistBusy)
	}

	if result.Status != "accepted" {
		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}

	return otel.ReportSuccess(span, &result), nil
}
