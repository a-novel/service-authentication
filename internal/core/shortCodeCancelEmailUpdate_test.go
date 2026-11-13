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

func TestShortCodeCancelEmailUpdate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	shortCodeID := uuid.MustParse("00000000-0000-0000-0000-0000000000a1")

	shortCode := &dao.ShortCode{
		ID:     shortCodeID,
		Usage:  core.ShortCodeUsageValidateEmail,
		Target: userID.String(),
		Data:   []byte(`"user1-new@email.com"`),
	}

	type daoSelectMock struct {
		resp *dao.ShortCode
		err  error
	}

	type daoMock struct {
		err error
	}

	testCases := []struct {
		name string

		daoSelectMock *daoSelectMock
		daoMock       *daoMock

		expectErr error
	}{
		{
			name: "Success",

			daoSelectMock: &daoSelectMock{resp: shortCode},
			daoMock:       &daoMock{},
		},
		{
			// The select skips expired and retired codes, so none of them awaits confirmation.
			name: "Error/NotFound",

			daoSelectMock: &daoSelectMock{err: dao.ErrShortCodeSelectNotFound},

			expectErr: core.ErrShortCodeNotFound,
		},
		{
			// A concurrent request confirmed or replaced the change, or its code expired, since the select.
			name: "Error/AlreadyDeleted",

			daoSelectMock: &daoSelectMock{resp: shortCode},
			daoMock:       &daoMock{err: dao.ErrShortCodeDeleteNotFound},

			expectErr: core.ErrShortCodeNotFound,
		},
		{
			name: "Error/Select",

			daoSelectMock: &daoSelectMock{err: errFoo},

			expectErr: errFoo,
		},
		{
			name: "Error/Delete",

			daoSelectMock: &daoSelectMock{resp: shortCode},
			daoMock:       &daoMock{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockShortCodeCancelEmailUpdateDao(t)
			mockDaoSelect := coremocks.NewMockShortCodeCancelEmailUpdateDaoSelect(t)

			selectRequest := &dao.ShortCodeSelectRequest{
				Usage:  core.ShortCodeUsageValidateEmail,
				Target: userID.String(),
			}

			mockDaoSelect.EXPECT().
				Exec(mock.Anything, selectRequest).
				Return(testCase.daoSelectMock.resp, testCase.daoSelectMock.err)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(data *dao.ShortCodeDeleteRequest) bool {
						return assert.Equal(t, shortCodeID, data.ID) &&
							assert.WithinDuration(t, time.Now(), data.Now, time.Second) &&
							assert.Equal(t, dao.ShortCodeDeleteCanceled, data.Comment)
					})).
					Return(nil, testCase.daoMock.err)
			}

			service := core.NewShortCodeCancelEmailUpdate(mockDao, mockDaoSelect)

			err := service.Exec(t.Context(), &core.ShortCodeCancelEmailUpdateRequest{UserID: userID})
			require.ErrorIs(t, err, testCase.expectErr)

			mockDao.AssertExpectations(t)
			mockDaoSelect.AssertExpectations(t)
		})
	}
}
