package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestShortCodeVerify(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	futureTime := time.Now().Add(time.Hour).Truncate(time.Second)
	pastTime := time.Now().Add(-time.Hour).Truncate(time.Second)

	shortCode := "test-code"
	encrypted, err := lib.GenerateArgon2(shortCode, lib.Argon2ParamsDefault)
	require.NoError(t, err)

	type daoSelectMock struct {
		resp *dao.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeVerifyRequest

		daoSelectMock *daoSelectMock

		expect    *core.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeVerifyRequest{
				Target: "test-target",
				Usage:  core.ShortCodeUsageValidateEmail,
				Code:   shortCode,
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     core.ShortCodeUsageValidateEmail,
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: futureTime,
				},
			},

			expect: &core.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Usage:     core.ShortCodeUsageValidateEmail,
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				ExpiresAt: futureTime,
				PlainCode: shortCode,
			},
		},
		{
			name: "WrongCode",

			request: &core.ShortCodeVerifyRequest{
				Target: "test-target",
				Usage:  core.ShortCodeUsageValidateEmail,
				Code:   "fake-code",
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     core.ShortCodeUsageValidateEmail,
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: futureTime,
				},
			},

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
		{
			name: "ExpiredCode",

			request: &core.ShortCodeVerifyRequest{
				Target: "test-target",
				Usage:  core.ShortCodeUsageValidateEmail,
				Code:   shortCode,
			},

			daoSelectMock: &daoSelectMock{
				resp: &dao.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     core.ShortCodeUsageValidateEmail,
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: pastTime,
				},
			},

			expectErr: core.ErrShortCodeConsumeExpired,
		},
		{
			name: "NoCode",

			request: &core.ShortCodeVerifyRequest{
				Target: "test-target",
				Usage:  core.ShortCodeUsageValidateEmail,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "NotFound",

			request: &core.ShortCodeVerifyRequest{
				Target: "test-target",
				Usage:  core.ShortCodeUsageValidateEmail,
				Code:   shortCode,
			},

			daoSelectMock: &daoSelectMock{
				err: dao.ErrShortCodeSelectNotFound,
			},

			expectErr: core.ErrShortCodeNotFound,
		},
		{
			name: "SelectError",

			request: &core.ShortCodeVerifyRequest{
				Target: "test-target",
				Usage:  core.ShortCodeUsageValidateEmail,
				Code:   shortCode,
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

			daoSelect := coremocks.NewMockShortCodeVerifyDao(t)

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.ShortCodeSelectRequest{
						Target: testCase.request.Target,
						Usage:  testCase.request.Usage,
					}).
					Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)
			}

			service := core.NewShortCodeVerify(daoSelect)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			daoSelect.AssertExpectations(t)
		})
	}
}
