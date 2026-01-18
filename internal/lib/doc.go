// Package lib provides cryptographic utilities for password hashing and random generation.
//
// Password hashing uses Argon2id (RFC 9106) with secure defaults recommended
// for password storage. The implementation follows OWASP guidelines for
// memory-hard password hashing.
//
// Random generation uses crypto/rand for cryptographically secure values,
// suitable for generating verification codes and other security-sensitive tokens.
package lib
