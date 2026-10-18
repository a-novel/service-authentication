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
)

func TestShortCodeConsume(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	futureTime := time.Now().Add(time.Hour).Truncate(time.Second)

	type daoMock struct {
		resp *dao.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		daoMock *daoMock

		expect    *core.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			daoMock: &daoMock{
				resp: &dao.ShortCode{
					ID:        shortCodeID,
					Code:      "encrypted",
					Usage:     core.ShortCodeUsageValidateEmail,
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: futureTime,
				},
			},

			expect: &core.ShortCode{
				ID:        shortCodeID,
				Usage:     core.ShortCodeUsageValidateEmail,
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				ExpiresAt: futureTime,
			},
		},
		{
			// Another request retired the code after this one verified it.
			name: "Error/AlreadyConsumed",

			daoMock: &daoMock{
				err: dao.ErrShortCodeDeleteNotFound,
			},

			expectErr: core.ErrShortCodeConsumeInvalid,
		},
		{
			name: "Error/Delete",

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			daoDelete := coremocks.NewMockShortCodeConsumeDao(t)
			daoDelete.EXPECT().
				Exec(mock.Anything, mock.MatchedBy(func(data *dao.ShortCodeDeleteRequest) bool {
					return assert.Equal(t, shortCodeID, data.ID) &&
						assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
						assert.Equal(t, dao.ShortCodeDeleteConsumed, data.Comment)
				})).
				Return(testCase.daoMock.resp, testCase.daoMock.err)

			service := core.NewShortCodeConsume(daoDelete)

			resp, err := service.Exec(t.Context(), &core.ShortCodeConsumeRequest{ID: shortCodeID})
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			daoDelete.AssertExpectations(t)
		})
	}
}
