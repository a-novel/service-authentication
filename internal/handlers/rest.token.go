package handlers

import "github.com/a-novel/service-authentication/v2/internal/services"

type Token struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func loadToken(s *services.Token) Token {
	return Token{AccessToken: s.AccessToken, RefreshToken: s.RefreshToken}
}
