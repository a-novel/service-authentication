package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestRefreshTokenRevocationPurge(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		purged int64
		err    error
	}

	testCases := []struct {
		name string

		// daoMocks answer the batches of revocations in order, and sessionsMocks those of sessions.
		daoMocks      []daoMock
		sessionsMocks []daoMock

		expect    *core.RefreshTokenRevocationPurgeResult
		expectErr error
	}{
		{
			name: "Success/Empty",

			daoMocks:      []daoMock{{purged: 0}},
			sessionsMocks: []daoMock{{purged: 0}},

			expect: &core.RefreshTokenRevocationPurgeResult{Purged: 0},
		},
		{
			// A full batch may leave more expired rows, so the run continues until a short one.
			name: "Success/SeveralBatches",

			daoMocks: []daoMock{
				{purged: core.RefreshTokenRevocationPurgeBatch},
				{purged: core.RefreshTokenRevocationPurgeBatch},
				{purged: 3},
			},
			sessionsMocks: []daoMock{
				{purged: core.RefreshTokenRevocationPurgeBatch},
				{purged: 5},
			},

			expect: &core.RefreshTokenRevocationPurgeResult{
				Purged:         2*core.RefreshTokenRevocationPurgeBatch + 3,
				SessionsPurged: core.RefreshTokenRevocationPurgeBatch + 5,
			},
		},
		{
			// The rows already deleted are reported with the failure.
			name: "Error/Dao",

			daoMocks: []daoMock{
				{purged: core.RefreshTokenRevocationPurgeBatch},
				{err: errFoo},
			},

			expect:    &core.RefreshTokenRevocationPurgeResult{Purged: core.RefreshTokenRevocationPurgeBatch},
			expectErr: errFoo,
		},
		{
			// The revocations already purged are reported with the failure on sessions.
			name: "Error/SessionsDao",

			daoMocks: []daoMock{{purged: 2}},
			sessionsMocks: []daoMock{
				{purged: core.RefreshTokenRevocationPurgeBatch},
				{err: errFoo},
			},

			expect: &core.RefreshTokenRevocationPurgeResult{
				Purged:         2,
				SessionsPurged: core.RefreshTokenRevocationPurgeBatch,
			},
			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mockDao := coremocks.NewMockRefreshTokenRevocationPurgeDao(t)
			request := &dao.RefreshTokenRevocationPurgeRequest{Limit: core.RefreshTokenRevocationPurgeBatch}

			for _, daoMock := range testCase.daoMocks {
				var resp *dao.RefreshTokenRevocationPurgeResult
				if daoMock.err == nil {
					resp = &dao.RefreshTokenRevocationPurgeResult{Purged: daoMock.purged}
				}

				mockDao.EXPECT().Exec(mock.Anything, request).Return(resp, daoMock.err).Once()
			}

			mockDaoSessions := coremocks.NewMockRefreshTokenRevocationPurgeDaoSessions(t)
			// The sessions that ended the retention ago go.
			sessionsRequest := mock.MatchedBy(func(request *dao.SessionPurgeRequest) bool {
				cutoff := time.Now().Add(-core.SessionRetention)

				return request.Limit == core.RefreshTokenRevocationPurgeBatch &&
					cutoff.Sub(request.EndedBefore).Abs() < time.Minute
			})

			for _, daoMock := range testCase.sessionsMocks {
				var resp *dao.SessionPurgeResult
				if daoMock.err == nil {
					resp = &dao.SessionPurgeResult{Purged: daoMock.purged}
				}

				mockDaoSessions.EXPECT().Exec(mock.Anything, sessionsRequest).Return(resp, daoMock.err).Once()
			}

			service := core.NewRefreshTokenRevocationPurge(mockDao, mockDaoSessions)

			res, err := service.Exec(t.Context())
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, res)

			mockDao.AssertExpectations(t)
			mockDaoSessions.AssertExpectations(t)
		})
	}
}
