package core_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestAuthorizationCodePurge(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		purged int64
		err    error
	}

	testCases := []struct {
		name string

		// daoMocks answer the batches in order.
		daoMocks []daoMock

		expect    *core.AuthorizationCodePurgeResult
		expectErr error
	}{
		{
			name: "Success/Empty",

			daoMocks: []daoMock{{purged: 0}},

			expect: &core.AuthorizationCodePurgeResult{Purged: 0},
		},
		{
			// A full batch may leave more expired rows, so the run continues until a short one.
			name: "Success/SeveralBatches",

			daoMocks: []daoMock{
				{purged: core.AuthorizationCodePurgeBatch},
				{purged: core.AuthorizationCodePurgeBatch},
				{purged: 3},
			},

			expect: &core.AuthorizationCodePurgeResult{Purged: 2*core.AuthorizationCodePurgeBatch + 3},
		},
		{
			// The rows already deleted are reported with the failure.
			name: "Error/Dao",

			daoMocks: []daoMock{
				{purged: core.AuthorizationCodePurgeBatch},
				{err: errFoo},
			},

			expect:    &core.AuthorizationCodePurgeResult{Purged: core.AuthorizationCodePurgeBatch},
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockAuthorizationCodePurgeDao(t)
			request := &dao.AuthorizationCodePurgeRequest{Limit: core.AuthorizationCodePurgeBatch}

			for _, daoMock := range testCase.daoMocks {
				var resp *dao.AuthorizationCodePurgeResult
				if daoMock.err == nil {
					resp = &dao.AuthorizationCodePurgeResult{Purged: daoMock.purged}
				}

				mockDao.EXPECT().Exec(mock.Anything, request).Return(resp, daoMock.err).Once()
			}

			service := core.NewAuthorizationCodePurge(mockDao)

			res, err := service.Exec(t.Context())
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			mockDao.AssertExpectations(t)
		})
	}
}
