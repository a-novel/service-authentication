package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
	"github.com/a-novel/service-authentication/v2/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/v2/internal/services/mocks"
)

func TestShortCodeConsume(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	futureTime := time.Now().Add(time.Hour).Truncate(time.Second)
	pastTime := time.Now().Add(-time.Hour).Truncate(time.Second)

	shortCode := "test-code"
	encrypted, err := lib.GenerateArgon2(shortCode, lib.Argon2ParamsDefault)
	require.NoError(t, err)

	type repositorySelectMock struct {
		resp *dao.ShortCode
		err  error
	}

	type repositoryDeleteMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *services.ShortCodeConsumeRequest

		repositorySelectMock *repositorySelectMock
		repositoryDeleteMock *repositoryDeleteMock

		expect    *services.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			request: &services.ShortCodeConsumeRequest{
				Target: "test-target",
				Usage:  services.ShortCodeUsageValidateEmail,
				Code:   shortCode,
			},

			repositorySelectMock: &repositorySelectMock{
				resp: &dao.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     services.ShortCodeUsageValidateEmail,
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: futureTime,
				},
			},

			repositoryDeleteMock: &repositoryDeleteMock{},

			expect: &services.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Usage:     services.ShortCodeUsageValidateEmail,
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				ExpiresAt: futureTime,
				PlainCode: shortCode,
			},
		},
		{
			name: "WrongCode",

			request: &services.ShortCodeConsumeRequest{
				Target: "test-target",
				Usage:  services.ShortCodeUsageValidateEmail,
				Code:   "fake-code",
			},

			repositorySelectMock: &repositorySelectMock{
				resp: &dao.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     services.ShortCodeUsageValidateEmail,
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: futureTime,
				},
			},

			expectErr: services.ErrShortCodeConsumeInvalid,
		},
		{
			name: "ExpiredCode",

			request: &services.ShortCodeConsumeRequest{
				Target: "test-target",
				Usage:  services.ShortCodeUsageValidateEmail,
				Code:   shortCode,
			},

			repositorySelectMock: &repositorySelectMock{
				resp: &dao.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     services.ShortCodeUsageValidateEmail,
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: pastTime,
				},
			},

			expectErr: services.ErrShortCodeConsumeExpired,
		},
		{
			name: "NoCode",

			request: &services.ShortCodeConsumeRequest{
				Target: "test-target",
				Usage:  services.ShortCodeUsageValidateEmail,
			},

			expectErr: services.ErrInvalidRequest,
		},
		{
			name: "SelectError",

			request: &services.ShortCodeConsumeRequest{
				Target: "test-target",
				Usage:  services.ShortCodeUsageValidateEmail,
				Code:   shortCode,
			},

			repositorySelectMock: &repositorySelectMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "DeleteError",

			request: &services.ShortCodeConsumeRequest{
				Target: "test-target",
				Usage:  services.ShortCodeUsageValidateEmail,
				Code:   shortCode,
			},

			repositorySelectMock: &repositorySelectMock{
				resp: &dao.ShortCode{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     services.ShortCodeUsageValidateEmail,
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: futureTime,
				},
			},

			repositoryDeleteMock: &repositoryDeleteMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repositorySelect := servicesmocks.NewMockShortCodeConsumeRepositorySelect(t)
			repositoryDelete := servicesmocks.NewMockShortCodeConsumeRepositoryDelete(t)

			if testCase.repositorySelectMock != nil {
				repositorySelect.EXPECT().
					Exec(mock.Anything, &dao.ShortCodeSelectRequest{
						Target: testCase.request.Target,
						Usage:  testCase.request.Usage,
					}).
					Return(testCase.repositorySelectMock.resp, testCase.repositorySelectMock.err)
			}

			if testCase.repositoryDeleteMock != nil {
				repositoryDelete.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(data *dao.ShortCodeDeleteRequest) bool {
						return assert.Equal(t, testCase.repositorySelectMock.resp.ID, data.ID) &&
							assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
							assert.Equal(t, "key consumed", data.Comment)
					})).
					Return(nil, testCase.repositoryDeleteMock.err)
			}

			service := services.NewShortCodeConsume(repositorySelect, repositoryDelete)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			repositorySelect.AssertExpectations(t)
			repositoryDelete.AssertExpectations(t)
		})
	}
}
