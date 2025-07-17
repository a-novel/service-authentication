package api_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/api"
	"github.com/a-novel/service-authentication/models/api"
)

func TestNewError(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	testCases := []struct {
		name string

		err error

		expect *apimodels.UnexpectedErrorStatusCode
	}{
		{
			name: "NoError",
		},
		{
			name: "SecurityError",

			err: &ogenerrors.SecurityError{
				Err: ogenerrors.ErrSecurityRequirementIsNotSatisfied,
			},

			expect: &apimodels.UnexpectedErrorStatusCode{
				StatusCode: http.StatusUnauthorized,
				Response:   apimodels.UnexpectedError{Error: "Unauthorized"},
			},
		},
		{
			name: "UnexpectedError",

			err: errFoo,

			expect: &apimodels.UnexpectedErrorStatusCode{
				StatusCode: http.StatusInternalServerError,
				Response:   apimodels.UnexpectedError{Error: "internal server error"},
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
