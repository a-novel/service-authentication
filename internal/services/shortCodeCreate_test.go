package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/service-authentication/internal/config"
	"github.com/a-novel/service-authentication/internal/dao"
	"github.com/a-novel/service-authentication/internal/lib"
	"github.com/a-novel/service-authentication/internal/services"
	servicesmocks "github.com/a-novel/service-authentication/internal/services/mocks"
)

func TestShortCodeCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type repositoryMock struct {
		resp *dao.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *services.ShortCodeCreateRequest

		repositoryMock *repositoryMock

		expectErr error
	}{
		{
			name: "Success",

			request: &services.ShortCodeCreateRequest{
				Usage:    services.ShortCodeUsageValidateEmail,
				Target:   "test-target",
				Data:     map[string]string{"test": "data"},
				TTL:      time.Hour,
				Override: true,
			},

			repositoryMock: &repositoryMock{
				resp: &dao.ShortCode{
					ID:     uuid.New(),
					Code:   "encrypted-code",
					Usage:  services.ShortCodeUsageValidateEmail,
					Target: "test-target",
					Data:   []byte(`{"test":"data"}`),
				},
			},
		},
		{
			name: "CreateShortCodeError",

			request: &services.ShortCodeCreateRequest{
				Usage:    services.ShortCodeUsageValidateEmail,
				Target:   "test-target",
				Data:     map[string]string{"test": "data"},
				TTL:      time.Hour,
				Override: true,
			},

			repositoryMock: &repositoryMock{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repository := servicesmocks.NewMockShortCodeCreateRepository(t)

			var encryptedShortCode string

			if testCase.repositoryMock != nil {
				repository.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(data *dao.ShortCodeInsertRequest) bool {
						now := time.Now()

						var dataMap map[string]string

						err := json.Unmarshal(data.Data, &dataMap)

						return assert.NoError(t, err) &&
							assert.NotEqual(t, uuid.Nil, data.ID) &&
							assert.NotEmpty(t, data.Code) &&
							assert.Equal(t, testCase.request.Usage, data.Usage) &&
							assert.Equal(t, testCase.request.Target, data.Target) &&
							assert.Equal(t, testCase.request.Data, dataMap) &&
							assert.WithinDuration(t, now, data.Now, time.Second) &&
							assert.WithinDuration(t, now.Add(testCase.request.TTL), data.ExpiresAt, time.Second) &&
							assert.Equal(t, testCase.request.Override, data.Override)
					})).
					RunAndReturn(func(_ context.Context, data *dao.ShortCodeInsertRequest) (*dao.ShortCode, error) {
						encryptedShortCode = data.Code

						return testCase.repositoryMock.resp, testCase.repositoryMock.err
					})
			}

			service := services.NewShortCodeCreate(repository, config.ShortCodesPresetDefault)

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			if testCase.expectErr == nil {
				require.NotNil(t, resp)
				require.NotEmpty(t, encryptedShortCode)

				require.Equal(t, testCase.request.Usage, resp.Usage)
				require.Equal(t, testCase.request.Target, resp.Target)

				require.NoError(t, lib.CompareScrypt(resp.PlainCode, encryptedShortCode))
			}

			repository.AssertExpectations(t)
		})
	}
}
