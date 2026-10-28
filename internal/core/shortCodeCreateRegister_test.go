package core_test

import (
	"encoding/json"
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

func TestShortCodeCreateRegister(t *testing.T) {
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

		request *core.ShortCodeCreateRegisterRequest

		serviceCreateMock *serviceCreateMock
		daoSelectMock     *daoSelectMock

		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				Role:  authconfig.RoleUser,
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
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

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				Role:  authconfig.RoleUser,
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrCredentialsSelectByEmailNotFound,
			},

			serviceCreateMock: &serviceCreateMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "Error/EmailAlreadyTaken",

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				Role:  authconfig.RoleUser,
			},

			daoSelectMock: &daoSelectMock{
				err: nil,
			},

			expectErr: dao.ErrCredentialsInsertAlreadyExists,
		},
		{
			name: "Error/Dao",

			request: &core.ShortCodeCreateRegisterRequest{
				Lang:  authconfig.LangFR,
				Email: "user@provider.com",
				Role:  authconfig.RoleUser,
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

			serviceCreate := coremocks.NewMockShortCodeCreateRegisterService(t)
			daoSelect := coremocks.NewMockShortCodeCreateRegisterDao(t)

			if testCase.serviceCreateMock != nil {
				serviceCreate.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(request *core.ShortCodeCreateRequest) bool {
						data, err := json.Marshal(request.Data)

						return err == nil &&
							request.Usage == core.ShortCodeUsageRegister &&
							request.Target == testCase.request.Email &&
							request.TTL == authconfig.ShortCodesPresetDefault.Usages[core.ShortCodeUsageRegister].TTL &&
							request.Override &&
							*request.Mail == core.ShortCodeCreateMail{Recipient: testCase.request.Email, Lang: testCase.request.Lang} &&
							string(data) == `{"role":"`+testCase.request.Role+`"}`
					})).
					Return(testCase.serviceCreateMock.resp, testCase.serviceCreateMock.err)
			}

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.CredentialsSelectByEmailRequest{
						Email: testCase.request.Email,
					}).
					Return(nil, testCase.daoSelectMock.err)
			}

			service := core.NewShortCodeCreateRegister(serviceCreate, daoSelect, authconfig.ShortCodesPresetDefault)

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
