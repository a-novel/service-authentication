package dao_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/a-novel/service-authentication/internal/lib"
	"os"
	"testing"
)

var ctx context.Context

func init() {
	var err error

	//nolint:fatcontext
	ctx, err = lib.NewPostgresContext(context.Background(), os.Getenv("DAO_DSN"))
	if err != nil {
		panic(err)
	}
}

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
