package lib

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"strings"
	"sync"

	"github.com/samber/lo"
	"golang.org/x/crypto/argon2"
)

var (
	// ErrInvalidHash is returned by [CompareArgon2] when the stored hash lacks the
	// encoded shape [GenerateArgon2] produces, which means the stored value is corrupted.
	ErrInvalidHash = errors.New("the encoded hash is in an invalid format")
	// ErrIncompatibleVersion is returned by [CompareArgon2] when the stored hash was
	// produced with an Argon2 version this binary cannot verify. Recovering requires
	// re-hashing the password once the user is authenticated by other means.
	ErrIncompatibleVersion = errors.New("the encoded hash is using an incompatible version of Argon2")
	// ErrInvalidPassword is returned by [CompareArgon2] when the stored hash is
	// well-formed but the supplied password does not match. Callers typically map it
	// to a 401 response.
	ErrInvalidPassword = errors.New("the password is invalid")
)

const (
	// Number of $-separated segments in an encoded Argon2id hash, checked before
	// decoding.
	argon2HashLen = 6
)

// argon2Params contains the parameters of an Argon2id hash.
type argon2Params struct {
	// Memory is the amount of memory used by the Argon2 algorithm (in kibibytes).
	Memory uint32
	// Iterations is the number of passes over the memory.
	Iterations uint32
	// Parallelism is the number of threads (or lanes) used by the algorithm.
	Parallelism uint8
	// SaltLength is the length of the random salt.
	SaltLength uint
	// KeyLength is the length of the generated key (or password hash).
	KeyLength uint32
}

// The second, lower-memory, Argon2id configuration RFC 9106 section 7.4 recommends: 3 passes over
// 64 MiB, here with a 32-byte salt and key.
const (
	argon2Iterations = 3
	argon2Memory     = 64 * 1024
	argon2SaltLength = 32
	argon2KeyLength  = 32
)

// argon2ParamsDefault hashes every password and short code. [GenerateArgon2] sets Parallelism to the
// host CPU count at hash time.
var argon2ParamsDefault = argon2Params{
	Memory:     argon2Memory,
	Iterations: argon2Iterations,
	SaltLength: argon2SaltLength,
	KeyLength:  argon2KeyLength,
}

// GenerateArgon2 hashes a password with Argon2id under a fresh random salt. It returns the
// standard encoded form, which [CompareArgon2] reads back:
//
//	$argon2id$v=<version>$m=<memory>,t=<iterations>,p=<lanes>$<salt>$<hash>
//
// The salt and the hash use unpadded standard base64.
func GenerateArgon2(password string) (string, error) {
	params := argon2ParamsDefault
	//nolint:gosec // min bounds the CPU count to uint8's range.
	params.Parallelism = uint8(min(runtime.NumCPU(), math.MaxUint8))

	salt := make([]byte, params.SaltLength)

	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := idKey(password, salt, &params)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// CompareArgon2 verifies a password against an encoded hash produced by
// [GenerateArgon2]. It returns nil on a match, [ErrInvalidPassword] on a mismatch,
// and [ErrInvalidHash] or [ErrIncompatibleVersion] when the stored hash cannot be
// decoded.
func CompareArgon2(password, encodedHash string) error {
	params, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return fmt.Errorf("decode hash: %w", err)
	}

	// Re-derive the key with the parameters and salt carried by the stored hash.
	otherHash := idKey(password, salt, params)

	// Constant time, so latency does not leak how much of the hash matched.
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return nil
	}

	return ErrInvalidPassword
}

// argon2Slots bounds the hashes running at once, one per CPU unless [LimitArgon2] resizes it:
// hashing is CPU-bound, so more concurrent hashes add memory but no throughput.
var argon2Slots = make(chan struct{}, runtime.GOMAXPROCS(0))

// LimitArgon2 caps the hashes running at once. Each allocates its memory parameter, 64 MiB by
// default, so size the cap to the instance's memory. Call it at startup, before any request.
func LimitArgon2(maxConcurrent int) {
	argon2Slots = make(chan struct{}, maxConcurrent)
}

// idKey derives the Argon2id key once a slot frees up, so a burst of logins waits instead of
// exhausting memory.
func idKey(password string, salt []byte, params *argon2Params) []byte {
	// A hash returns its slot to the gate it took it from.
	slots := argon2Slots
	slots <- struct{}{}

	defer func() { <-slots }()

	return argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
}

// dummyArgon2Hash is a throwaway Argon2id hash with the default parameters, so verifying against it
// costs the same as a real [CompareArgon2] call. It is built on first use. A binary that never
// compares against it, such as a consumer of the Go client, skips its 64 MiB hash.
var dummyArgon2Hash = sync.OnceValue(func() string {
	return lo.Must(GenerateArgon2("not-a-real-password"))
})

// PrepareDummyArgon2 builds the throwaway hash [DummyCompareArgon2] verifies against. Call it at
// startup: the build costs a hash of its own, and a build during a request slows that lookup miss.
func PrepareDummyArgon2() {
	dummyArgon2Hash()
}

// DummyCompareArgon2 runs a full Argon2id verification against a throwaway hash and discards the
// result. Call it on the "subject not found" branch of an authentication flow, such as an unknown
// email. A lookup miss then costs the same as a wrong password, so the response time reveals
// nothing about whether the subject exists.
func DummyCompareArgon2(password string) {
	_ = CompareArgon2(password, dummyArgon2Hash())
}

func decodeHash(encodedHash string) (*argon2Params, []byte, []byte, error) {
	values := strings.Split(encodedHash, "$")
	if len(values) != argon2HashLen {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int

	_, err := fmt.Sscanf(values[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse version: %w", errors.Join(ErrInvalidHash, err))
	}

	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	params := &argon2Params{}

	_, err = fmt.Sscanf(values[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse parameters: %w", errors.Join(ErrInvalidHash, err))
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(values[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decode salt: %w", errors.Join(ErrInvalidHash, err))
	}

	params.SaltLength = uint(len(salt))

	hash, err := base64.RawStdEncoding.Strict().DecodeString(values[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decode hash: %w", errors.Join(ErrInvalidHash, err))
	}

	rawHashLength := len(hash)
	if rawHashLength > math.MaxUint32 {
		return nil, nil, nil, fmt.Errorf("%w: hash length: %d", ErrInvalidHash, rawHashLength)
	}

	params.KeyLength = uint32(rawHashLength)

	return params, salt, hash, nil
}
