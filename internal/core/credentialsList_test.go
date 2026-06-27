package core_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
)

func TestCredentialsList(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp []*dao.Credentials
		err  error
	}

	testCases := []struct {
		name string

		request *core.CredentialsListRequest

		daoMock *daoMock

		expect    []*core.Credentials
		expectErr error
	}{
		{
			name: "Success",

			request: &core.CredentialsListRequest{
				Limit:  10,
				Offset: 0,
				Roles:  []string{config.RoleUser},
			},

			daoMock: &daoMock{
				resp: []*dao.Credentials{
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						Email:     "user3@email.com",
						Role:      config.RoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						Email:     "user2@email.com",
						Role:      config.RoleAdmin,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					{
						ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						Email:     "user1@email.com",
						Role:      config.RoleUser,
						CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
						UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},

			expect: []*core.Credentials{
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					Email:     "user3@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Email:     "user2@email.com",
					Role:      config.RoleAdmin,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
				{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Email:     "user1@email.com",
					Role:      config.RoleUser,
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					UpdatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
		},
		{
			name: "Error",

			request: &core.CredentialsListRequest{
				Limit:  10,
				Offset: 0,
				Roles:  []string{config.RoleUser},
			},

			daoMock: &daoMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			mockDao := coremocks.NewMockCredentialsListDao(t)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, &dao.CredentialsListRequest{
						Limit:  testCase.request.Limit,
						Offset: testCase.request.Offset,
						Roles:  testCase.request.Roles,
					}).
					Return(testCase.daoMock.resp, testCase.daoMock.err)
			}

			service := core.NewCredentialsList(mockDao)

			resp, err := service.Exec(ctx, testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			mockDao.AssertExpectations(t)
		})
	}
}
