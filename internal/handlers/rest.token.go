package handlers

import "github.com/a-novel/service-authentication/v2/internal/core"

// Token is the access and refresh token pair returned by the token endpoints.
type Token struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func loadToken(s *core.Token) Token {
	return Token{AccessToken: s.AccessToken, RefreshToken: s.RefreshToken}
}
