package apiclient

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/a-novel-kit/context"

	"github.com/a-novel/authentication/api/codegen"
	"github.com/a-novel/authentication/models"
)

var (
	ErrUnexpectedClaimsType = errors.New("unexpected claims type")
	ErrCheckSession         = errors.New("check session")
)

type SecurityHandlerService struct {
	endpoint string
}

func (security *SecurityHandlerService) Authenticate(
	ctx context.Context, accessToken string,
) (*models.AccessTokenClaims, error) {
	securitySource := NewSecuritySource()
	securitySource.SetToken(accessToken)

	client, err := codegen.NewClient(security.endpoint, securitySource)
	if err != nil {
		return nil, fmt.Errorf("connect auth client: %w", err)
	}

	rawClaims, err := client.CheckSession(ctx)
	if err != nil {
		return nil, errors.Join(err, ErrCheckSession)
	}

	claims, ok := rawClaims.(*codegen.Claims)
	if !ok {
		return nil, fmt.Errorf("%w: %T\n%v", ErrUnexpectedClaimsType, rawClaims, rawClaims)
	}

	userID, userIDOK := claims.GetUserID().Get()
	refreshTokenID, refreshTokenIDOK := claims.GetRefreshTokenID().Get()

	return &models.AccessTokenClaims{
		UserID: lo.Ternary(userIDOK, &userID, nil),
		Roles: lo.Map(claims.GetRoles(), func(item codegen.Role, _ int) models.Role {
			return models.Role(item)
		}),
		RefreshTokenID: lo.Ternary(refreshTokenIDOK, &refreshTokenID, nil),
	}, nil
}

func NewSecurityHandlerService(endpoint string) *SecurityHandlerService {
	return &SecurityHandlerService{endpoint: endpoint}
}
