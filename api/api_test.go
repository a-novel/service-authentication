package api_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
)

func TestNewError(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		err error

		expect *codegen.UnexpectedErrorStatusCode
	}{
		{
			name: "NoError",
		},
		{
			name: "SecurityError",

			err: &ogenerrors.SecurityError{
				Err: ogenerrors.ErrSecurityRequirementIsNotSatisfied,
			},

			expect: &codegen.UnexpectedErrorStatusCode{
				StatusCode: http.StatusUnauthorized,
				Response:   codegen.UnexpectedError{Error: "Unauthorized"},
			},
		},
		{
			name: "UnexpectedError",

			err: errFoo,

			expect: &codegen.UnexpectedErrorStatusCode{
				StatusCode: http.StatusInternalServerError,
				Response:   codegen.UnexpectedError{Error: "internal server error"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			res := new(api.API).NewError(t.Context(), testCase.err)
			require.Equal(t, testCase.expect, res)
		})
	}
}
