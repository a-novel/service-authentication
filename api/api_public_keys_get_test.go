package api_test

import (
	"errors"
	"testing"

	"github.com/go-faster/jx"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/authentication/api"
	"github.com/a-novel/authentication/api/codegen"
	apimocks "github.com/a-novel/authentication/api/mocks"
	"github.com/a-novel/authentication/internal/dao"
)

func TestGetPublicKey(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type selectKeyData struct {
		res *jwa.JWK
		err error
	}

	testCases := []struct {
		name string

		params codegen.GetPublicKeyParams

		selectKeyData *selectKeyData

		expect    codegen.GetPublicKeyRes
		expectErr error
	}{
		{
			name: "Success",

			params: codegen.GetPublicKeyParams{
				Kid: codegen.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			selectKeyData: &selectKeyData{
				res: &jwa.JWK{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: []jwa.KeyOp{"test-keyops"},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000001",
					},
					Payload: []byte(`{"test":"payload"}`),
				},
			},

			expect: &codegen.JWK{
				Kty:    "test-kty",
				Use:    "test-use",
				KeyOps: []codegen.KeyOp{"test-keyops"},
				Alg:    "test-alg",
				Kid: codegen.OptKID{
					Value: codegen.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
					Set:   true,
				},
				AdditionalProps: codegen.JWKAdditional{
					"test": jx.Raw(`"payload"`),
				},
			},
		},
		{
			name: "KeyNotFound",

			params: codegen.GetPublicKeyParams{
				Kid: codegen.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			selectKeyData: &selectKeyData{
				err: dao.ErrKeyNotFound,
			},

			expect: &codegen.NotFoundError{Error: "key not found"},
		},
		{
			name: "Error",

			params: codegen.GetPublicKeyParams{
				Kid: codegen.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
			},

			selectKeyData: &selectKeyData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockSelectKeyService(t)

			if testCase.selectKeyData != nil {
				source.EXPECT().
					SelectKey(t.Context(), mock.AnythingOfType("services.SelectKeyRequest")).
					Return(testCase.selectKeyData.res, testCase.selectKeyData.err)
			}

			handler := api.API{SelectKeyService: source}

			res, err := handler.GetPublicKey(t.Context(), testCase.params)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
