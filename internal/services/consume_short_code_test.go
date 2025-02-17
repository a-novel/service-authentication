package services_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/a-novel/authentication/internal/dao"
	"github.com/a-novel/authentication/internal/lib"
	"github.com/a-novel/authentication/internal/services"
	servicesmocks "github.com/a-novel/authentication/internal/services/mocks"
	"github.com/a-novel/authentication/models"
)

func TestConsumeShortCode(t *testing.T) {
	t.Parallel()

	errFoo := errors.New("foo")

	shortCode := "test-code"
	encrypted, err := lib.GenerateScrypt(shortCode, lib.ScryptParamsDefault)
	require.NoError(t, err)

	type selectShortCodeData struct {
		resp *dao.ShortCodeEntity
		err  error
	}

	type deleteShortCodeData struct {
		err error
	}

	testCases := []struct {
		name string

		request services.ConsumeShortCodeRequest

		selectShortCodeData *selectShortCodeData
		deleteShortCodeData *deleteShortCodeData

		expect    *models.ShortCode
		expectErr error
	}{
		{
			name: "Success",

			request: services.ConsumeShortCodeRequest{
				Target: "test-target",
				Usage:  "test-usage",
				Code:   shortCode,
			},

			selectShortCodeData: &selectShortCodeData{
				resp: &dao.ShortCodeEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			deleteShortCodeData: &deleteShortCodeData{},

			expect: &models.ShortCode{
				ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Usage:     "test-usage",
				Target:    "test-target",
				Data:      []byte("test-data"),
				CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
				ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				PlainCode: shortCode,
			},
		},
		{
			name: "WrongCode",

			request: services.ConsumeShortCodeRequest{
				Target: "test-target",
				Usage:  "test-usage",
				Code:   "fake-code",
			},

			selectShortCodeData: &selectShortCodeData{
				resp: &dao.ShortCodeEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectErr: services.ErrInvalidShortCode,
		},
		{
			name: "NoCode",

			request: services.ConsumeShortCodeRequest{
				Target: "test-target",
				Usage:  "test-usage",
			},

			selectShortCodeData: &selectShortCodeData{
				resp: &dao.ShortCodeEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			expectErr: services.ErrInvalidShortCode,
		},
		{
			name: "SelectError",

			request: services.ConsumeShortCodeRequest{
				Target: "test-target",
				Usage:  "test-usage",
				Code:   shortCode,
			},

			selectShortCodeData: &selectShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
		{
			name: "DeleteError",

			request: services.ConsumeShortCodeRequest{
				Target: "test-target",
				Usage:  "test-usage",
				Code:   shortCode,
			},

			selectShortCodeData: &selectShortCodeData{
				resp: &dao.ShortCodeEntity{
					ID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Code:      encrypted,
					Usage:     "test-usage",
					Target:    "test-target",
					Data:      []byte("test-data"),
					CreatedAt: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					ExpiresAt: time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},

			deleteShortCodeData: &deleteShortCodeData{
				err: errFoo,
			},

			expectErr: errFoo,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			source := servicesmocks.NewMockConsumeShortCodeSource(t)

			if testCase.selectShortCodeData != nil {
				source.
					On("SelectShortCodeByParams", t.Context(), dao.SelectShortCodeByParamsData{
						Target: testCase.request.Target,
						Usage:  testCase.request.Usage,
					}).
					Return(testCase.selectShortCodeData.resp, testCase.selectShortCodeData.err)
			}

			if testCase.deleteShortCodeData != nil {
				source.
					On("DeleteShortCode", t.Context(), mock.AnythingOfType("dao.DeleteShortCodeData")).
					Return(nil, testCase.deleteShortCodeData.err)
			}

			service := services.NewConsumeShortCodeService(source)

			resp, err := service.ConsumeShortCode(t.Context(), testCase.request)
			require.ErrorIs(t, err, testCase.expectErr)
			require.Equal(t, testCase.expect, resp)

			source.AssertExpectations(t)
		})
	}
}
