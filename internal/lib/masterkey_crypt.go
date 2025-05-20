package lib

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/nacl/secretbox"
)

var (
	ErrInvalidSecret = errors.New("invalid secret")

	ErrEncryptMasterKey = errors.New("EncryptMasterKey")
	ErrDecryptMasterKey = errors.New("DecryptMasterKey")
)

func NewErrEncryptMasterKey(err error) error {
	return errors.Join(err, ErrEncryptMasterKey)
}

func NewErrDecryptMasterKey(err error) error {
	return errors.Join(err, ErrDecryptMasterKey)
}

// EncryptMasterKey encrypts the input using the master key.
func EncryptMasterKey(ctx context.Context, data any) ([]byte, error) {
	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return nil, NewErrEncryptMasterKey(fmt.Errorf("get master key: %w", err))
	}

	serializedData, err := json.Marshal(data)
	if err != nil {
		return nil, NewErrEncryptMasterKey(fmt.Errorf("serialize data: %w", err))
	}

	var nonce [24]byte
	if _, err = io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, NewErrEncryptMasterKey(fmt.Errorf("generate nonce: %w", err))
	}

	encrypted := secretbox.Seal(nonce[:], serializedData, &nonce, &secret)

	return encrypted, nil
}

// DecryptMasterKey decrypts the input using the master key.
func DecryptMasterKey(ctx context.Context, data []byte, output any) error {
	secret, err := MasterKeyContext(ctx)
	if err != nil {
		return fmt.Errorf("get master key: %w", err)
	}

	var decryptNonce [24]byte

	copy(decryptNonce[:], data[:24])

	decrypted, ok := secretbox.Open(nil, data[24:], &decryptNonce, &secret)
	if !ok {
		return NewErrDecryptMasterKey(fmt.Errorf("decrypt data: %w", ErrInvalidSecret))
	}

	if err = json.Unmarshal(decrypted, &output); err != nil {
		return NewErrDecryptMasterKey(fmt.Errorf("unmarshal data: %w", err))
	}

	return nil
}
