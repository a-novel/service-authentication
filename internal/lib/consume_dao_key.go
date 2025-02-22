package lib

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt/jwa"

	"github.com/a-novel/authentication/internal/dao"
)

var ErrConsumeDAOKey = errors.New("ConsumeDAOKey")

func NewErrConsumeDAOKey(err error) error {
	return errors.Join(err, ErrConsumeDAOKey)
}

// ConsumeDAOKey converts a key from DAO entity to aJWK object.
func ConsumeDAOKey(ctx context.Context, key *dao.KeyEntity, private bool) (*jwa.JWK, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(
		// In case of a symmetric key, the public member will be nil, and the private member will be returned
		// instead.
		lo.Ternary(private || key.PublicKey == nil, key.PrivateKey, lo.FromPtr(key.PublicKey)),
	)
	if err != nil {
		return nil, NewErrConsumeDAOKey(fmt.Errorf("decode key: %w", err))
	}

	var deserialized *jwa.JWK

	err = lo.TernaryF(
		private || key.PublicKey == nil,
		// Private keys also needs to be decrypted.
		func() error { return DecryptMasterKey(ctx, decoded, &deserialized) },
		func() error { return json.Unmarshal(decoded, &deserialized) },
	)
	if err != nil {
		return nil, NewErrConsumeDAOKey(fmt.Errorf("deserialize key: %w", err))
	}

	return deserialized, nil
}
