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

func TestShortCodeCreateEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type serviceCreateMock struct {
		resp *core.ShortCode
		err  error
	}

	type daoSelectMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeCreateEmailUpdateRequest

		serviceCreateMock *serviceCreateMock
		daoSelectMock     *daoSelectMock

		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
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

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},
		},
		{
			name: "Error/CreateShortCode",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/EmailAlreadyTaken",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			daoSelectMock: &daoSelectMock{
				err: nil,
			},

			expectErr: dao.ErrCredentialsUpdateEmailAlreadyExists,
		},
		{
			name: "Error/Dao",

			request: &core.ShortCodeCreateEmailUpdateRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				ID:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			},

			daoSelectMock: &daoSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			serviceCreate := coremocks.NewMockShortCodeCreateEmailUpdateService(t)
			daoSelect := coremocks.NewMockShortCodeCreateEmailUpdateDao(t)

			if testCase.serviceCreateMock != nil {
				shortCodeCreateRequest := &core.ShortCodeCreateRequest{
					Usage:    core.ShortCodeUsageValidateEmail,
					Target:   testCase.request.ID.String(),
					TTL:      authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageValidateEmail].TTL,
					Data:     testCase.request.Email,
					Override: true,
					Mail:     &core.ShortCodeCreateMail{Recipient: testCase.request.Email, Lang: testCase.request.Lang},
				}

				serviceCreate.EXPECT().
					Exec(mock.Anything, shortCodeCreateRequest).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(nil, testCase.daoSelectMock.err)
			}

			service := core.NewShortCodeCreateEmailUpdate(serviceCreate, daoSelect, authconfig.ShortCodesPresetDefault)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if err == nil {
				require.Equal(t, testCase.serviceCreateMock.resp, resp)
			}

			serviceCreate.AssertExpectations(t)
			daoSelect.AssertExpectations(t)
		})
	}
}
