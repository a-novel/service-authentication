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

func TestShortCodeCreatePasswordReset(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.Credentials
		err  error
	}

	type serviceCreateMock struct {
		resp *core.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeCreatePasswordResetRequest

		daoMock           *daoMock
		serviceCreateMock *serviceCreateMock

		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeCreatePasswordResetRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			serviceCreateMock: &serviceCreateMock{
				resp: &core.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					PlainCode: "abcdef123456",
				},
			},
		},
		{
			name: "Error/CreateShortCode",

			request: &core.ShortCodeCreatePasswordResetRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
			},

			daoMock: &daoMock{
				resp: &dao.Credentials{
					ID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				},
			},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/SelectEmail",

			request: &core.ShortCodeCreatePasswordResetRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
			},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			serviceCreate := coremocks.NewMockShortCodeCreatePasswordResetService(t)
			mockDao := coremocks.NewMockShortCodeCreatePasswordResetDao(t)

			if testCase.serviceCreateMock != nil {
				shortCodeCreateRequest := &core.ShortCodeCreateRequest{
					Usage:    core.ShortCodeUsageResetPassword,
					Target:   testCase.daoMock.resp.ID.String(),
					TTL:      authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageResetPassword].TTL,
					Override: true,
					Mail:     &core.ShortCodeCreateMail{Recipient: testCase.daoMock.resp.Email, Lang: testCase.request.Lang},
				}

				serviceCreate.EXPECT().
					Exec(mock.Anything, shortCodeCreateRequest).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewShortCodeCreatePasswordReset(serviceCreate, mockDao, authconfig.ShortCodesPresetDefault)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if err == nil {
				require.Equal(t, testCase.serviceCreateMock.resp, resp)
			}

			serviceCreate.AssertExpectations(t)
			mockDao.AssertExpectations(t)
		})
	}
}
