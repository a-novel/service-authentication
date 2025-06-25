package services_test

import (
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
)

func TestSelectKey(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context())
	require.NoError(t, err)

	errFoo := errors.New("foo")

	type selectKeyData struct {
		resp *dao.KeyEntity
		err  error
	}

	testCases := []struct {
		name string

		request services.SelectKeyRequest

		selectKeyData *selectKeyData

		expect    *jwa.JWK
		expectErr error
	}{
		{
			name: "Success/Public",

			request: services.SelectKeyRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			selectKeyData: &selectKeyData{
				resp: &dao.KeyEntity{
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
			},

			expect: &jwa.JWK{
				JWKCommon: jwa.JWKCommon{
					KTY:    "test-kty",
					Use:    "test-use",
					KeyOps: []jwa.KeyOp{"test-key-ops"},
					Alg:    "test-alg",
					KID:    "00000000-0000-0000-0000-000000000001",
				},
				Payload: []byte(`{"value":"public-key-1"}`),
			},
		},
		{
			name: "Success/Public/Symmetric",

			request: services.SelectKeyRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			selectKeyData: &selectKeyData{
				resp: &dao.KeyEntity{
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
			},

			expect: &jwa.JWK{
				JWKCommon: jwa.JWKCommon{
					KTY:    "test-kty",
					Use:    "test-use",
					KeyOps: []jwa.KeyOp{"test-key-ops"},
					Alg:    "test-alg",
					KID:    "00000000-0000-0000-0000-000000000001",
				},
				Payload: []byte(`{"value":"private-key-1"}`),
			},
		},
		{
			name: "Success/Private",

			request: services.SelectKeyRequest{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Private: true,
			},

			selectKeyData: &selectKeyData{
				resp: &dao.KeyEntity{
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
			},

			expect: &jwa.JWK{
				JWKCommon: jwa.JWKCommon{
					KTY:    "test-kty",
					Use:    "test-use",
					KeyOps: []jwa.KeyOp{"test-key-ops"},
					Alg:    "test-alg",
					KID:    "00000000-0000-0000-0000-000000000001",
				},
				Payload: []byte(`{"value":"private-key-1"}`),
			},
		},
		{
			name: "SearchError",

			request: services.SelectKeyRequest{
				ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

			source := servicesmocks.NewMockSelectKeySource(t)

			if testCase.selectKeyData != nil {
				source.EXPECT().
					SelectKey(mock.Anything, testCase.request.ID).
					Return(testCase.selectKeyData.resp, testCase.selectKeyData.err)
			}

			service := services.NewSelectKeyService(source)

			resp, err := service.SelectKey(ctx, testCase.request)
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
