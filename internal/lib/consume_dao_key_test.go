package lib_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
)

func TestConsumeDAOKey(t *testing.T) {
	t.Parallel()

	ctx, err := lib.NewMasterKeyContext(t.Context())
	require.NoError(t, err)

	testCases := []struct {
		name string

		key     *dao.KeyEntity
		private bool

		expect    *jwa.JWK
		expectErr error
	}{
		{
			name: "PublicKey",

			key: &dao.KeyEntity{
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
			name: "SymmetricKey",

			key: &dao.KeyEntity{
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
			name: "Private",

			private: true,

			key: &dao.KeyEntity{
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
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := lib.ConsumeDAOKey(ctx, testCase.key, testCase.private)
			require.ErrorIs(t, err, testCase.expectErr)

			// The json.RawMessage in the JWK payload causes trouble with Go comparison,
			// so instead we directly compare the JSON representations.
			jsonExpect, err := json.Marshal(testCase.expect)
			require.NoError(t, err)
			jsonResp, err := json.Marshal(result)
			require.NoError(t, err)

			require.JSONEq(t, string(jsonExpect), string(jsonResp))
		})
	}
}
