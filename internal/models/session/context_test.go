package session_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/a-novel/service-authentication/v2/internal/models/session"
)

func TestGetClaimsContext(t *testing.T) {
	t.Parallel()

	t.Run("Success/NoClaimsSet", func(t *testing.T) {
		t.Parallel()

		// No value stored under ClaimsContextKey — optional-auth path: returns
		// (nil, nil) so endpoints that allow unauthenticated access can keep working.
		claims, err := session.GetClaimsContext(t.Context())
		require.NoError(t, err)
		require.Nil(t, claims)
	})

	t.Run("Success/ValidClaims", func(t *testing.T) {
		t.Parallel()

		want := &session.Claims{
			UserID: lo.ToPtr(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			Roles:  []string{"user"},
		}
		ctx := session.SetClaimsContext(t.Context(), want)

		got, err := session.GetClaimsContext(ctx)
		require.NoError(t, err)
		require.Equal(t, want, got)
	})

	t.Run("Error/UnexpectedClaims", func(t *testing.T) {
		t.Parallel()

		// A wrong type stored under ClaimsContextKey surfaces as ErrUnexpectedClaims.
		// An endpoint checking only `claims == nil` still fails closed.
		ctx := context.WithValue(t.Context(), session.ClaimsContextKey{}, "not-claims")

		claims, err := session.GetClaimsContext(ctx)
		require.ErrorIs(t, err, session.ErrUnexpectedClaims)
		require.Nil(t, claims)
	})
}

func TestMustGetClaimsContext(t *testing.T) {
	t.Parallel()

	claims := &session.Claims{Roles: []string{"user"}}

	testCases := []struct {
		name string

		// stored is the value under ClaimsContextKey; nil stores nothing.
		stored any

		expect    *session.Claims
		expectErr error
	}{
		{
			name:   "Success",
			stored: claims,
			expect: claims,
		},
		{
			name:      "Error/MissingAuth",
			expectErr: session.ErrMissingAuth,
		},
		{
			name:      "Error/UnexpectedClaims",
			stored:    "not-claims",
			expectErr: session.ErrUnexpectedClaims,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if testCase.stored != nil {
				ctx = context.WithValue(ctx, session.ClaimsContextKey{}, testCase.stored)
			}

			got, err := session.MustGetClaimsContext(ctx)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, got)
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
