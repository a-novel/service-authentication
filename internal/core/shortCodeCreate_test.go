package core_test

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

	"github.com/a-novel/service-authentication/v2/internal/config"
	"github.com/a-novel/service-authentication/v2/internal/core"
	coremocks "github.com/a-novel/service-authentication/v2/internal/core/mocks"
	"github.com/a-novel/service-authentication/v2/internal/dao"
	"github.com/a-novel/service-authentication/v2/internal/lib"
)

func TestShortCodeCreate(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	type daoMock struct {
		resp *dao.ShortCode
		err  error
	}

	testCases := []struct {
		name string

		request *core.ShortCodeCreateRequest

		daoMock *daoMock

		expectErr error
	}{
		{
			name: "Success",

			request: &core.ShortCodeCreateRequest{
				Usage:    core.ShortCodeUsageValidateEmail,
				Target:   "test-target",
				Data:     map[string]string{"test": "data"},
				TTL:      time.Hour,
				Override: true,
			},

			daoMock: &daoMock{
				resp: &dao.ShortCode{
					ID:     uuid.New(),
					Code:   "encrypted-code",
					Usage:  core.ShortCodeUsageValidateEmail,
					Target: "test-target",
					Data:   []byte(`{"test":"data"}`),
				},
			},
		},
		{
			name: "CreateShortCodeError",

			request: &core.ShortCodeCreateRequest{
				Usage:    core.ShortCodeUsageValidateEmail,
				Target:   "test-target",
				Data:     map[string]string{"test": "data"},
				TTL:      time.Hour,
				Override: true,
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

			mockDao := coremocks.NewMockShortCodeCreateDao(t)

			var (
				encryptedShortCode string
				stampedNow         time.Time
				stampedExpiresAt   time.Time
			)

			if testCase.daoMock != nil {
				mockDao.EXPECT().
					Exec(mock.Anything, mock.MatchedBy(func(data *dao.ShortCodeInsertRequest) bool {
						var dataMap map[string]string

						err := json.Unmarshal(data.Data, &dataMap)

						return assert.NoError(t, err) &&
							assert.NotEqual(t, uuid.Nil, data.ID) &&
							assert.NotEmpty(t, data.Code) &&
							assert.Equal(t, testCase.request.Usage, data.Usage) &&
							assert.Equal(t, testCase.request.Target, data.Target) &&
							assert.Equal(t, testCase.request.Data, dataMap) &&
							assert.Equal(t, testCase.request.Override, data.Override)
					})).
					RunAndReturn(func(_ context.Context, data *dao.ShortCodeInsertRequest) (*dao.ShortCode, error) {
						encryptedShortCode = data.Code
						stampedNow = data.Now
						stampedExpiresAt = data.ExpiresAt

						return testCase.daoMock.resp, testCase.daoMock.err
					})
			}

			service := core.NewShortCodeCreate(mockDao, config.ShortCodesPresetDefault)

			// Bracket the call rather than compare against a clock read somewhere else.
			// The service stamps its timestamp before hashing the code with Argon2id,
			// which is deliberately expensive, so any two reads taken either side of that
			// differ by however long the hash took — over a second on a loaded machine.
			// A timestamp taken during the call is between these two by construction, at
			// any speed.
			before := time.Now()

			resp, err := service.Exec(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)

			after := time.Now()

			if testCase.expectErr == nil {
				require.NotNil(t, resp)
				require.NotEmpty(t, encryptedShortCode)

				assert.WithinRange(t, stampedNow, before, after)
				// Exact, not approximate: the service reads the clock once and derives the
				// expiry from that read, so anything else means it read it twice.
				assert.Equal(t, stampedNow.Add(testCase.request.TTL), stampedExpiresAt)

				require.Equal(t, testCase.request.Usage, resp.Usage)
				require.Equal(t, testCase.request.Target, resp.Target)

				require.NoError(t, lib.CompareArgon2(resp.PlainCode, encryptedShortCode))
			}

			mockDao.AssertExpectations(t)
		})
	}
}
