package core

// Token is a signed token pair returned by the core token operations. Anonymous
// tokens (see TokenCreateAnon) fill only AccessToken and leave RefreshToken empty.
type Token struct {
	AccessToken  string
	RefreshToken string
}
