package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	pgctx "github.com/a-novel-kit/context/pgbun"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
	servicesmocks "github.com/a-novel/authentication/internal/services/mocks"
	"github.com/a-novel/authentication/models"
)

func TestRegister(t *testing.T) { //nolint:paralleltest
	errFoo := errors.New("foo")

	type createCredentialsData struct {
		resp *dao.CredentialsEntity
		err  error
	}

	type issueTokenData struct {
		resp string
		err  error
	}

	type consumeShortCodeData struct {
		err error
	}

	testCases := []struct {
		name string

		request services.RegisterRequest

		createCredentialsData *createCredentialsData
		issueTokenData        *issueTokenData
		consumeShortCodeData  *consumeShortCodeData

		expect    string
		expectErr error
	}{
		{
			name: "Success",

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{},

			createCredentialsData: &createCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			issueTokenData: &issueTokenData{
				resp: "access-token",
			},

			expect: "access-token",
		},
		{
			name: "Error/ConsumeShortCode",

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/CreateCredentials",

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{},

			createCredentialsData: &createCredentialsData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/IssueToken",

			request: services.RegisterRequest{
				Email:     "user@provider.com",
				Password:  "password-2",
				ShortCode: "short-code",
			},

			consumeShortCodeData: &consumeShortCodeData{},

			createCredentialsData: &createCredentialsData{
				resp: &dao.CredentialsEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user@provider.com",
					Password:  "password-2-hashed",
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			issueTokenData: &issueTokenData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases { //nolint:paralleltest
		t.Run(testCase.name, func(t *testing.T) {
			ctx, err := pgctx.NewContext(t.Context(), nil)
			require.NoError(t, err)

			source := servicesmocks.NewMockRegisterSource(t)

			if testCase.consumeShortCodeData != nil {
				source.EXPECT().
					ConsumeShortCode(mock.Anything, services.ConsumeShortCodeRequest{
						Usage:  models.ShortCodeUsageRequestRegister,
						Target: testCase.request.Email,
						Code:   testCase.request.ShortCode,
					}).
					Return(nil, testCase.consumeShortCodeData.err)
			}

			if testCase.createCredentialsData != nil {
				source.EXPECT().
					InsertCredentials(mock.Anything, mock.MatchedBy(func(data dao.InsertCredentialsData) bool {
						return assert.Equal(t, testCase.request.Email, data.Email) &&
							assert.NotEqual(t, uuid.Nil, data.ID) &&
							assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
							assert.NoError(t, lib.CompareScrypt(testCase.request.Password, data.Password))
					})).
					Return(testCase.createCredentialsData.resp, testCase.createCredentialsData.err)
			}

			if testCase.issueTokenData != nil {
				source.EXPECT().
					IssueToken(mock.Anything, services.IssueTokenRequest{
						UserID: &testCase.createCredentialsData.resp.ID,
						Roles:  []models.Role{models.RoleUser},
					}).
					Return(testCase.issueTokenData.resp, testCase.issueTokenData.err)
			}

			service := services.NewRegisterService(source)

			resp, err := service.Register(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
