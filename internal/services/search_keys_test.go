package services_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
)

func TestSearchKeys(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context())
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type searchKeyData struct {
		resp []*dao.KeyEntity
		err  error
	}

	testCases := []struct {
		name string

		request services.SearchKeysRequest

		searchKeyData *searchKeyData

		expect    []*jwa.JWK
		expectErr error
	}{
		{
			name: "Success/Public",

			request: services.SearchKeysRequest{
				Usage: "test",
			},

			searchKeyData: &searchKeyData{
				resp: []*dao.KeyEntity{
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),

						PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000001",
							},
							Payload: []byte(
								`{"value":"private-key-1"}`,
							),
						}),
						PublicKey: lo.ToPtr(mustSerializeBase64Value(t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000001",
							},
							Payload: []byte(`{"value":"public-key-1"}`),
						})),
					},
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

						PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000002",
							},
							Payload: []byte(`{"value":"private-key-2"}`),
						}),
						PublicKey: lo.ToPtr(mustSerializeBase64Value(t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000002",
							},
							Payload: []byte(`{"value":"public-key-2"}`),
						})),
					},
				},
			},

			expect: []*jwa.JWK{
				{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: []jwa.KeyOp{"test-key-ops"},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000001",
					},
					Payload: []byte(`{"value":"public-key-1"}`),
				},
				{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: []jwa.KeyOp{"test-key-ops"},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000002",
					},
					Payload: []byte(`{"value":"public-key-2"}`),
				},
			},
		},
		{
			name: "Success/Public/Symmetric",

			request: services.SearchKeysRequest{
				Usage: "test",
			},

			searchKeyData: &searchKeyData{
				resp: []*dao.KeyEntity{
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),

						PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000001",
							},
							Payload: []byte(`{"value":"private-key-1"}`),
						}),
					},
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

						PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000002",
							},
							Payload: []byte(`{"value":"private-key-2"}`),
						}),
					},
				},
			},

			expect: []*jwa.JWK{
				{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: []jwa.KeyOp{"test-key-ops"},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000001",
					},
					Payload: []byte(`{"value":"private-key-1"}`),
				},
				{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: []jwa.KeyOp{"test-key-ops"},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000002",
					},
					Payload: []byte(`{"value":"private-key-2"}`),
				},
			},
		},
		{
			name: "Success/Private",

			request: services.SearchKeysRequest{
				Usage:   "test",
				Private: true,
			},

			searchKeyData: &searchKeyData{
				resp: []*dao.KeyEntity{
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),

						PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000001",
							},
							Payload: []byte(`{"value":"private-key-1"}`),
						}),
						PublicKey: lo.ToPtr(mustSerializeBase64Value(t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000001",
							},
							Payload: []byte(`{"value":"public-key-1"}`),
						})),
					},
					{
						ID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),

						PrivateKey: mustEncryptBase64Value(ctx, t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000002",
							},
							Payload: []byte(`{"value":"private-key-2"}`),
						}),
						PublicKey: lo.ToPtr(mustSerializeBase64Value(t, &jwa.JWK{
							JWKCommon: jwa.JWKCommon{
								KTY:    "test-kty",
								Use:    "test-use",
								KeyOps: []jwa.KeyOp{"test-key-ops"},
								Alg:    "test-alg",
								KID:    "00000000-0000-0000-0000-000000000002",
							},
							Payload: []byte(`{"value":"public-key-2"}`),
						})),
					},
				},
			},

			expect: []*jwa.JWK{
				{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: []jwa.KeyOp{"test-key-ops"},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000001",
					},
					Payload: []byte(`{"value":"private-key-1"}`),
				},
				{
					JWKCommon: jwa.JWKCommon{
						KTY:    "test-kty",
						Use:    "test-use",
						KeyOps: []jwa.KeyOp{"test-key-ops"},
						Alg:    "test-alg",
						KID:    "00000000-0000-0000-0000-000000000002",
					},
					Payload: []byte(`{"value":"private-key-2"}`),
				},
			},
		},
		{
			name: "SearchError",

			request: services.SearchKeysRequest{
				Usage: "test",
			},

			searchKeyData: &searchKeyData{
				err: errFoo,
			},

			expect:    []*jwa.JWK(nil),
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockSearchKeysSource(t)

			if testCase.searchKeyData != nil {
				source.EXPECT().
					SearchKeys(mock.Anything, testCase.request.Usage).
					Return(testCase.searchKeyData.resp, testCase.searchKeyData.err)
			}

			service := services.NewSearchKeysService(source)

			resp, err := service.SearchKeys(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			// The json.RawMessage in the JWK payload causes trouble with Go comparison,
			// so instead we directly compare the JSON representations.
			jsonExpect, err := json.Marshal(testCase.expect)
			require.NoError(t, err)
			jsonResp, err := json.Marshal(resp)
			require.NoError(t, err)

			require.JSONEq(t, string(jsonExpect), string(jsonResp))

			source.AssertExpectations(t)
		})
	}
}
