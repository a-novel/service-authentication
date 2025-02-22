package services

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/ed25519"

	"github.com/a-novel-kit/context"
	"github.com/a-novel-kit/jwt"
	"github.com/a-novel-kit/jwt/jwk"
	"github.com/a-novel-kit/jwt/jwp"
	"github.com/a-novel-kit/jwt/jws"

	"github.com/a-novel/authentication/config"
	"github.com/a-novel/authentication/models"
)

var ErrAuthenticateService = errors.New("AuthenticateService.Authenticate")

func NewErrAuthenticateService(err error) error {
	return errors.Join(err, ErrAuthenticateService)
}

// AuthenticateService is the service used to perform the AuthenticateService.Authenticate action.
//
// You may create one using the NewAuthenticateService function.
type AuthenticateService struct {
	recipient *jwt.Recipient
}

// Authenticate verifies the claims of an access token request and returns the claims if they are valid. Once
// validated, those claims can be trusted to perform authenticated actions.
//
// New claims can be issued using the IssueTokenService.
func (service *AuthenticateService) Authenticate(
	ctx context.Context, accessToken string,
) (*models.AccessTokenClaims, error) {
	// Don't bother trying to authenticate if the token is empty.
	if accessToken == "" {
		return nil, NewErrAuthenticateService(fmt.Errorf("token is empty: %w", models.ErrUnauthorized))
	}

	var claims models.AccessTokenClaims
	if err := service.recipient.Consume(ctx, accessToken, &claims); err != nil {
		if errors.Is(err, jws.ErrInvalidSignature) {
			return nil, NewErrAuthenticateService(fmt.Errorf("consume token: %w", models.ErrUnauthorized))
		}

		return nil, NewErrAuthenticateService(fmt.Errorf("consume token: %w", err))
	}

	return &claims, nil
}

func NewAuthenticateService(authVerifySource *jwk.Source[ed25519.PublicKey]) *AuthenticateService {
	verifier := jws.NewSourcedED25519Verifier(authVerifySource)

	deserializer := jwp.NewClaimsChecker(&jwp.ClaimsCheckerConfig{
		Checks: []jwp.ClaimsCheck{
			jwp.NewClaimsCheckTarget(jwt.TargetConfig{
				Issuer:   config.Tokens.Usages[models.KeyUsageAuth].Issuer,
				Audience: config.Tokens.Usages[models.KeyUsageAuth].Audience,
				Subject:  config.Tokens.Usages[models.KeyUsageAuth].Subject,
			}),
			jwp.NewClaimsCheckTimestamp(config.Tokens.Usages[models.KeyUsageAuth].Leeway, true),
		},
	})

	return &AuthenticateService{
		recipient: jwt.NewRecipient(jwt.RecipientConfig{
			Plugins:      []jwt.RecipientPlugin{verifier},
			Deserializer: deserializer.Unmarshal,
		}),
	}
}
