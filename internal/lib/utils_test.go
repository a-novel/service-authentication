package lib_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/a-novel/service-authentication/internal/lib"
)

func mustEncryptValue(ctx context.Context, t *testing.T, data any) []byte {
	t.Helper()

	res, err := lib.EncryptMasterKey(ctx, data)
	if err != nil {
		panic(err)
	}

	return res
}

func mustEncryptBase64Value(ctx context.Context, t *testing.T, data any) string {
	t.Helper()

	res := mustEncryptValue(ctx, t, data)

	return base64.RawURLEncoding.EncodeToString(res)
}

func mustSerializeBase64Value(t *testing.T, data any) string {
	t.Helper()

	res, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return base64.RawURLEncoding.EncodeToString(res)
}
