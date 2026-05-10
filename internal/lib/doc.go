// Package lib holds the cryptographic primitives used by the authentication
// service: Argon2id password hashing (RFC 9106) and URL-safe random-string
// generation backed by crypto/rand. It depends on no other internal package
// and is the lowest layer in the service.
package lib
