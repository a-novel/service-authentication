package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestCredentialsUpdateLocationConsent(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	userID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	type errMock struct {
		err error
	}

	testCases := []struct {
		name string

		request *core.CredentialsUpdateLocationConsentRequest

		transaction       bool
		daoMock           *errMock
		clearLocationMock *errMock

		expectErr error
	}{
		{
			// Turning the consent on leaves the sessions as they are: they record their location from their
			// next access token on.
			name: "Success/On",

			request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID, Enabled: lo.ToPtr(true)},

			transaction: true,
			daoMock:     &errMock{},
		},
		{
			// Turning it off erases every stored location of the account, in the same transaction.
			name: "Success/Off",

			request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID, Enabled: lo.ToPtr(false)},

			transaction:       true,
			daoMock:           &errMock{},
			clearLocationMock: &errMock{},
		},
		{
			name: "Error/MissingEnabled",

			request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID},

			expectErr: core.ErrInvalidRequest,
		},
		{
			name: "Error/NotFound",

			request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID, Enabled: lo.ToPtr(false)},

			transaction: true,
			daoMock:     &errMock{err: dao.ErrCredentialsUpdateLocationConsentNotFound},

			expectErr: core.ErrCredentialsUpdateLocationConsentNotFound,
		},
		{
			// A failed erasure rolls the consent back.
			name: "Error/ClearLocations",

			request: &core.CredentialsUpdateLocationConsentRequest{UserID: userID, Enabled: lo.ToPtr(false)},

			transaction:       true,
			daoMock:           &errMock{},
			clearLocationMock: &errMock{err: errFoo},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockCredentialsUpdateLocationConsentDao(t)
			mockDaoClearLocations := coremocks.NewMockCredentialsUpdateLocationConsentDaoClearLocations(t)
			transactor := coremocks.NewMockTransactor(t)

			// Both writes run inside the transaction body, so a failure of either fails the whole.
			if testCase.transaction {
				transactor.EXPECT().WithinTx(mock.Anything, mock.Anything).
					RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).
					Once()
			}

			if testCase.daoMock != nil {
				request := &dao.CredentialsUpdateLocationConsentRequest{
					ID: userID, LocationConsent: *testCase.request.Enabled,
				}

				mockDao.EXPECT().Exec(mock.Anything, request).Return(&dao.Credentials{}, testCase.daoMock.err).Once()
			}

			if testCase.clearLocationMock != nil {
				mockDaoClearLocations.EXPECT().
					Exec(mock.Anything, &dao.SessionClearLocationsRequest{UserID: userID}).
					Return(testCase.clearLocationMock.err).
					Once()
			}

			service := core.NewCredentialsUpdateLocationConsent(mockDao, mockDaoClearLocations, transactor)

			err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			mockDao.AssertExpectations(t)
			mockDaoClearLocations.AssertExpectations(t)
			transactor.AssertExpectations(t)
		})
	}
}
