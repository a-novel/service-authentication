package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	authconfig "github.com/a-novel/service-authentication/v2/internal/config/auth"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestCredentialsGetSelf(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	credentialsEntity := &dao.Credentials{
		ID:        userID,
		Email:     "user1@email.com",
		Role:      authconfig.RoleUser,
		CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	credentials := core.Credentials{
		ID:        userID,
		Email:     "user1@email.com",
		Role:      authconfig.RoleUser,
		CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type daoShortCodeSelectMock struct {
		resp *dao.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		daoMock                *daoMock
		daoShortCodeSelectMock *daoShortCodeSelectMock

		expect    *core.CredentialsSelf
		expectErr error
	}{
		{
			name: "Success",

			daoMock: &daoMock{
				resp: credentialsEntity,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				err: dao.ErrShortCodeSelectNotFound,
			},

			expect: &core.CredentialsSelf{Credentials: credentials},
		},
		{
			name: "Success/PendingEmail",

			daoMock: &daoMock{
				resp: credentialsEntity,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				resp: &dao.ShortCode{
					Usage:     core.ShortCodeUsageValidateEmail,
					Target:    userID.String(),
					Data:      []byte(`"user1-new@email.com"`),
					ExpiresAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},

			expect: &core.CredentialsSelf{
				Credentials: credentials,
				PendingEmail: &core.PendingEmail{
					Email:     "user1-new@email.com",
					ExpiresAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Error/NotFound",

			daoMock: &daoMock{
				err: dao.ErrCredentialsSelectNotFound,
			},

			expectErr: core.ErrCredentialsGetNotFound,
		},
		{
			name: "Error/Credentials",

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/ShortCode",

			daoMock: &daoMock{
				resp: credentialsEntity,
			},
			daoShortCodeSelectMock: &daoShortCodeSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			mockDao := coremocks.NewMockCredentialsGetSelfDao(t)
			mockDaoShortCodeSelect := coremocks.NewMockCredentialsGetSelfDaoShortCodeSelect(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectRequest{ID: userID}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			if testCase.daoShortCodeSelectMock != nil {
				shortCodeRequest := &dao.ShortCodeSelectRequest{
					Usage:  core.ShortCodeUsageValidateEmail,
					Target: userID.String(),
				}

				mockDaoShortCodeSelect.EXPECT().
					Exec(mock.Anything, shortCodeRequest).
					Return(testCase.daoShortCodeSelectMock.resp, testCase.daoShortCodeSelectMock.err)
			}

			service := core.NewCredentialsGetSelf(mockDao, mockDaoShortCodeSelect)

			resp, err := service.Exec(ctx, &core.CredentialsGetSelfRequest{ID: userID})
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
			mockDaoShortCodeSelect.AssertExpectations(t)
		})
	}
}
