package lib

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"

	"github.com/a-novel-kit/context"
)

var ErrInvalidSecret = errors.New("invalid secret")

// EncryptMasterKey encrypts the input using the master key.
func EncryptMasterKey(ctx context.Context, data any) ([]byte, error) {
	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("(EncryptMasterKey) get master key: %w", err)
	}

	serializedData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("(EncryptMasterKey) serialize data: %w", err)
	}

	var nonce [24]byte
	if _, err = io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, fmt.Errorf("(EncryptMasterKey) generate nonce: %w", err)
	}

	encrypted := secretbox.Seal(nonce[:], serializedData, &nonce, &secret)

	return encrypted, nil
}

// DecryptMasterKey decrypts the input using the master key.
func DecryptMasterKey(ctx context.Context, data []byte, output any) error {
	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return fmt.Errorf("(DecryptMasterKey) get master key: %w", err)
	}

	var decryptNonce [24]byte

	copy(decryptNonce[:], data[:24])

	decrypted, ok := secretbox.Open(nil, data[24:], &decryptNonce, &secret)
	if !ok {
		return fmt.Errorf("(DecryptMasterKey) decrypt data: %w", ErrInvalidSecret)
	}

	if err = json.Unmarshal(decrypted, &output); err != nil {
		return fmt.Errorf("(DecryptMasterKey) unmarshal data: %w", err)
	}

	return nil
}
