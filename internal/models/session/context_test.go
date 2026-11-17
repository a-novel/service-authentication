package session_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

var (
	userClaims = &session.Claims{
		UserID: new(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
		Roles:  []string{"auth:user"},
	}
	anonymousClaims = &session.Claims{Roles: []string{"auth:anon"}}
)

// claimsContext returns a context carrying claims, or none when claims is nil.
func claimsContext(t *testing.T, claims *session.Claims) context.Context {
	t.Helper()

	if claims == nil {
		return t.Context()
	}

	return session.SetClaimsContext(t.Context(), claims)
}

func TestGetClaimsContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		claims *session.Claims
	}{
		// An optional-auth endpoint without a token gets no claims.
		{name: "Success/NoClaims"},
		{name: "Success/Claims", claims: userClaims},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.claims, session.GetClaimsContext(claimsContext(t, testCase.claims)))
		})
	}
}

func TestRequireClaimsContext(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		claims *session.Claims

		expect    *session.Claims
		expectErr error
	}{
		{name: "Success", claims: userClaims, expect: userClaims},
		{name: "Success/Anonymous", claims: anonymousClaims, expect: anonymousClaims},
		{name: "Error/MissingAuth", expectErr: session.ErrMissingAuth},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			claims, err := session.RequireClaimsContext(claimsContext(t, testCase.claims))
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, claims)
		})
	}
}

func TestRequestUser(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		claims *session.Claims

		expect    uuid.UUID
		expectErr error
	}{
		{name: "Success", claims: userClaims, expect: *userClaims.UserID},
		{name: "Error/MissingAuth", expectErr: session.ErrMissingAuth},
		{name: "Error/AnonymousCaller", claims: anonymousClaims, expectErr: session.ErrAnonymousCaller},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			userID, err := session.RequestUser(claimsContext(t, testCase.claims))
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, userID)
		})
	}
}

//nolint:paralleltest // Replaces the process-wide tracer provider.
func TestSetClaimsContext(t *testing.T) {
	provider := sdktrace.NewTracerProvider()
	previous := otel.GetTracerProvider()

	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previous)
		require.NoError(t, provider.Shutdown(context.WithoutCancel(t.Context())))
	})

	ctx, parent := provider.Tracer("test").Start(t.Context(), "parent")
	defer parent.End()

	ctx = session.SetClaimsContext(ctx, &session.Claims{Roles: []string{"user"}})

	_, child := provider.Tracer("test").Start(ctx, "child")
	defer child.End()

	require.Equal(t, parent.SpanContext(), child.(sdktrace.ReadOnlySpan).Parent())
}
