package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestShortCodeConsume(t *testing.T) {
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

	type daoDeleteMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeConsumeRequest

		daoSelectMock *daoSelectMock
		daoDeleteMock *daoDeleteMock

		expect    *core.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeConsumeRequest{
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

			daoDeleteMock: &daoDeleteMock{},

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

			request: &core.ShortCodeConsumeRequest{
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

			request: &core.ShortCodeConsumeRequest{
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

			request: &core.ShortCodeConsumeRequest{
				Target: "test-target",
				Usage:  core.ShortCodeUsageValidateEmail,
			},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "SelectError",

			request: &core.ShortCodeConsumeRequest{
				Target: "test-target",
				Usage:  core.ShortCodeUsageValidateEmail,
				Code:   shortCode,
			},

			daoSelectMock: &daoSelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "DeleteError",

			request: &core.ShortCodeConsumeRequest{
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

			daoDeleteMock: &daoDeleteMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoSelect := coremocks.NewMockShortCodeConsumeDaoSelect(t)
			daoDelete := coremocks.NewMockShortCodeConsumeDaoDelete(t)

			if testCase.daoSelectMock != nil {
				daoSelect.EXPECT().
					Exec(mock.Anything, &dao.ShortCodeSelectRequest{
						Target: testCase.request.Target,
						Usage:  testCase.request.Usage,
					}).
					Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)
			}

			if testCase.daoDeleteMock != nil {
				daoDelete.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(data *dao.ShortCodeDeleteRequest) bool {
						return assert.Equal(t, testCase.daoSelectMock.resp.ID, data.ID) &&
							assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
							assert.Equal(t, "key consumed", data.Comment)
					})).
					Return(nil, testCase.daoDeleteMock.err)
			}

			service := core.NewShortCodeConsume(daoSelect, daoDelete)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			daoSelect.AssertExpectations(t)
			daoDelete.AssertExpectations(t)
		})
	}
}
