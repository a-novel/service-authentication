package api_test

import (
	"errors"
	"testing"

	"github.com/go-faster/jx"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-authentication/api"
	"github.com/a-novel/service-authentication/api/codegen"
	apimocks "github.com/a-novel/service-authentication/api/mocks"
	"github.com/a-novel/service-authentication/internal/services"
	"github.com/a-novel/service-authentication/models"
)

func TestListPublicKeys(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type listKeysData struct {
		res []*jwa.JWK
		err error
	}

	testCases := []struct {
		name string

		params codegen.ListPublicKeysParams

		listKeysData *listKeysData

		expect    codegen.ListPublicKeysRes
		expectErr error
	}{
		{
			name: "Success",

			params: codegen.ListPublicKeysParams{
				Usage: "test-usage",
			},

			listKeysData: &listKeysData{
				res: []*jwa.JWK{
					{
						JWKCommon: jwa.JWKCommon{
							KTY:    "test-kty",
							Use:    "test-use",
							KeyOps: []jwa.KeyOp{"test-keyops"},
							Alg:    "test-alg",
							KID:    "00000000-0000-0000-0000-000000000002",
						},
						Payload: []byte(`{"test":"payload-2"}`),
					},
					{
						JWKCommon: jwa.JWKCommon{
							KTY:    "test-kty",
							Use:    "test-use",
							KeyOps: []jwa.KeyOp{"test-keyops"},
							Alg:    "test-alg",
							KID:    "00000000-0000-0000-0000-000000000001",
						},
						Payload: []byte(`{"test":"payload-1"}`),
					},
				},
			},

			expect: lo.ToPtr(codegen.ListPublicKeysOKApplicationJSON([]codegen.JWK{
				{
					Kty:    "test-kty",
					Use:    "test-use",
					KeyOps: []codegen.KeyOp{"test-keyops"},
					Alg:    "test-alg",
					Kid: codegen.OptKID{
						Value: codegen.KID(uuid.MustParse("00000000-0000-0000-0000-000000000002")),
						Set:   true,
					},
					AdditionalProps: codegen.JWKAdditional{
						"test": jx.Raw(`"payload-2"`),
					},
				},
				{
					Kty:    "test-kty",
					Use:    "test-use",
					KeyOps: []codegen.KeyOp{"test-keyops"},
					Alg:    "test-alg",
					Kid: codegen.OptKID{
						Value: codegen.KID(uuid.MustParse("00000000-0000-0000-0000-000000000001")),
						Set:   true,
					},
					AdditionalProps: codegen.JWKAdditional{
						"test": jx.Raw(`"payload-1"`),
					},
				},
			})),
		},
		{
			name: "NoResults",

			params: codegen.ListPublicKeysParams{
				Usage: "test-usage",
			},

			listKeysData: &listKeysData{
				res: []*jwa.JWK{},
			},

			expect: lo.ToPtr(codegen.ListPublicKeysOKApplicationJSON([]codegen.JWK{})),
		},
		{
			name: "Error",

			params: codegen.ListPublicKeysParams{
				Usage: "test-usage",
			},

			listKeysData: &listKeysData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := apimocks.NewMockSearchKeysService(t)

			if testCase.listKeysData != nil {
				source.EXPECT().
					SearchKeys(t.Context(), services.SearchKeysRequest{
						Usage:   models.KeyUsage(testCase.params.Usage),
						Private: false,
					}).
					Return(testCase.listKeysData.res, testCase.listKeysData.err)
			}

			handler := api.API{SearchKeysService: source}

			res, err := handler.ListPublicKeys(t.Context(), testCase.params)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			source.AssertExpectations(t)
		})
	}
}
