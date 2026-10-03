package dao

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	// ErrWaitlistAlreadyJoined is returned when the email already has a pending invitation request.
	ErrWaitlistAlreadyJoined = errors.New("already on waitlist")
)

const (
	// WaitlistActionJoin appends a new pending address or reports a duplicate.
	WaitlistActionJoin = "join"
	// WaitlistActionRemove deletes rows matching one address or a bounded batch.
	WaitlistActionRemove = "remove"
	// WaitlistActionList reads a page for maintenance without modifying rows.
	WaitlistActionList = "list"
	// waitlistMaxResponseSize allows a full maintenance page of escaped emails.
	waitlistMaxResponseSize = 1 << 20
)

// WaitlistRequest is a signed operation for the repository-owned Google writer.
type WaitlistRequest struct {
	// Action selects join, remove, or list; these are private writer operations.
	Action string `json:"action"`
	// Email identifies the row. Comparison follows the auth service's case-sensitive contract.
	Email string `json:"email,omitempty"`
	// Lang is the preferred invitation language.
	Lang string `json:"lang,omitempty"`
	// After is the last email from the previous list page. Empty starts a new scan.
	After string `json:"after,omitempty"`
	// Emails selects a bounded bulk removal instead of the single Email field.
	Emails []string `json:"emails,omitempty"`
}

// WaitlistResult carries private maintenance data; joins return an empty result.
type WaitlistResult struct {
	// Emails is a sorted, unique page of pending addresses for list operations.
	Emails []string `json:"emails,omitempty"`
	// Removed counts rows actually deleted, including duplicate rows.
	Removed int `json:"removed,omitempty"`
}

// waitlistPayload binds the operation to a recent timestamp before signing.
type waitlistPayload struct {
	*WaitlistRequest

	Timestamp int64 `json:"timestamp"`
}

// waitlistEnvelope transports the exact signed JSON and its base64url HMAC-SHA256 signature.
type waitlistEnvelope struct {
	Payload   string `json:"payload"`
	Signature string `json:"signature"`
}

// waitlistResponse separates writer acknowledgements from caller-visible maintenance data.
type waitlistResponse struct {
	WaitlistResult

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
				switch {
				case len(via) > 1, request.Method != http.MethodGet:
					return ErrWaitlistUnavailable
				case request.URL.Scheme != "https", request.URL.Host != "script.googleusercontent.com":
					return ErrWaitlistUnavailable
				case request.URL.User != nil:
					return ErrWaitlistUnavailable
				}

				return nil
			},
		},
	}, nil
}

// Exec signs one writer operation. A disabled writer permits removals but rejects joins and maintenance scans.
func (writer *GoogleWaitlist) Exec(ctx context.Context, request *WaitlistRequest) (*WaitlistResult, error) {
	ctx, span := otel.Tracer().Start(ctx, "dao.GoogleWaitlist")
	defer span.End()

	if writer.config.URL == "" {
		if request.Action == WaitlistActionRemove {
			return &WaitlistResult{}, nil
		}

		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}

	payload, err := json.Marshal(waitlistPayload{WaitlistRequest: request, Timestamp: time.Now().Unix()})
	if err != nil {
		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}

	mac := hmac.New(sha256.New, []byte(writer.config.Secret))
	_, _ = mac.Write(payload) // hash.Hash.Write never returns an error.

	body, err := json.Marshal(waitlistEnvelope{
		Payload: string(payload), Signature: base64.RawURLEncoding.EncodeToString(mac.Sum(nil)),
	})
	if err != nil {
		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
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
	defer func() { _ = response.Body.Close() }() // Closing cannot change the writer's acknowledgement.

	var result waitlistResponse

	decodeErr := json.NewDecoder(io.LimitReader(response.Body, waitlistMaxResponseSize)).Decode(&result)
	if response.StatusCode != http.StatusOK || decodeErr != nil {
		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}

	switch result.Status {
	case "accepted":
		return &result.WaitlistResult, nil
	case "already_waitlisted":
		return nil, otel.ReportError(span, ErrWaitlistAlreadyJoined)
	case "busy":
		return nil, otel.ReportError(span, ErrWaitlistBusy)
	default:
		return nil, otel.ReportError(span, ErrWaitlistUnavailable)
	}
}
